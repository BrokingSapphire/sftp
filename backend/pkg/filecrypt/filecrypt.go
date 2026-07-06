// Package filecrypt provides transparent, seekable encryption at rest for file
// content using AES-256 in CTR mode. Each object is stored as [16-byte IV] +
// ciphertext. CTR is a stream cipher, so decryption supports random access
// (Seek) — this is what lets ranged HTTP downloads and media scrubbing keep
// working on encrypted files.
//
// The key is a single server-side master key: this protects against theft of
// the disk, backups or NAS (the bytes on disk are meaningless without the key).
// It is NOT end-to-end/zero-knowledge — an operator holding the key can decrypt,
// which is unavoidable for a server that must render previews, serve share
// links and speak SFTP.
package filecrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// IVLen is the per-object initialisation vector length (AES block size).
const IVLen = aes.BlockSize

// Cipher encrypts and decrypts file streams with a fixed 256-bit key.
type Cipher struct {
	block cipher.Block
}

// New builds a Cipher from a 32-byte key (accepts hex or raw 32 bytes).
func New(key string) (*Cipher, error) {
	raw := []byte(key)
	if len(raw) != 32 {
		if decoded, err := hex.DecodeString(key); err == nil && len(decoded) == 32 {
			raw = decoded
		}
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (or 64 hex chars), got %d", len(raw))
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, err
	}
	return &Cipher{block: block}, nil
}

// EncryptTo writes a fresh random IV to dst, then streams src encrypted after
// it, returning the number of plaintext bytes processed.
func (c *Cipher) EncryptTo(dst io.Writer, src io.Reader) (int64, error) {
	iv := make([]byte, IVLen)
	if _, err := rand.Read(iv); err != nil {
		return 0, err
	}
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}
	stream := cipher.NewCTR(c.block, iv)
	w := &cipher.StreamWriter{S: stream, W: dst}
	return io.Copy(w, src)
}

// ReaderAt returns a seekable plaintext view over an encrypted file. `size` is
// the full on-disk size (IV + ciphertext); plaintext length is size-IVLen.
func (c *Cipher) Reader(f io.ReadSeeker, size int64) (*SecureReader, error) {
	iv := make([]byte, IVLen)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(f, iv); err != nil {
		return nil, err
	}
	sr := &SecureReader{c: c, f: f, iv: iv, size: size - IVLen}
	if err := sr.reseek(0); err != nil {
		return nil, err
	}
	return sr, nil
}

// SecureReader is a plaintext io.ReadSeeker over an encrypted file.
type SecureReader struct {
	c      *Cipher
	f      io.ReadSeeker
	iv     []byte
	size   int64 // plaintext size
	pos    int64 // plaintext position
	stream cipher.Stream
}

// reseek positions the underlying file and CTR keystream at plaintext offset p.
func (s *SecureReader) reseek(p int64) error {
	block := p / aes.BlockSize
	skip := p % aes.BlockSize

	ctr := make([]byte, IVLen)
	copy(ctr, s.iv)
	addCounter(ctr, block)

	// Ciphertext byte for plaintext offset p sits at file offset IVLen+p.
	if _, err := s.f.Seek(IVLen+p, io.SeekStart); err != nil {
		return err
	}
	s.stream = cipher.NewCTR(s.c.block, ctr)
	if skip > 0 {
		// Advance the keystream by `skip` bytes so it aligns to p.
		s.stream.XORKeyStream(make([]byte, skip), make([]byte, skip))
	}
	s.pos = p
	return nil
}

// Read decrypts into p.
func (s *SecureReader) Read(p []byte) (int, error) {
	if s.pos >= s.size {
		return 0, io.EOF
	}
	if int64(len(p)) > s.size-s.pos {
		p = p[:s.size-s.pos]
	}
	n, err := s.f.Read(p)
	if n > 0 {
		s.stream.XORKeyStream(p[:n], p[:n])
		s.pos += int64(n)
	}
	return n, err
}

// Seek moves the plaintext cursor.
func (s *SecureReader) Seek(offset int64, whence int) (int64, error) {
	var target int64
	switch whence {
	case io.SeekStart:
		target = offset
	case io.SeekCurrent:
		target = s.pos + offset
	case io.SeekEnd:
		target = s.size + offset
	}
	if target < 0 {
		return 0, fmt.Errorf("negative seek")
	}
	if err := s.reseek(target); err != nil {
		return 0, err
	}
	return target, nil
}

// addCounter adds n to a big-endian counter (ctr) in place, matching how
// crypto/cipher increments the CTR counter per block.
func addCounter(ctr []byte, n int64) {
	carry := uint64(n)
	for i := len(ctr) - 1; i >= 0 && carry > 0; i-- {
		cur := uint64(ctr[i]) + (carry & 0xff)
		ctr[i] = byte(cur)
		carry = (carry >> 8) + (cur >> 8)
	}
}
