package ai

import (
	"strings"
	"testing"
)

func TestChunkEmpty(t *testing.T) {
	if c := chunk(""); c != nil {
		t.Fatalf("empty text should chunk to nil, got %d", len(c))
	}
	if c := chunk("   \n  "); c != nil {
		t.Fatalf("whitespace should chunk to nil")
	}
}

func TestChunkSplitsAndCaps(t *testing.T) {
	// ~200 words → several chunks of ~chunkSize chars.
	text := strings.Repeat("word ", 4000)
	c := chunk(text)
	if len(c) == 0 {
		t.Fatal("expected chunks")
	}
	if len(c) > maxChunks {
		t.Fatalf("chunk count %d exceeds cap %d", len(c), maxChunks)
	}
	for _, ch := range c {
		if len(ch) > chunkSize+16 {
			t.Fatalf("chunk too large: %d", len(ch))
		}
	}
}

func TestChunkShort(t *testing.T) {
	c := chunk("just a short sentence")
	if len(c) != 1 {
		t.Fatalf("short text should be one chunk, got %d", len(c))
	}
}
