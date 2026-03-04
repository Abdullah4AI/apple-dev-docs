package web

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// ECIESEncrypt encrypts a plaintext value using the ECIES scheme used by
// the App Store Connect Xcode Cloud UI for secret environment variables.
//
// Algorithm (reverse-engineered from ASC web UI JS):
//  1. Decode server P-256 public key (64 bytes raw x||y), prepend 0x04
//  2. Generate ephemeral ECDH P-256 key pair
//  3. ECDH key agreement → 32-byte shared secret
//  4. HKDF-SHA256(key=shared_secret, salt=random_32, info="") → AES-256 key
//  5. AES-256-GCM(key, iv=random_12, plaintext) → ciphertext + 16-byte tag
//  6. Output = salt(32) || ephemeral_pub_no_prefix(64) || iv(12) || ciphertext_with_tag
//  7. Base64 encode
func ECIESEncrypt(serverKeyB64 string, plaintext string) (string, error) {
	// 1. Decode server public key (64 bytes raw) and build uncompressed point
	serverKeyRaw, err := base64.StdEncoding.DecodeString(serverKeyB64)
	if err != nil {
		return "", fmt.Errorf("decode server key: %w", err)
	}
	if len(serverKeyRaw) != 64 {
		return "", fmt.Errorf("expected 64-byte server key, got %d", len(serverKeyRaw))
	}
	// Prepend 0x04 for uncompressed EC point format
	uncompressed := make([]byte, 65)
	uncompressed[0] = 0x04
	copy(uncompressed[1:], serverKeyRaw)

	serverPub, err := ecdh.P256().NewPublicKey(uncompressed)
	if err != nil {
		return "", fmt.Errorf("import server public key: %w", err)
	}

	// 2. Generate ephemeral ECDH P-256 key pair
	ephPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate ephemeral key: %w", err)
	}

	// 3. ECDH key agreement → shared secret (raw x-coordinate, 32 bytes)
	sharedSecret, err := ephPriv.ECDH(serverPub)
	if err != nil {
		return "", fmt.Errorf("ecdh key agreement: %w", err)
	}

	// 4. HKDF-SHA256 to derive final AES-256 key
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	hkdfReader := hkdf.New(sha256.New, sharedSecret, salt, []byte(""))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, aesKey); err != nil {
		return "", fmt.Errorf("hkdf derive key: %w", err)
	}

	// 5. AES-256-GCM encrypt
	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("generate iv: %w", err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("create aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	ciphertext := gcm.Seal(nil, iv, []byte(plaintext), nil)

	// 6. Concatenate: salt(32) || ephemeral_pub_no_prefix(64) || iv(12) || ciphertext+tag
	ephPubRaw := ephPriv.PublicKey().Bytes() // 65 bytes with 0x04 prefix
	ephPubNoPrefix := ephPubRaw[1:]          // 64 bytes

	output := make([]byte, 0, 32+64+12+len(ciphertext))
	output = append(output, salt...)
	output = append(output, ephPubNoPrefix...)
	output = append(output, iv...)
	output = append(output, ciphertext...)

	// 7. Base64 encode
	return base64.StdEncoding.EncodeToString(output), nil
}
