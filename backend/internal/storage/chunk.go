package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// chunkDir returns the temp directory holding an upload's chunks.
func (e *Engine) chunkDir(uploadID string) string {
	return filepath.Join(e.temp, "uploads", filepath.Base(uploadID))
}

// WriteChunk persists a single chunk and returns its size. Chunks are named by
// index so they can be reassembled in order and resumed after interruption.
func (e *Engine) WriteChunk(uploadID string, index int, r io.Reader) (int64, error) {
	dir := e.chunkDir(uploadID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return 0, err
	}
	path := filepath.Join(dir, fmt.Sprintf("%06d.part", index))
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return 0, err
	}
	if err := f.Sync(); err != nil {
		return 0, err
	}
	return n, nil
}

// HasChunk reports whether a chunk was already received (for resume).
func (e *Engine) HasChunk(uploadID string, index int) bool {
	path := filepath.Join(e.chunkDir(uploadID), fmt.Sprintf("%06d.part", index))
	_, err := os.Stat(path)
	return err == nil
}

// AssembleAndSave concatenates chunks 0..totalChunks-1 in order, streams them
// through the engine (computing size + checksum), and cleans up the chunks.
func (e *Engine) AssembleAndSave(uploadID string, totalChunks int) (SaveResult, error) {
	readers := make([]io.Reader, 0, totalChunks)
	closers := make([]io.Closer, 0, totalChunks)
	dir := e.chunkDir(uploadID)
	for i := 0; i < totalChunks; i++ {
		path := filepath.Join(dir, fmt.Sprintf("%06d.part", i))
		f, err := os.Open(path)
		if err != nil {
			for _, c := range closers {
				_ = c.Close()
			}
			return SaveResult{}, fmt.Errorf("missing chunk %d: %w", i, err)
		}
		readers = append(readers, f)
		closers = append(closers, f)
	}

	res, err := e.Save(io.MultiReader(readers...))
	for _, c := range closers {
		_ = c.Close()
	}
	if err != nil {
		return SaveResult{}, err
	}
	e.CleanupUpload(uploadID)
	return res, nil
}

// CleanupUpload removes all chunks for an upload (called on complete or abort).
func (e *Engine) CleanupUpload(uploadID string) {
	_ = os.RemoveAll(e.chunkDir(uploadID))
}
