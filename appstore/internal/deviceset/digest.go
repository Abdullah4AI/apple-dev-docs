// Package deviceset provides the canonical, privacy-preserving identity of an
// explicit Apple device set shared by signing and artifact inspection.
package deviceset

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
	"strings"
)

// Result binds a canonical set's unique normalized count and SHA-256 digest.
type Result struct {
	Count  int
	SHA256 string
}

// Normalize removes accepted visual separators and canonicalizes case.
func Normalize(value string) string {
	replacer := strings.NewReplacer("-", "", ":", "")
	return strings.ToUpper(replacer.Replace(strings.TrimSpace(value)))
}

// NormalizeUnique returns sorted unique non-empty canonical identifiers.
func NormalizeUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := Normalize(value); normalized != "" {
			set[normalized] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// Digest returns a domain-separated, count- and length-framed SHA-256 over the
// sorted unique normalized identifiers. Empty sets retain an empty digest.
func Digest(values []string) Result {
	canonical := NormalizeUnique(values)
	result := Result{Count: len(canonical)}
	if len(canonical) == 0 {
		return result
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("asc-device-set-v1\x00"))
	writeFrameLength(hash, uint64(len(canonical)))
	for _, value := range canonical {
		writeFrameLength(hash, uint64(len(value)))
		_, _ = hash.Write([]byte(value))
	}
	result.SHA256 = hex.EncodeToString(hash.Sum(nil))
	return result
}

type digestWriter interface {
	Write([]byte) (int, error)
}

func writeFrameLength(writer digestWriter, value uint64) {
	var frame [8]byte
	binary.BigEndian.PutUint64(frame[:], value)
	_, _ = writer.Write(frame[:])
}
