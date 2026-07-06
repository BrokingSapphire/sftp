// Package storage is the local-filesystem storage engine. Physical files are
// addressed by opaque, sharded keys (never by user-supplied names) so the
// engine is immune to path traversal and filename collisions. Logical names
// and hierarchy live in the database.
package storage

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/pkg/filecrypt"
)

// Engine stores and retrieves file content on a mounted filesystem, optionally
// encrypting content at rest.
type Engine struct {
	root   string
	temp   string
	cipher *filecrypt.Cipher // nil = store plaintext
}

// New creates the storage engine, ensuring the root and temp dirs exist. When
// encKey is non-empty, file content is encrypted at rest (AES-256-CTR).
func New(root, temp, encKey string) (*Engine, error) {
	for _, d := range []string{root, temp} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			return nil, fmt.Errorf("create storage dir %q: %w", d, err)
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absTmp, err := filepath.Abs(temp)
	if err != nil {
		return nil, err
	}
	e := &Engine{root: abs, temp: absTmp}
	if encKey != "" {
		c, err := filecrypt.New(encKey)
		if err != nil {
			return nil, fmt.Errorf("storage encryption: %w", err)
		}
		e.cipher = c
	}
	return e, nil
}

// Encrypted reports whether at-rest encryption is enabled.
func (e *Engine) Encrypted() bool { return e.cipher != nil }

// NewKey returns a fresh, sharded storage key (e.g. "a1/b2/<uuid>").
func NewKey() string {
	id := uuid.NewString()
	return filepath.ToSlash(filepath.Join(id[0:2], id[2:4], id))
}

// resolve maps a storage key to an absolute path, refusing any key that would
// escape the storage root (defence-in-depth against traversal).
func (e *Engine) resolve(key string) (string, error) {
	clean := filepath.Clean("/" + filepath.FromSlash(key)) // force absolute, strip ../
	full := filepath.Join(e.root, clean)
	// A valid object key always resolves to a path strictly beneath the root;
	// this rejects both traversal ("../x") and the root itself ("." / "..").
	if !strings.HasPrefix(full, e.root+string(os.PathSeparator)) {
		return "", fmt.Errorf("storage key escapes root: %q", key)
	}
	return full, nil
}

// SaveResult reports the outcome of a streamed save.
type SaveResult struct {
	Key      string
	Size     int64
	Checksum string // hex SHA-256
}

// Save streams r into a new object, computing size and SHA-256 in one pass.
// Writes to a temp file then atomically renames into place.
func (e *Engine) Save(r io.Reader) (SaveResult, error) {
	key := NewKey()
	dst, err := e.resolve(key)
	if err != nil {
		return SaveResult{}, err
	}
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0o750); err != nil {
		return SaveResult{}, err
	}

	// Create the staging file in the destination directory so the final rename
	// is always intra-filesystem (the temp dir may be a separate mount/volume,
	// which would make os.Rename fail with EXDEV).
	tmp, err := os.CreateTemp(dstDir, ".save-*")
	if err != nil {
		return SaveResult{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	// Batch disk writes through a large buffered writer and copy with a 1 MiB
	// buffer to minimise syscalls — materially faster for big uploads. The
	// SHA-256 checksum is always computed over the PLAINTEXT so it is stable
	// regardless of whether encryption is enabled.
	bw := bufio.NewWriterSize(tmp, 1<<20)
	h := sha256.New()
	var size int64
	if e.cipher != nil {
		// Encrypt to disk; hash the plaintext via a tee.
		size, err = e.cipher.EncryptTo(bw, io.TeeReader(r, h))
	} else {
		buf := make([]byte, 1<<20)
		size, err = io.CopyBuffer(io.MultiWriter(bw, h), r, buf)
	}
	if err != nil {
		tmp.Close()
		return SaveResult{}, err
	}
	if err := bw.Flush(); err != nil {
		tmp.Close()
		return SaveResult{}, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return SaveResult{}, err
	}
	if err := tmp.Close(); err != nil {
		return SaveResult{}, err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return SaveResult{}, err
	}
	return SaveResult{Key: key, Size: size, Checksum: hex.EncodeToString(h.Sum(nil))}, nil
}

// Open returns the object for reading as a seekable, closeable stream
// (transparently decrypting when encryption is enabled). Seek support keeps
// HTTP range requests working on encrypted files.
func (e *Engine) Open(key string) (io.ReadSeekCloser, error) {
	full, err := e.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	if e.cipher == nil {
		return f, nil
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	sr, err := e.cipher.Reader(f, info.Size())
	if err != nil {
		f.Close()
		return nil, err
	}
	return &secureFile{f: f, sr: sr}, nil
}

// secureFile adapts a decrypting SecureReader + its backing file into an
// io.ReadSeekCloser.
type secureFile struct {
	f  *os.File
	sr *filecrypt.SecureReader
}

func (s *secureFile) Read(p []byte) (int, error)                 { return s.sr.Read(p) }
func (s *secureFile) Seek(off int64, whence int) (int64, error) { return s.sr.Seek(off, whence) }
func (s *secureFile) Close() error                              { return s.f.Close() }

// Stat returns file info for the object.
func (e *Engine) Stat(key string) (os.FileInfo, error) {
	full, err := e.resolve(key)
	if err != nil {
		return nil, err
	}
	return os.Stat(full)
}

// Delete removes the object. Missing objects are not an error.
func (e *Engine) Delete(key string) error {
	full, err := e.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// TempDir returns the temp directory (used by the chunked-upload assembler).
func (e *Engine) TempDir() string { return e.temp }
