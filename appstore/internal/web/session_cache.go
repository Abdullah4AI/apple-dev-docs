package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/99designs/keyring"
)

const (
	webSessionCacheEnabledEnv = "ASC_WEB_SESSION_CACHE"
	webSessionCacheDirEnv     = "ASC_WEB_SESSION_CACHE_DIR"
	webSessionBackendEnv      = "ASC_WEB_SESSION_CACHE_BACKEND"

	webSessionCacheVersion = 1

	webSessionKeyringService = "asc-web-session"
	webSessionStoreItem      = "asc:web-session:store"
	webSessionKeyPrefix      = "asc:web-session:"
	webSessionLastKeyItem    = "asc:web-session:last"
)

var (
	ErrCachedSessionExpired          = errors.New("cached web session expired")
	ErrCachedSessionValidationFailed = errors.New("cached web session could not be validated")
	errMalformedSessionFile          = errors.New("web session cache is malformed")
	// errMalformedSessionStore identifies malformed aggregate keychain data.
	// It is separate from the file-cache sentinel so an explicit keychain
	// recovery cannot be triggered by an unrelated file-read error.
	errMalformedSessionStore = errors.New("web session store is malformed")
)

type sessionBackend int

const (
	sessionBackendOff sessionBackend = iota
	sessionBackendKeychain
	sessionBackendFile
)

// sessionEntryOrigin records which backend a cached entry was actually read
// from. A conditional delete needs it: the entry whose stamp matched is the
// only one proven stale, and the other backend may hold a newer session.
type sessionEntryOrigin int

const (
	sessionEntryOriginNone sessionEntryOrigin = iota
	sessionEntryOriginKeychain
	sessionEntryOriginFile
)

type backendSelection struct {
	backend          sessionBackend
	fallbackFile     bool
	fallbackKeychain bool
}

type persistedSession struct {
	Version         int                  `json:"version"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Generation      string               `json:"generation,omitempty"`
	UserEmail       string               `json:"user_email,omitempty"`
	DeveloperTeamID string               `json:"developer_team_id,omitempty"`
	Cookies         map[string][]pCookie `json:"cookies"`
}

type persistedSessionStore struct {
	Version  int                         `json:"version"`
	LastKey  string                      `json:"last_key,omitempty"`
	Sessions map[string]persistedSession `json:"sessions,omitempty"`
}

type pCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Path     string    `json:"path,omitempty"`
	Domain   string    `json:"domain,omitempty"`
	Expires  time.Time `json:"expires,omitempty"`
	MaxAge   int       `json:"max_age,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HttpOnly bool      `json:"http_only,omitempty"`
	SameSite int       `json:"same_site,omitempty"`
}

type persistedLastSession struct {
	Version int    `json:"version"`
	Key     string `json:"key"`
}

var (
	sessionKeyringOpen = func() (keyring.Keyring, error) {
		return keyring.Open(keyring.Config{
			ServiceName:                    webSessionKeyringService,
			KeychainTrustApplication:       true,
			KeychainSynchronizable:         false,
			KeychainAccessibleWhenUnlocked: true,
			AllowedBackends: []keyring.BackendType{
				keyring.KeychainBackend,
				keyring.WinCredBackend,
				keyring.SecretServiceBackend,
				keyring.KWalletBackend,
				keyring.KeyCtlBackend,
			},
		})
	}
	sessionFileWrite   = os.WriteFile
	sessionInfoFetcher = getSessionInfo

	// sessionCompareDeleteBarrier runs between the stamp comparison and the
	// delete in DeleteSessionIfMatches. Tests set it to schedule a concurrent
	// persist inside that window; it is nil in production.
	sessionCompareDeleteBarrier func()
	sessionGenerationReader     = func(b []byte) (int, error) { return rand.Read(b) }
)

func webSessionCacheEnabled() bool {
	raw := strings.TrimSpace(os.Getenv(webSessionCacheEnabledEnv))
	if raw == "" {
		return true
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func resolveBackendSelection() backendSelection {
	if !webSessionCacheEnabled() {
		return backendSelection{backend: sessionBackendOff}
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(webSessionBackendEnv))) {
	case "off", "none", "disabled":
		return backendSelection{backend: sessionBackendOff}
	case "file":
		return backendSelection{backend: sessionBackendFile}
	case "keychain":
		// Allow explicit keychain mode to import sessions from the file cache
		// so users can switch back after running on the default file-backed mode.
		return backendSelection{backend: sessionBackendKeychain, fallbackFile: true}
	case "", "auto":
		// Default to file-backed web sessions so successful logins can be reused
		// without recurring per-binary keychain approval prompts.
		return backendSelection{backend: sessionBackendFile, fallbackKeychain: true}
	default:
		return backendSelection{backend: sessionBackendFile, fallbackKeychain: true}
	}
}

func webSessionCacheDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(webSessionCacheDirEnv)); custom != "" {
		return custom, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".asc", "web"), nil
}

func webSessionCacheKey(username string) string {
	normalized := strings.ToLower(strings.TrimSpace(username))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func webSessionFilePath(key string) (string, error) {
	dir, err := webSessionCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session-"+key+".json"), nil
}

func webSessionLastFilePath() (string, error) {
	dir, err := webSessionCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "last.json"), nil
}

func sessionCookieURLs() []*url.URL {
	return []*url.URL{
		{Scheme: "https", Host: "appstoreconnect.apple.com", Path: "/"},
		{Scheme: "https", Host: "developer.apple.com", Path: "/"},
		{Scheme: "https", Host: "idmsa.apple.com", Path: "/"},
		{Scheme: "https", Host: "gsa.apple.com", Path: "/"},
	}
}

func isExpiredCookie(c pCookie, now time.Time) bool {
	if c.MaxAge < 0 {
		return true
	}
	if !c.Expires.IsZero() && c.Expires.Before(now) {
		return true
	}
	return false
}

func serializeCookieJar(jar http.CookieJar, userEmail string) persistedSession {
	sess, _ := serializeCookieJarWithError(jar, userEmail)
	return sess
}

func serializeCookieJarWithError(jar http.CookieJar, userEmail string) (persistedSession, error) {
	now := time.Now().UTC()
	var generation [16]byte
	if _, err := sessionGenerationReader(generation[:]); err != nil {
		return persistedSession{}, fmt.Errorf("generate session cache identity: %w", err)
	}
	out := persistedSession{
		Version:    webSessionCacheVersion,
		UpdatedAt:  now,
		Generation: fmt.Sprintf("%x", generation[:]),
		UserEmail:  strings.TrimSpace(userEmail),
		Cookies:    map[string][]pCookie{},
	}
	for _, u := range sessionCookieURLs() {
		cookies := jar.Cookies(u)
		if len(cookies) == 0 {
			continue
		}
		list := make([]pCookie, 0, len(cookies))
		for _, c := range cookies {
			if c == nil || c.Name == "" {
				continue
			}
			pc := pCookie{
				Name:     c.Name,
				Value:    c.Value,
				Path:     c.Path,
				Domain:   c.Domain,
				Expires:  c.Expires,
				MaxAge:   c.MaxAge,
				Secure:   c.Secure,
				HttpOnly: c.HttpOnly,
				SameSite: int(c.SameSite),
			}
			if isExpiredCookie(pc, now) {
				continue
			}
			list = append(list, pc)
		}
		if len(list) > 0 {
			out.Cookies[u.String()] = list
		}
	}
	return out, nil
}

func hydrateCookieJar(jar http.CookieJar, sess persistedSession) int {
	now := time.Now().UTC()
	loaded := 0
	for base, list := range sess.Cookies {
		u, err := url.Parse(base)
		if err != nil {
			continue
		}
		cookies := make([]*http.Cookie, 0, len(list))
		for _, pc := range list {
			if pc.Name == "" || isExpiredCookie(pc, now) {
				continue
			}
			cookies = append(cookies, &http.Cookie{
				Name:     pc.Name,
				Value:    pc.Value,
				Path:     pc.Path,
				Domain:   pc.Domain,
				Expires:  pc.Expires,
				MaxAge:   pc.MaxAge,
				Secure:   pc.Secure,
				HttpOnly: pc.HttpOnly,
				SameSite: http.SameSite(pc.SameSite),
			})
		}
		if len(cookies) > 0 {
			jar.SetCookies(u, cookies)
			loaded += len(cookies)
		}
	}
	return loaded
}

func keyringSessionItem(key string) string {
	return webSessionKeyPrefix + key
}

func isKeyringUnavailable(err error) bool {
	return errors.Is(err, keyring.ErrNoAvailImpl)
}

func newPersistedSessionStore() persistedSessionStore {
	return persistedSessionStore{
		Version:  webSessionCacheVersion,
		Sessions: map[string]persistedSession{},
	}
}

func normalizePersistedSessionStore(store persistedSessionStore) persistedSessionStore {
	if store.Version == 0 {
		store.Version = webSessionCacheVersion
	}
	if store.Sessions == nil {
		store.Sessions = map[string]persistedSession{}
	}
	return store
}

func resolvePersistedSessionStoreLastKey(store persistedSessionStore) (string, bool) {
	store = normalizePersistedSessionStore(store)
	if key := strings.TrimSpace(store.LastKey); key != "" {
		if _, ok := store.Sessions[key]; ok {
			return key, true
		}
	}
	if len(store.Sessions) == 0 {
		return "", false
	}
	keys := make([]string, 0, len(store.Sessions))
	for key := range store.Sessions {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return "", false
	}
	sort.Slice(keys, func(i, j int) bool {
		left := store.Sessions[keys[i]].UpdatedAt
		right := store.Sessions[keys[j]].UpdatedAt
		if left.Equal(right) {
			return keys[i] < keys[j]
		}
		return left.After(right)
	})
	return keys[0], true
}

func readLegacySessionFromKeyring(kr keyring.Keyring, key string) (persistedSession, bool, error) {
	item, err := kr.Get(keyringSessionItem(key))
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return persistedSession{}, false, nil
		}
		return persistedSession{}, false, err
	}
	var sess persistedSession
	if err := json.Unmarshal(item.Data, &sess); err != nil {
		return persistedSession{}, false, fmt.Errorf("failed to decode keychain session: %w", err)
	}
	if sess.Version != webSessionCacheVersion {
		return persistedSession{}, false, nil
	}
	return sess, true, nil
}

func readLegacyLastKeyFromKeyring(kr keyring.Keyring) (string, bool, error) {
	item, err := kr.Get(webSessionLastKeyItem)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	var last persistedLastSession
	if err := json.Unmarshal(item.Data, &last); err != nil {
		return "", false, err
	}
	if last.Version != webSessionCacheVersion || strings.TrimSpace(last.Key) == "" {
		return "", false, nil
	}
	return strings.TrimSpace(last.Key), true, nil
}

func readLegacySessionStoreFromKeyring(kr keyring.Keyring) (persistedSessionStore, bool, error) {
	keys, err := kr.Keys()
	if err != nil {
		return persistedSessionStore{}, false, err
	}
	store := newPersistedSessionStore()
	for _, itemKey := range keys {
		if !strings.HasPrefix(itemKey, webSessionKeyPrefix) || itemKey == webSessionLastKeyItem || itemKey == webSessionStoreItem {
			continue
		}
		key := strings.TrimPrefix(itemKey, webSessionKeyPrefix)
		sess, ok, err := readLegacySessionFromKeyring(kr, key)
		if err != nil {
			return persistedSessionStore{}, false, err
		}
		if ok {
			store.Sessions[key] = sess
		}
	}
	if len(store.Sessions) == 0 {
		return persistedSessionStore{}, false, nil
	}
	if lastKey, ok, err := readLegacyLastKeyFromKeyring(kr); err != nil {
		return persistedSessionStore{}, false, err
	} else if ok {
		store.LastKey = lastKey
	}
	if resolved, ok := resolvePersistedSessionStoreLastKey(store); ok {
		store.LastKey = resolved
	}
	return store, true, nil
}

func readSessionStoreFromKeyring(kr keyring.Keyring) (persistedSessionStore, bool, error) {
	item, err := kr.Get(webSessionStoreItem)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return readLegacySessionStoreFromKeyring(kr)
		}
		return persistedSessionStore{}, false, err
	}
	var store persistedSessionStore
	if err := json.Unmarshal(item.Data, &store); err != nil {
		return persistedSessionStore{}, false, fmt.Errorf("%w: failed to decode keychain session store: %w", errMalformedSessionStore, err)
	}
	if store.Version != webSessionCacheVersion {
		return persistedSessionStore{}, false, nil
	}
	store = normalizePersistedSessionStore(store)
	if resolved, ok := resolvePersistedSessionStoreLastKey(store); ok {
		store.LastKey = resolved
	}
	return store, true, nil
}

func writeSessionStoreToKeyring(kr keyring.Keyring, store persistedSessionStore) error {
	store = normalizePersistedSessionStore(store)
	raw, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("failed to marshal session store: %w", err)
	}
	return kr.Set(keyring.Item{
		Key:   webSessionStoreItem,
		Data:  raw,
		Label: "ASC Web Session Store",
	})
}

func removeSessionStoreFromKeyring(kr keyring.Keyring) error {
	err := kr.Remove(webSessionStoreItem)
	if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return err
	}
	return nil
}

func removeLegacySessionFromKeyring(kr keyring.Keyring, key string) error {
	err := kr.Remove(keyringSessionItem(key))
	if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return err
	}
	return nil
}

func removeLegacyLastKeyFromKeyring(kr keyring.Keyring) error {
	err := kr.Remove(webSessionLastKeyItem)
	if err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return err
	}
	return nil
}

func writeSessionToKeychain(key string, sess persistedSession) error {
	return withSessionStoreLock(func() error { return writeSessionToKeychainUnlocked(key, sess) })
}

func writeSessionToKeychainUnlocked(key string, sess persistedSession) error {
	kr, err := sessionKeyringOpen()
	if err != nil {
		return err
	}
	store, ok, err := readSessionStoreFromKeyring(kr)
	if err != nil {
		return err
	}
	if !ok {
		store = newPersistedSessionStore()
	}
	store = normalizePersistedSessionStore(store)
	store.Sessions[key] = sess
	store.LastKey = key
	return writeSessionStoreToKeyring(kr, store)
}

func writeSessionToKeychainIfAbsentUnlocked(key string, sess persistedSession) error {
	kr, err := sessionKeyringOpen()
	if err != nil {
		return err
	}
	store, ok, err := readSessionStoreFromKeyring(kr)
	if err != nil {
		// Never treat malformed or unreadable state as absent for a create-only
		// import. Doing so could destroy another account's credentials.
		return err
	}
	if ok {
		if _, exists := store.Sessions[key]; exists {
			return cachedSessionAlreadyExistsError(key)
		}
	} else {
		store = newPersistedSessionStore()
	}
	store = normalizePersistedSessionStore(store)
	store.Sessions[key] = sess
	store.LastKey = key
	return writeSessionStoreToKeyring(kr, store)
}

func writeSessionToKeychainWithRecoveryUnlocked(key string, sess persistedSession, recoverMalformed bool) error {
	kr, err := sessionKeyringOpen()
	if err != nil {
		return err
	}
	store, ok, err := readSessionStoreFromKeyring(kr)
	if err != nil {
		if !recoverMalformed || !errors.Is(err, errMalformedSessionStore) {
			return err
		}
		store = newPersistedSessionStore()
		ok = true
	}
	if !ok {
		store = newPersistedSessionStore()
	}
	store = normalizePersistedSessionStore(store)
	store.Sessions[key] = sess
	store.LastKey = key
	return writeSessionStoreToKeyring(kr, store)
}

func keychainSessionEntryCollisionUnlocked(key string) error {
	kr, err := sessionKeyringOpen()
	if err != nil {
		return err
	}
	store, ok, err := readSessionStoreFromKeyring(kr)
	if err != nil {
		return err
	}
	if ok {
		if _, exists := store.Sessions[key]; exists {
			return cachedSessionAlreadyExistsError(key)
		}
	}
	return nil
}

func readSessionFromKeychain(key string) (persistedSession, bool, error) {
	kr, err := sessionKeyringOpen()
	if err != nil {
		return persistedSession{}, false, err
	}
	store, ok, err := readSessionStoreFromKeyring(kr)
	if err != nil || !ok {
		return persistedSession{}, false, err
	}
	sess, ok := store.Sessions[key]
	if !ok {
		return persistedSession{}, false, nil
	}
	return sess, true, nil
}

type sessionFileBackup struct {
	exists bool
	data   []byte
	mode   os.FileMode
}

type sessionFileState struct {
	sessionPath string
	lastPath    string
	session     sessionFileBackup
	last        sessionFileBackup
}

func backupSessionFile(path string) (sessionFileBackup, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return sessionFileBackup{}, nil
		}
		return sessionFileBackup{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return sessionFileBackup{}, err
	}
	return sessionFileBackup{exists: true, data: data, mode: info.Mode().Perm()}, nil
}

func restoreSessionFile(path string, backup sessionFileBackup) error {
	if !backup.exists {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".asc-web-session-rollback-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(backup.mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(backup.data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// captureFileSessionState snapshots both files that make up a file-backed
// session. The last-session pointer is part of the cache state: restoring only
// the account file can leave a failed overwrite selecting a different session.
func captureFileSessionState(key string) (sessionFileState, error) {
	sessionPath, err := webSessionFilePath(key)
	if err != nil {
		return sessionFileState{}, err
	}
	lastPath, err := webSessionLastFilePath()
	if err != nil {
		return sessionFileState{}, err
	}
	sessionBackup, err := backupSessionFile(sessionPath)
	if err != nil {
		return sessionFileState{}, fmt.Errorf("failed to back up session cache: %w", err)
	}
	lastBackup, err := backupSessionFile(lastPath)
	if err != nil {
		return sessionFileState{}, fmt.Errorf("failed to back up last session pointer: %w", err)
	}
	return sessionFileState{
		sessionPath: sessionPath,
		lastPath:    lastPath,
		session:     sessionBackup,
		last:        lastBackup,
	}, nil
}

func (state sessionFileState) restore() error {
	return errors.Join(
		restoreSessionFile(state.sessionPath, state.session),
		restoreSessionFile(state.lastPath, state.last),
	)
}

func writeSessionToFile(key string, sess persistedSession) error {
	dir, err := webSessionCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create session cache dir: %w", err)
	}

	raw, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	state, err := captureFileSessionState(key)
	if err != nil {
		return err
	}
	rollback := func(cause error) error {
		if rollbackErr := state.restore(); rollbackErr != nil {
			return errors.Join(cause, fmt.Errorf("failed to roll back session cache: %w", rollbackErr))
		}
		return cause
	}

	tmpSessionPath := state.sessionPath + ".tmp"
	if err := sessionFileWrite(tmpSessionPath, raw, 0o600); err != nil {
		_ = os.Remove(tmpSessionPath)
		return fmt.Errorf("failed to write session cache: %w", err)
	}
	if err := os.Rename(tmpSessionPath, state.sessionPath); err != nil {
		_ = os.Remove(tmpSessionPath)
		return fmt.Errorf("failed to finalize session cache: %w", err)
	}

	lastRaw, err := json.Marshal(persistedLastSession{Version: webSessionCacheVersion, Key: key})
	if err != nil {
		return rollback(fmt.Errorf("failed to marshal last session pointer: %w", err))
	}
	tmpLastPath := state.lastPath + ".tmp"
	if err := sessionFileWrite(tmpLastPath, lastRaw, 0o600); err != nil {
		_ = os.Remove(tmpLastPath)
		return rollback(fmt.Errorf("failed to write last session pointer: %w", err))
	}
	if err := os.Rename(tmpLastPath, state.lastPath); err != nil {
		_ = os.Remove(tmpLastPath)
		return rollback(fmt.Errorf("failed to finalize last session pointer: %w", err))
	}
	return nil
}

// writeSessionToFileIfAbsent creates a file-backed session without replacing
// an entry that appeared after import validation. O_EXCL is the persistence
// boundary: a preceding existence check alone would leave a TOCTOU window.
func writeSessionToFileIfAbsent(key string, sess persistedSession) error {
	dir, err := webSessionCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create session cache dir: %w", err)
	}

	raw, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	state, err := captureFileSessionState(key)
	if err != nil {
		return err
	}
	rollback := func(cause error) error {
		if rollbackErr := state.restore(); rollbackErr != nil {
			return errors.Join(cause, fmt.Errorf("failed to roll back session cache: %w", rollbackErr))
		}
		return cause
	}

	file, err := os.OpenFile(state.sessionPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return cachedSessionAlreadyExistsError(key)
		}
		return fmt.Errorf("failed to create session cache: %w", err)
	}
	if n, writeErr := file.Write(raw); writeErr != nil {
		_ = file.Close()
		return rollback(fmt.Errorf("failed to write session cache: %w", writeErr))
	} else if n != len(raw) {
		_ = file.Close()
		return rollback(fmt.Errorf("failed to write session cache: %w", io.ErrShortWrite))
	}
	if err := file.Close(); err != nil {
		return rollback(fmt.Errorf("failed to finalize session cache: %w", err))
	}

	lastRaw, err := json.Marshal(persistedLastSession{Version: webSessionCacheVersion, Key: key})
	if err != nil {
		return rollback(fmt.Errorf("failed to marshal last session pointer: %w", err))
	}
	tmpLastPath := state.lastPath + ".tmp"
	if err := sessionFileWrite(tmpLastPath, lastRaw, 0o600); err != nil {
		_ = os.Remove(tmpLastPath)
		return rollback(fmt.Errorf("failed to write last session pointer: %w", err))
	}
	if err := os.Rename(tmpLastPath, state.lastPath); err != nil {
		_ = os.Remove(tmpLastPath)
		return rollback(fmt.Errorf("failed to finalize last session pointer: %w", err))
	}
	return nil
}

func cachedSessionAlreadyExistsError(key string) error {
	return fmt.Errorf("cached web session already exists for %s: %w", key, os.ErrExist)
}

// fileSessionEntryCollision reports whether any file artifact already
// occupies the target path. Lstat intentionally counts a malformed file or a
// symlink as occupied: no-overwrite must not guess that either is absent.
func fileSessionEntryCollision(key string) error {
	path, err := webSessionFilePath(key)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to inspect session cache: %w", err)
	}
	return cachedSessionAlreadyExistsError(key)
}

func readSessionFromFile(key string) (persistedSession, bool, error) {
	path, err := webSessionFilePath(key)
	if err != nil {
		return persistedSession{}, false, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return persistedSession{}, false, nil
		}
		return persistedSession{}, false, err
	}
	var sess persistedSession
	if err := json.Unmarshal(raw, &sess); err != nil {
		return persistedSession{}, false, fmt.Errorf("%w: %w", errMalformedSessionFile, err)
	}
	if sess.Version != webSessionCacheVersion {
		return persistedSession{}, false, nil
	}
	return sess, true, nil
}

func readLastKeyFromFile() (string, bool, error) {
	path, err := webSessionLastFilePath()
	if err != nil {
		return "", false, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var last persistedLastSession
	if err := json.Unmarshal(raw, &last); err != nil {
		return "", false, err
	}
	if last.Version != webSessionCacheVersion || strings.TrimSpace(last.Key) == "" {
		return "", false, nil
	}
	return strings.TrimSpace(last.Key), true, nil
}

func persistSessionBySelection(selection backendSelection, key string, sess persistedSession) error {
	if selection.backend == sessionBackendOff {
		return nil
	}
	// Hold the entry lock so a concurrent conditional delete cannot compare a
	// stamp before this write and delete the entry it produces.
	return withSessionEntryLock(key, func() error {
		return persistSessionBySelectionLocked(selection, key, sess)
	})
}

func persistSessionBySelectionLocked(selection backendSelection, key string, sess persistedSession) error {
	switch selection.backend {
	case sessionBackendOff:
		return nil
	case sessionBackendKeychain:
		if err := writeSessionToKeychain(key, sess); err != nil {
			if selection.fallbackFile && isKeyringUnavailable(err) {
				return writeSessionToFile(key, sess)
			}
			return err
		}
		return nil
	case sessionBackendFile:
		return writeSessionToFile(key, sess)
	default:
		return nil
	}
}

type keychainItemState struct {
	key    string
	item   keyring.Item
	exists bool
}

type keychainSessionState struct {
	items    []keychainItemState
	captured bool
}

// captureKeychainSessionState snapshots the aggregate and legacy items that
// can change while replacing one account. Raw item bytes preserve the prior
// last-session choice and malformed-store bytes when a later write fails.
// Callers hold the shared store lock when the keychain is part of a mutation.
func captureKeychainSessionState(key string) (keychainSessionState, error) {
	kr, err := sessionKeyringOpen()
	if err != nil {
		if isKeyringUnavailable(err) {
			return keychainSessionState{}, nil
		}
		return keychainSessionState{}, fmt.Errorf("failed to back up keychain session: %w", err)
	}
	keys := []string{webSessionStoreItem, keyringSessionItem(key), webSessionLastKeyItem}
	state := keychainSessionState{items: make([]keychainItemState, 0, len(keys)), captured: true}
	for _, itemKey := range keys {
		item, err := kr.Get(itemKey)
		if err != nil {
			if errors.Is(err, keyring.ErrKeyNotFound) {
				state.items = append(state.items, keychainItemState{key: itemKey})
				continue
			}
			return keychainSessionState{}, fmt.Errorf("failed to back up keychain item: %w", err)
		}
		state.items = append(state.items, keychainItemState{key: itemKey, item: item, exists: true})
	}
	return state, nil
}

func (state keychainSessionState) restore() error {
	if !state.captured {
		return nil
	}
	kr, err := sessionKeyringOpen()
	if err != nil {
		return fmt.Errorf("failed to restore keychain session: %w", err)
	}
	var restoreErr error
	for _, itemState := range state.items {
		if itemState.exists {
			restoreErr = errors.Join(restoreErr, kr.Set(itemState.item))
			continue
		}
		removeErr := kr.Remove(itemState.key)
		if errors.Is(removeErr, keyring.ErrKeyNotFound) {
			removeErr = nil
		}
		restoreErr = errors.Join(restoreErr, removeErr)
	}
	return restoreErr
}

type importedSessionState struct {
	file     *sessionFileState
	keychain *keychainSessionState
}

func captureImportedSessionState(selection backendSelection, key string) (importedSessionState, error) {
	state := importedSessionState{}
	if selection.backend == sessionBackendFile || selection.fallbackFile {
		fileState, err := captureFileSessionState(key)
		if err != nil {
			return importedSessionState{}, err
		}
		state.file = &fileState
	}
	if selection.backend == sessionBackendKeychain || selection.fallbackKeychain {
		keychainState, err := captureKeychainSessionState(key)
		if err != nil {
			return importedSessionState{}, err
		}
		state.keychain = &keychainState
	}
	return state, nil
}

func (state importedSessionState) restore() error {
	var restoreErr error
	if state.file != nil {
		restoreErr = errors.Join(restoreErr, state.file.restore())
	}
	if state.keychain != nil {
		restoreErr = errors.Join(restoreErr, state.keychain.restore())
	}
	return restoreErr
}

// persistImportedSessionBySelection stores an imported session at the final
// persistence boundary. Explicit overwrite imports snapshot both backends
// before cleanup so a failure after one backend changes restores the previous
// session, mirror, and last-session pointer exactly.
func persistImportedSessionBySelection(selection backendSelection, key string, sess persistedSession, overwrite bool) error {
	if selection.backend == sessionBackendOff {
		return nil
	}
	return withSessionEntryLock(key, func() error {
		// Any selection that can inspect or mutate the aggregate keychain must
		// hold the fail-closed shared lock. This also serializes a keychain
		// collision check with the file O_EXCL create in fallback mode.
		if selection.backend == sessionBackendKeychain || selection.fallbackKeychain {
			return withSessionStoreLock(func() error {
				return persistImportedSessionBySelectionLocked(selection, key, sess, overwrite)
			})
		}
		return persistImportedSessionBySelectionLocked(selection, key, sess, overwrite)
	})
}

func persistImportedSessionBySelectionLocked(selection backendSelection, key string, sess persistedSession, overwrite bool) error {
	switch selection.backend {
	case sessionBackendOff:
		return nil
	case sessionBackendKeychain:
		if !overwrite {
			if selection.fallbackFile {
				if err := fileSessionEntryCollision(key); err != nil {
					return err
				}
			}
			if err := writeSessionToKeychainIfAbsentUnlocked(key, sess); err != nil {
				if selection.fallbackFile && isKeyringUnavailable(err) {
					return writeSessionToFileIfAbsent(key, sess)
				}
				return err
			}
			return nil
		}

		state, err := captureImportedSessionState(selection, key)
		if err != nil {
			return err
		}
		if selection.fallbackFile {
			if err := deleteMirroredSessionFromFile(key); err != nil {
				return errors.Join(err, state.restore())
			}
		}
		if err := writeSessionToKeychainWithRecoveryUnlocked(key, sess, true); err != nil {
			// Fail closed: a stale keychain entry must not remain ahead of a
			// replacement written only to the file fallback.
			return errors.Join(err, state.restore())
		}
		return nil

	case sessionBackendFile:
		if !overwrite {
			if selection.fallbackKeychain {
				err := keychainSessionEntryCollisionUnlocked(key)
				if !isKeyringUnavailable(err) && err != nil {
					return fmt.Errorf("failed to inspect keychain session: %w", err)
				}
			}
			return writeSessionToFileIfAbsent(key, sess)
		}

		state, err := captureImportedSessionState(selection, key)
		if err != nil {
			return err
		}
		if selection.fallbackKeychain {
			if err := deleteSessionFromKeychainWithRecoveryUnlocked(key, true); err != nil && !isKeyringUnavailable(err) {
				return errors.Join(err, state.restore())
			}
		}
		if err := writeSessionToFile(key, sess); err != nil {
			return errors.Join(err, state.restore())
		}
		return nil
	default:
		return nil
	}
}

func readSessionFromFileWithKeychainFallback(key string, fallbackKeychain bool) (persistedSession, bool, error) {
	sess, _, ok, err := readSessionFromFileWithKeychainFallbackOrigin(key, fallbackKeychain)
	return sess, ok, err
}

func readSessionFromFileWithKeychainFallbackOrigin(key string, fallbackKeychain bool) (persistedSession, sessionEntryOrigin, bool, error) {
	sess, ok, err := readSessionFromFile(key)
	if err == nil && (ok || !fallbackKeychain) {
		return sess, sessionEntryOriginWhenFound(sessionEntryOriginFile, ok), ok, nil
	}
	if err != nil && !fallbackKeychain {
		return persistedSession{}, sessionEntryOriginNone, false, err
	}

	sess, ok, keychainErr := readSessionFromKeychain(key)
	if keychainErr != nil {
		if err != nil {
			return persistedSession{}, sessionEntryOriginNone, false, err
		}
		return persistedSession{}, sessionEntryOriginNone, false, nil
	}
	if err != nil && !ok {
		return persistedSession{}, sessionEntryOriginNone, false, err
	}
	return sess, sessionEntryOriginWhenFound(sessionEntryOriginKeychain, ok), ok, nil
}

func sessionEntryOriginWhenFound(origin sessionEntryOrigin, found bool) sessionEntryOrigin {
	if !found {
		return sessionEntryOriginNone
	}
	return origin
}

func readSessionFromFileIgnoringErrors(key string) (persistedSession, bool, error) {
	sess, ok, err := readSessionFromFile(key)
	if err != nil {
		return persistedSession{}, false, nil
	}
	return sess, ok, nil
}

func readLastSessionFromFileIgnoringErrors() (persistedSession, bool, error) {
	key, ok, err := readLastKeyFromFile()
	if err != nil || !ok {
		return persistedSession{}, false, nil
	}
	return readSessionFromFileIgnoringErrors(key)
}

func readSessionBySelection(selection backendSelection, key string) (persistedSession, bool, error) {
	sess, _, ok, err := readSessionBySelectionWithOrigin(selection, key)
	return sess, ok, err
}

func readSessionBySelectionWithOrigin(selection backendSelection, key string) (persistedSession, sessionEntryOrigin, bool, error) {
	switch selection.backend {
	case sessionBackendOff:
		return persistedSession{}, sessionEntryOriginNone, false, nil
	case sessionBackendKeychain:
		sess, ok, err := readSessionFromKeychain(key)
		if err != nil {
			if selection.fallbackFile && isKeyringUnavailable(err) {
				return readSessionFromFileIgnoringErrorsWithOrigin(key)
			}
			return persistedSession{}, sessionEntryOriginNone, false, err
		}
		if !ok && selection.fallbackFile {
			return readSessionFromFileIgnoringErrorsWithOrigin(key)
		}
		return sess, sessionEntryOriginWhenFound(sessionEntryOriginKeychain, ok), ok, nil
	case sessionBackendFile:
		return readSessionFromFileWithKeychainFallbackOrigin(key, selection.fallbackKeychain)
	default:
		return persistedSession{}, sessionEntryOriginNone, false, nil
	}
}

func readSessionFromFileIgnoringErrorsWithOrigin(key string) (persistedSession, sessionEntryOrigin, bool, error) {
	sess, ok, err := readSessionFromFileIgnoringErrors(key)
	return sess, sessionEntryOriginWhenFound(sessionEntryOriginFile, ok), ok, err
}

func readLastSessionFromKeychain() (persistedSession, bool, error) {
	kr, err := sessionKeyringOpen()
	if err != nil {
		return persistedSession{}, false, err
	}
	store, ok, err := readSessionStoreFromKeyring(kr)
	if err != nil || !ok {
		return persistedSession{}, false, err
	}
	lastKey, ok := resolvePersistedSessionStoreLastKey(store)
	if !ok {
		return persistedSession{}, false, nil
	}
	sess, ok := store.Sessions[lastKey]
	if !ok {
		return persistedSession{}, false, nil
	}
	return sess, true, nil
}

func readLastSessionBySelection(selection backendSelection) (persistedSession, bool, error) {
	switch selection.backend {
	case sessionBackendOff:
		return persistedSession{}, false, nil
	case sessionBackendKeychain:
		sess, ok, err := readLastSessionFromKeychain()
		if err != nil {
			if selection.fallbackFile && isKeyringUnavailable(err) {
				return readLastSessionFromFileIgnoringErrors()
			}
			return persistedSession{}, false, err
		}
		if !ok && selection.fallbackFile {
			return readLastSessionFromFileIgnoringErrors()
		}
		return sess, ok, nil
	case sessionBackendFile:
		key, ok, err := readLastKeyFromFile()
		if err == nil && ok {
			return readSessionFromFileWithKeychainFallback(key, selection.fallbackKeychain)
		}
		if err != nil {
			if !selection.fallbackKeychain {
				return persistedSession{}, false, err
			}
			sess, ok, keychainErr := readLastSessionFromKeychain()
			if keychainErr == nil && ok {
				return sess, ok, nil
			}
			return persistedSession{}, false, err
		}
		if !selection.fallbackKeychain {
			return persistedSession{}, false, nil
		}
		sess, ok, err := readLastSessionFromKeychain()
		if err != nil {
			return persistedSession{}, false, nil
		}
		return sess, ok, nil
	default:
		return persistedSession{}, false, nil
	}
}

func deleteSessionFromFile(key string) error {
	path, err := webSessionFilePath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func deleteSessionFromKeychain(key string) error {
	return withSessionStoreLock(func() error { return deleteSessionFromKeychainUnlocked(key) })
}

func deleteSessionFromKeychainUnlocked(key string) error {
	return deleteSessionFromKeychainWithRecoveryUnlocked(key, false)
}

func deleteSessionFromKeychainWithRecoveryUnlocked(key string, recoverMalformed bool) error {
	kr, err := sessionKeyringOpen()
	if err != nil {
		return err
	}
	store, ok, err := readSessionStoreFromKeyring(kr)
	if err != nil {
		if recoverMalformed && errors.Is(err, errMalformedSessionStore) {
			if err := removeSessionStoreFromKeyring(kr); err != nil {
				return err
			}
			if err := removeLegacySessionFromKeyring(kr, key); err != nil {
				return err
			}
			return removeLegacyLastKeyFromKeyring(kr)
		}
		return err
	}
	if ok {
		delete(store.Sessions, key)
		if len(store.Sessions) == 0 {
			if err := removeSessionStoreFromKeyring(kr); err != nil {
				return err
			}
		} else {
			if resolved, ok := resolvePersistedSessionStoreLastKey(store); ok {
				store.LastKey = resolved
			} else {
				store.LastKey = ""
			}
			if err := writeSessionStoreToKeyring(kr, store); err != nil {
				return err
			}
		}
	}
	if err := removeLegacySessionFromKeyring(kr, key); err != nil {
		return err
	}
	return removeLegacyLastKeyFromKeyring(kr)
}

func clearLastKeyInFile() error {
	path, err := webSessionLastFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func clearLastKeyInKeychainUnlocked() error {
	kr, err := sessionKeyringOpen()
	if err != nil {
		return err
	}
	store, ok, err := readSessionStoreFromKeyring(kr)
	if err != nil {
		return err
	}
	if ok {
		store.LastKey = ""
		if len(store.Sessions) == 0 {
			if err := removeSessionStoreFromKeyring(kr); err != nil {
				return err
			}
		} else if err := writeSessionStoreToKeyring(kr, store); err != nil {
			return err
		}
	}
	return removeLegacyLastKeyFromKeyring(kr)
}

func deleteAllFromFile() error {
	dir, err := webSessionCacheDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if (strings.HasPrefix(name, "session-") && strings.HasSuffix(name, ".json")) || name == "last.json" {
			if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func deleteAllFromKeychainUnlocked() error {
	kr, err := sessionKeyringOpen()
	if err != nil {
		return err
	}
	if err := removeSessionStoreFromKeyring(kr); err != nil {
		return err
	}
	keys, err := kr.Keys()
	if err != nil {
		return err
	}
	for _, key := range keys {
		if key == webSessionStoreItem || key == webSessionLastKeyItem || strings.HasPrefix(key, webSessionKeyPrefix) {
			if err := kr.Remove(key); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
				return err
			}
		}
	}
	return nil
}

func resumeFromPersistedSession(ctx context.Context, sess persistedSession) (*AuthSession, bool, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, false, err
	}
	loaded := hydrateCookieJar(jar, sess)
	if loaded == 0 {
		return nil, false, nil
	}
	client := newWebHTTPClient(jar)
	info, err := sessionInfoFetcher(ctx, client)
	if err != nil {
		if isSessionInfoAuthExpired(err) {
			// Callers treat expiration as a soft re-auth path, so return the sentinel
			// directly instead of burying it inside transport-specific context.
			return nil, false, ErrCachedSessionExpired
		}
		return nil, false, nil
	}
	session := &AuthSession{Client: client}
	applySessionInfo(session, info)
	session.DeveloperTeamID = strings.TrimSpace(sess.DeveloperTeamID)
	return session, true, nil
}

func validatePersistedSessionReadOnly(ctx context.Context, sess persistedSession) (*http.Client, *sessionInfo, bool, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, nil, false, err
	}
	loaded := hydrateCookieJar(jar, sess)
	if loaded == 0 {
		return nil, nil, false, nil
	}
	client := newWebHTTPClient(jar)
	info, err := sessionInfoFetcher(ctx, client)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, nil, false, ctxErr
		}
		if isSessionInfoAuthExpired(err) {
			return nil, nil, false, ErrCachedSessionExpired
		}
		return nil, nil, false, ErrCachedSessionValidationFailed
	}
	return client, info, true, nil
}

func resumeFromPersistedSessionReadOnly(ctx context.Context, sess persistedSession) (*AuthSession, bool, error) {
	identity := &AuthSession{UserEmail: strings.TrimSpace(sess.UserEmail)}
	client, info, ok, err := validatePersistedSessionReadOnly(ctx, sess)
	if err != nil || !ok {
		return identity, ok, err
	}
	session := &AuthSession{Client: client, UserEmail: strings.TrimSpace(sess.UserEmail)}
	applySessionInfo(session, info)
	session.DeveloperTeamID = strings.TrimSpace(sess.DeveloperTeamID)
	return session, true, nil
}

func readOnlyFileBackendSelection() backendSelection {
	selection := resolveBackendSelection()
	if selection.backend == sessionBackendOff {
		return selection
	}
	// Deep validation is non-interactive. Read only file-backed caches so an
	// automatic or explicitly selected Keychain backend cannot open a native
	// authorization prompt.
	return backendSelection{backend: sessionBackendFile}
}

func loadSessionFromPersistedSession(sess persistedSession) (*AuthSession, bool, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, false, err
	}
	loaded := hydrateCookieJar(jar, sess)
	if loaded == 0 {
		return nil, false, nil
	}
	return &AuthSession{
		Client:           newWebHTTPClient(jar),
		UserEmail:        strings.TrimSpace(sess.UserEmail),
		DeveloperTeamID:  strings.TrimSpace(sess.DeveloperTeamID),
		cachedUpdatedAt:  sess.UpdatedAt,
		cachedGeneration: sess.Generation,
	}, true, nil
}

// PersistSession stores web-session cookies for later reuse.
func PersistSession(session *AuthSession) error {
	if session == nil || session.Client == nil || session.Client.Jar == nil {
		return nil
	}
	username := strings.TrimSpace(session.UserEmail)
	if username == "" {
		return nil
	}

	selection := resolveBackendSelection()
	if selection.backend == sessionBackendOff {
		return nil
	}

	key := webSessionCacheKey(username)
	serialized, err := serializeCookieJarWithError(session.Client.Jar, username)
	if err != nil {
		return err
	}
	serialized.DeveloperTeamID = strings.TrimSpace(session.DeveloperTeamID)
	return persistSessionBySelection(selection, key, serialized)
}

// LoadCachedSession loads a cached web session cookie jar without validating it
// against the live App Store Connect session endpoint. This is used for
// best-effort relogin attempts that want to preserve Apple trust cookies.
func LoadCachedSession(username string) (*AuthSession, bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, false, nil
	}

	selection := resolveBackendSelection()
	if selection.backend == sessionBackendOff {
		return nil, false, nil
	}

	key := webSessionCacheKey(username)
	sess, ok, err := readSessionBySelection(selection, key)
	if err != nil || !ok {
		return nil, false, err
	}
	return loadSessionFromPersistedSession(sess)
}

// ResumeCachedSessionWithoutPersist validates a cached session for one Apple
// ID without prompting or writing/migrating cached state.
func ResumeCachedSessionWithoutPersist(ctx context.Context, username string) (*AuthSession, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, false, nil
	}

	selection := readOnlyFileBackendSelection()
	if selection.backend == sessionBackendOff {
		return nil, false, nil
	}

	key := webSessionCacheKey(username)
	sess, ok, err := readSessionBySelection(selection, key)
	if err != nil || !ok {
		return nil, false, err
	}
	if strings.TrimSpace(sess.UserEmail) == "" {
		sess.UserEmail = username
	}
	return resumeFromPersistedSessionReadOnly(ctx, sess)
}

// TryResumeSession attempts to resume a session for a specific Apple ID.
func TryResumeSession(ctx context.Context, username string) (*AuthSession, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, false, nil
	}

	selection := resolveBackendSelection()
	if selection.backend == sessionBackendOff {
		return nil, false, nil
	}

	key := webSessionCacheKey(username)
	sess, ok, err := readSessionBySelection(selection, key)
	if err != nil || !ok {
		return nil, false, err
	}
	resumed, ok, err := resumeFromPersistedSession(ctx, sess)
	if err != nil || !ok || resumed == nil {
		return resumed, ok, err
	}
	// Best effort: persist refreshed cookies after successful session validation.
	_ = PersistSession(resumed)
	return resumed, true, nil
}

// LoadLastCachedSession loads the last cached web session cookie jar without
// validating it against the live App Store Connect session endpoint.
func LoadLastCachedSession() (*AuthSession, bool, error) {
	selection := resolveBackendSelection()
	if selection.backend == sessionBackendOff {
		return nil, false, nil
	}

	sess, ok, err := readLastSessionBySelection(selection)
	if err != nil || !ok {
		return nil, false, err
	}
	return loadSessionFromPersistedSession(sess)
}

// ResumeLastCachedSessionWithoutPersist validates the last cached session
// without prompting or writing/migrating cached state.
func ResumeLastCachedSessionWithoutPersist(ctx context.Context) (*AuthSession, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	selection := readOnlyFileBackendSelection()
	if selection.backend == sessionBackendOff {
		return nil, false, nil
	}

	sess, ok, err := readLastSessionBySelection(selection)
	if err != nil || !ok {
		return nil, false, err
	}
	return resumeFromPersistedSessionReadOnly(ctx, sess)
}

// TryResumeLastSession attempts to resume the last successful web session.
func TryResumeLastSession(ctx context.Context) (*AuthSession, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	selection := resolveBackendSelection()
	if selection.backend == sessionBackendOff {
		return nil, false, nil
	}

	sess, ok, err := readLastSessionBySelection(selection)
	if err != nil || !ok {
		return nil, false, err
	}
	resumed, ok, err := resumeFromPersistedSession(ctx, sess)
	if err != nil || !ok || resumed == nil {
		return resumed, ok, err
	}
	// Best effort: persist refreshed cookies after successful session validation.
	_ = PersistSession(resumed)
	return resumed, true, nil
}

// DeleteSession removes the cached session for a specific Apple ID.
func DeleteSession(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}
	key := webSessionCacheKey(username)
	selection := resolveBackendSelection()
	if selection.backend == sessionBackendOff {
		return deleteSessionEntryLocked(selection, key)
	}
	return withSessionEntryLock(key, func() error {
		return deleteSessionEntryLocked(selection, key)
	})
}

// deleteSessionEntryLocked removes every cached entry for key. Callers already
// holding the entry lock use it directly so a nested acquisition cannot stall
// behind the lock they hold.
func deleteSessionEntryLocked(selection backendSelection, key string) error {
	var err error
	switch selection.backend {
	case sessionBackendOff:
		err = nil
	case sessionBackendKeychain:
		if deleteErr := deleteSessionFromKeychain(key); deleteErr != nil {
			if selection.fallbackFile && isKeyringUnavailable(deleteErr) {
				err = deleteMirroredSessionFromFile(key)
			} else {
				err = deleteErr
			}
		} else if selection.fallbackFile {
			err = deleteMirroredSessionFromFile(key)
		}
	case sessionBackendFile:
		if deleteErr := deleteSessionFromFile(key); deleteErr != nil {
			err = deleteErr
		} else {
			err = clearLastKeyInFileIfMatches(key)
		}
		if selection.fallbackKeychain {
			err = joinDeleteErrors(err, ignoreUnavailableKeyringError(deleteSessionFromKeychain(key)))
		}
	default:
		err = nil
	}
	return err
}

// DeleteSessionIfMatches removes the cached web session for username only while
// the stored entry is still the one loaded carries. A caller that proves its
// loaded cookie jar unusable would otherwise delete by Apple ID alone and take
// out a valid replacement that a concurrent process persisted while it was
// working through 2FA, leaving no cached session at all. Reporting whether the
// delete happened lets the caller stay quiet when a newer entry was preserved.
//
// The comparison and the delete it authorizes run under the entry lock that
// persistence also takes, so a replacement written between them is no longer
// deleted by a decision made before it existed. Only the entry whose stamp
// matched is removed: the other backend can hold a newer session persisted by
// a process configured with a different ASC_WEB_SESSION_CACHE_BACKEND, and it
// is removed only when it carries the same stamp.
//
// When the current entry cannot be read, or the caller has no stamp to
// compare, the unconditional delete stands: a proven-stale jar left on disk is
// reloaded by the next invocation and burns another 2FA code against the same
// failure.
func DeleteSessionIfMatches(username string, loaded *AuthSession) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, nil
	}
	if loaded == nil || (loaded.cachedUpdatedAt.IsZero() && loaded.cachedGeneration == "") {
		return true, DeleteSession(username)
	}

	selection := resolveBackendSelection()
	if selection.backend == sessionBackendOff {
		return false, nil
	}
	key := webSessionCacheKey(username)
	deleted := false
	err := withSessionEntryLock(key, func() error {
		current, origin, ok, err := readSessionBySelectionWithOrigin(selection, key)
		if err != nil {
			deleted = true
			return deleteSessionEntryLocked(selection, key)
		}
		if !ok {
			return nil
		}
		if !samePersistedSessionIdentity(current, loaded) {
			return nil
		}
		if sessionCompareDeleteBarrier != nil {
			sessionCompareDeleteBarrier()
		}
		deleted = true
		return deleteMatchedSessionEntryLocked(selection, key, origin, current.UpdatedAt, current.Generation)
	})
	return deleted, err
}

func samePersistedSessionIdentity(current persistedSession, loaded *AuthSession) bool {
	if loaded.cachedGeneration != "" || current.Generation != "" {
		return loaded.cachedGeneration != "" && current.Generation != "" && current.Generation == loaded.cachedGeneration
	}
	return !loaded.cachedUpdatedAt.IsZero() && current.UpdatedAt.Equal(loaded.cachedUpdatedAt)
}

// deleteMatchedSessionEntryLocked removes the cache entry whose stamp matched
// the loaded session, plus the mirrored entry in the other backend only when
// that one carries the same stamp and is therefore the same proven-stale
// session. Legacy artifacts are always cleared: nothing writes them any more,
// so they can only hold a session at least as stale as the matched one.
func deleteMatchedSessionEntryLocked(selection backendSelection, key string, origin sessionEntryOrigin, stamp time.Time, generation string) error {
	var err error
	switch origin {
	case sessionEntryOriginFile:
		if deleteErr := deleteSessionFromFile(key); deleteErr != nil {
			err = deleteErr
		} else {
			err = clearLastKeyInFileIfMatches(key)
		}
		if sessionMirrorEnabled(selection) && keychainSessionCarriesIdentity(key, stamp, generation) {
			err = joinDeleteErrors(err, ignoreUnavailableKeyringError(deleteSessionFromKeychain(key)))
		}
	case sessionEntryOriginKeychain:
		if deleteErr := deleteSessionFromKeychain(key); deleteErr != nil && (!selection.fallbackFile || !isKeyringUnavailable(deleteErr)) {
			err = deleteErr
		}
		if sessionMirrorEnabled(selection) && fileSessionCarriesIdentity(key, stamp, generation) {
			err = joinDeleteErrors(err, deleteMirroredSessionFromFile(key))
		}
	default:
		return nil
	}
	return err
}

// sessionMirrorEnabled reports whether the selection keeps entries in both
// backends, which is what makes a mirrored entry possible at all.
func sessionMirrorEnabled(selection backendSelection) bool {
	switch selection.backend {
	case sessionBackendFile:
		return selection.fallbackKeychain
	case sessionBackendKeychain:
		return selection.fallbackFile
	default:
		return false
	}
}

// fileSessionCarriesStamp reports whether the file entry is the same session as
// the matched one. An unreadable entry counts: it cannot be the valid
// replacement this guard exists to protect, and leaving a corrupt file behind
// only makes the next invocation fall back to a staler backend.
func fileSessionCarriesIdentity(key string, stamp time.Time, generation string) bool {
	sess, ok, err := readSessionFromFile(key)
	if err != nil {
		// A keychain entry already proven stale may safely clean up a corrupt
		// mirrored file; leaving it causes repeated fallback failures.
		return errors.Is(err, errMalformedSessionFile)
	}
	return ok && persistedSessionIdentityMatches(sess, stamp, generation)
}

// keychainSessionCarriesStamp reports whether the keychain entry is the same
// session as the matched one. A keychain that cannot be read is left alone
// rather than cleared blindly: unavailability says nothing about the entry.
func keychainSessionCarriesIdentity(key string, stamp time.Time, generation string) bool {
	sess, ok, err := readSessionFromKeychain(key)
	if err != nil {
		return false
	}
	return ok && persistedSessionIdentityMatches(sess, stamp, generation)
}

func persistedSessionIdentityMatches(sess persistedSession, stamp time.Time, generation string) bool {
	if generation != "" || sess.Generation != "" {
		return generation != "" && sess.Generation != "" && generation == sess.Generation
	}
	return sess.UpdatedAt.Equal(stamp)
}

// DeleteAllSessions removes all cached web sessions.
func DeleteAllSessions() error {
	selection := resolveBackendSelection()
	if selection.backend == sessionBackendOff {
		return deleteAllSessionsLocked(selection)
	}
	return withSessionDeleteAllLock(selection, func() error {
		return deleteAllSessionsLocked(selection)
	})
}

// deleteAllSessionsLocked removes all cached web sessions while its caller
// holds the cache-global lock and, when a keychain backend is selected, the
// stable aggregate-store lock. Keep keychain calls on their unlocked helpers:
// the outer transaction already owns that lock.
func deleteAllSessionsLocked(selection backendSelection) error {
	var err error
	switch selection.backend {
	case sessionBackendOff:
		err = nil
	case sessionBackendKeychain:
		if deleteErr := deleteAllFromKeychainUnlocked(); deleteErr != nil {
			if selection.fallbackFile && isKeyringUnavailable(deleteErr) {
				err = deleteAllFromFile()
			} else {
				err = deleteErr
			}
		} else if selection.fallbackFile {
			err = deleteAllFromFile()
		}
	case sessionBackendFile:
		if deleteErr := deleteAllFromFile(); deleteErr != nil {
			err = deleteErr
		} else {
			err = clearLastSessionMarkerUnlocked()
		}
		if selection.fallbackKeychain {
			err = joinDeleteErrors(err, ignoreUnavailableKeyringError(deleteAllFromKeychainUnlocked()))
		}
	default:
		err = nil
	}
	return err
}

func joinDeleteErrors(primaryErr, secondaryErr error) error {
	if primaryErr == nil {
		return secondaryErr
	}
	if secondaryErr == nil {
		return primaryErr
	}
	return errors.Join(primaryErr, secondaryErr)
}

func ignoreUnavailableKeyringError(err error) error {
	if isKeyringUnavailable(err) {
		return nil
	}
	return err
}

func deleteMirroredSessionFromFile(key string) error {
	return joinDeleteErrors(deleteSessionFromFile(key), clearLastKeyInFileIfMatches(key))
}

// clearLastSessionMarker clears the "last used session" pointer.
func clearLastSessionMarker() error {
	selection := resolveBackendSelection()
	if selection.backend == sessionBackendKeychain || selection.fallbackKeychain {
		return withSessionStoreLock(clearLastSessionMarkerUnlocked)
	}
	return clearLastSessionMarkerUnlocked()
}

func clearLastSessionMarkerUnlocked() error {
	selection := resolveBackendSelection()
	switch selection.backend {
	case sessionBackendOff:
		return nil
	case sessionBackendKeychain:
		if err := clearLastKeyInKeychainUnlocked(); err != nil {
			if selection.fallbackFile && isKeyringUnavailable(err) {
				return clearLastKeyInFile()
			}
			return err
		}
		return nil
	case sessionBackendFile:
		err := clearLastKeyInFile()
		if selection.fallbackKeychain {
			err = joinDeleteErrors(err, ignoreUnavailableKeyringError(clearLastKeyInKeychainUnlocked()))
		}
		return err
	default:
		return nil
	}
}

func clearLastKeyInFileIfMatches(key string) error {
	lastKey, ok, err := readLastKeyFromFile()
	if err != nil {
		// Session deletion already succeeded. If the marker is malformed/unreadable,
		// clear it best-effort instead of turning logout into a false-negative.
		_ = clearLastKeyInFile()
		return nil
	}
	if !ok || lastKey != key {
		return nil
	}
	return clearLastKeyInFile()
}
