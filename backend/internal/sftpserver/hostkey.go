package sftpserver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// generateHostKey creates an ed25519 host key, persists it (0600) and returns
// its PEM encoding.
func generateHostKey(path string) ([]byte, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519: %w", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	data := pem.EncodeToMemory(block)

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, fmt.Errorf("write host key: %w", err)
	}
	return data, nil
}

func contextBg() context.Context { return context.Background() }
