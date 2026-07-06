package filecrypt

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

const testKey = "0123456789abcdef0123456789abcdef" // 32 bytes

func TestRoundtrip(t *testing.T) {
	c, err := New(testKey)
	if err != nil {
		t.Fatal(err)
	}
	plain := make([]byte, 100_000)
	rand.Read(plain)

	var enc bytes.Buffer
	n, err := c.EncryptTo(&enc, bytes.NewReader(plain))
	if err != nil || n != int64(len(plain)) {
		t.Fatalf("encrypt: n=%d err=%v", n, err)
	}
	if enc.Len() != len(plain)+IVLen {
		t.Fatalf("ciphertext size = %d, want %d", enc.Len(), len(plain)+IVLen)
	}

	sr, err := c.Reader(bytes.NewReader(enc.Bytes()), int64(enc.Len()))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(sr)
	if !bytes.Equal(got, plain) {
		t.Fatal("roundtrip mismatch")
	}
}

func TestSeekRange(t *testing.T) {
	c, _ := New(testKey)
	plain := make([]byte, 50_000)
	rand.Read(plain)

	var enc bytes.Buffer
	c.EncryptTo(&enc, bytes.NewReader(plain))

	sr, err := c.Reader(bytes.NewReader(enc.Bytes()), int64(enc.Len()))
	if err != nil {
		t.Fatal(err)
	}

	// Read a mid-file range (like an HTTP Range request) at a non-block offset.
	for _, off := range []int64{0, 1, 15, 16, 17, 4097, 33333} {
		if _, err := sr.Seek(off, io.SeekStart); err != nil {
			t.Fatalf("seek %d: %v", off, err)
		}
		buf := make([]byte, 1000)
		nn, _ := io.ReadFull(sr, buf)
		if !bytes.Equal(buf[:nn], plain[off:off+int64(nn)]) {
			t.Fatalf("range read at %d mismatched", off)
		}
	}
}

func TestBadKey(t *testing.T) {
	if _, err := New("short"); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestStreamWriterReader(t *testing.T) {
	c, _ := New(testKey)
	plain := make([]byte, 40_000)
	rand.Read(plain)

	var enc bytes.Buffer
	w, err := c.EncryptWriter(&enc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(plain); err != nil {
		t.Fatal(err)
	}
	if enc.Len() != len(plain)+IVLen {
		t.Fatalf("size = %d", enc.Len())
	}
	r, err := c.DecryptReader(&enc)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(r)
	if !bytes.Equal(got, plain) {
		t.Fatal("stream roundtrip mismatch")
	}
}
