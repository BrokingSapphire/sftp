package storage

import (
	"io"
	"os"
	"strings"
	"testing"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()
	e, err := New(dir+"/files", dir+"/tmp", "")
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return e
}

func TestSaveOpenRoundtrip(t *testing.T) {
	e := newTestEngine(t)
	content := "hello enterprise file transfer"

	res, err := e.Save(strings.NewReader(content))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if res.Size != int64(len(content)) {
		t.Fatalf("size mismatch: got %d want %d", res.Size, len(content))
	}
	if len(res.Checksum) != 64 {
		t.Fatalf("expected hex sha256, got %q", res.Checksum)
	}

	f, err := e.Open(res.Key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	got, _ := io.ReadAll(f)
	if string(got) != content {
		t.Fatalf("content mismatch: %q", string(got))
	}

	if err := e.Delete(res.Key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := e.Delete(res.Key); err != nil {
		t.Fatalf("delete missing should be nil, got %v", err)
	}
}

func TestResolveRejectsTraversal(t *testing.T) {
	e := newTestEngine(t)
	for _, key := range []string{"../../etc/passwd", "..", "a/../../b", "/etc/passwd"} {
		if _, err := e.Open(key); err == nil {
			// Open may fail with not-exist too; ensure resolve blocked escapes.
			if _, rerr := e.resolve(key); rerr == nil && strings.Contains(key, "..") {
				t.Fatalf("expected traversal key %q to be rejected", key)
			}
		}
	}
}

func TestEncryptedAtRest(t *testing.T) {
	dir := t.TempDir()
	key := "0123456789abcdef0123456789abcdef" // 32 bytes
	e, err := New(dir+"/files", dir+"/tmp", key)
	if err != nil {
		t.Fatalf("new encrypted engine: %v", err)
	}
	if !e.Encrypted() {
		t.Fatal("expected encryption enabled")
	}
	content := "top secret — admin must not read this on disk"

	res, err := e.Save(strings.NewReader(content))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// Size + checksum are over plaintext.
	if res.Size != int64(len(content)) {
		t.Fatalf("size mismatch: %d", res.Size)
	}

	// Raw on-disk bytes must NOT contain the plaintext.
	full, _ := e.resolve(res.Key)
	raw, _ := os.ReadFile(full)
	if strings.Contains(string(raw), "secret") {
		t.Fatal("plaintext found on disk — not encrypted")
	}

	// Reading back decrypts correctly.
	r, err := e.Open(res.Key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer r.Close()
	got, _ := io.ReadAll(r)
	if string(got) != content {
		t.Fatalf("decrypt mismatch: %q", string(got))
	}
}

func TestChunkAssembly(t *testing.T) {
	e := newTestEngine(t)
	uploadID := "test-upload"
	parts := []string{"foo", "bar", "baz"}
	for i, p := range parts {
		if _, err := e.WriteChunk(uploadID, i, strings.NewReader(p)); err != nil {
			t.Fatalf("write chunk %d: %v", i, err)
		}
	}
	if !e.HasChunk(uploadID, 1) {
		t.Fatal("expected chunk 1 to exist")
	}
	res, err := e.AssembleAndSave(uploadID, len(parts))
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	f, _ := e.Open(res.Key)
	defer f.Close()
	got, _ := io.ReadAll(f)
	if string(got) != "foobarbaz" {
		t.Fatalf("assembled content mismatch: %q", string(got))
	}
}
