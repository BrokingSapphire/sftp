// Package apikey generates and hashes programmatic-access API keys.
//
// Key format:  sftp_<prefix>_<secret>
//   - prefix  short, non-secret identifier stored in the DB and shown in the UI
//   - secret  256-bit URL-safe random string, shown to the user exactly once
//
// Only the SHA-256 of the whole key is persisted, so a database compromise
// does not leak usable keys.
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

const prefixLen = 8

// Generated is a freshly minted key: the plaintext (returned once) plus the
// derived prefix and hash for storage.
type Generated struct {
	Plaintext string
	Prefix    string
	Hash      string
}

// New generates a new API key.
func New() (Generated, error) {
	prefix, err := randString(prefixLen)
	if err != nil {
		return Generated{}, err
	}
	secret, err := randString(32)
	if err != nil {
		return Generated{}, err
	}
	plaintext := "sftp_" + prefix + "_" + secret
	return Generated{Plaintext: plaintext, Prefix: prefix, Hash: Hash(plaintext)}, nil
}

// Hash returns the hex SHA-256 of a full key (what is persisted / looked up).
func Hash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Valid reports whether s looks like one of our keys.
func Valid(s string) bool {
	return strings.HasPrefix(s, "sftp_") && strings.Count(s, "_") >= 2
}

func randString(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
