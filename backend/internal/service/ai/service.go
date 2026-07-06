// Package ai implements semantic search and retrieval-augmented "ask your files"
// on top of a self-hosted Ollama server. It is inert unless enabled in config.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"sapphirebroking.com/sftp_service/internal/db/sftpdb"
	"sapphirebroking.com/sftp_service/pkg/ai"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

const (
	chunkSize      = 800 // characters per embedded chunk
	maxChunks      = 20  // per file
	retrieveK      = 5   // chunks fed to the model
	candidateLimit = 3000
)

// Service provides embeddings + RAG. When disabled, methods return ErrDisabled.
type Service struct {
	q       *sftpdb.Queries
	client  *ai.Client
	enabled bool
	log     logger.Logger
	stop    chan struct{}
}

// ErrDisabled is returned when AI is not configured.
var ErrDisabled = fmt.Errorf("ai features are not enabled")

// New builds the AI service. When enabled is false the client may be nil.
func New(q *sftpdb.Queries, client *ai.Client, enabled bool, log logger.Logger) *Service {
	return &Service{q: q, client: client, enabled: enabled && client != nil, log: log.Named("service.ai"), stop: make(chan struct{})}
}

// Enabled reports whether AI features are available.
func (s *Service) Enabled() bool { return s.enabled }

// Source is a file that contributed to an answer.
type Source struct {
	FileID string `json:"file_id"`
	Name   string `json:"name"`
}

// Answer is the result of an ask-your-files query.
type Answer struct {
	Answer  string   `json:"answer"`
	Sources []Source `json:"sources"`
}

// Hit is a semantic-search result.
type Hit struct {
	FileID  string  `json:"file_id"`
	Name    string  `json:"name"`
	Score   float32 `json:"score"`
	Snippet string  `json:"snippet"`
}

type scored struct {
	fileID  uuid.UUID
	name    string
	content string
	score   float32
}

// rank embeds the query and scores the caller's chunks by cosine similarity.
func (s *Service) rank(ctx context.Context, owner uuid.UUID, query string) ([]scored, error) {
	qvec, err := s.client.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListEmbeddingsByOwner(ctx, sftpdb.ListEmbeddingsByOwnerParams{OwnerID: owner, RowLimit: candidateLimit})
	if err != nil {
		return nil, err
	}
	out := make([]scored, 0, len(rows))
	for _, r := range rows {
		var vec []float32
		if json.Unmarshal(r.Embedding, &vec) != nil {
			continue
		}
		out = append(out, scored{fileID: r.FileID, name: r.Name, content: r.Content, score: ai.Cosine(qvec, vec)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out, nil
}

// SemanticSearch returns the most relevant files for a natural-language query.
func (s *Service) SemanticSearch(ctx context.Context, owner uuid.UUID, query string) ([]Hit, error) {
	if !s.enabled {
		return nil, ErrDisabled
	}
	ranked, err := s.rank(ctx, owner, query)
	if err != nil {
		return nil, err
	}
	seen := map[uuid.UUID]bool{}
	hits := make([]Hit, 0, retrieveK)
	for _, r := range ranked {
		if seen[r.fileID] || r.score <= 0 {
			continue
		}
		seen[r.fileID] = true
		hits = append(hits, Hit{FileID: r.fileID.String(), Name: r.name, Score: r.score, Snippet: snippet(r.content)})
		if len(hits) >= retrieveK*2 {
			break
		}
	}
	return hits, nil
}

// Ask answers a question using the caller's own documents (RAG).
func (s *Service) Ask(ctx context.Context, owner uuid.UUID, question string) (*Answer, error) {
	if !s.enabled {
		return nil, ErrDisabled
	}
	ranked, err := s.rank(ctx, owner, question)
	if err != nil {
		return nil, err
	}
	if len(ranked) == 0 {
		return &Answer{Answer: "I couldn't find anything relevant in your files yet."}, nil
	}
	var ctxBuf strings.Builder
	seen := map[uuid.UUID]bool{}
	var sources []Source
	for i, r := range ranked {
		if i >= retrieveK {
			break
		}
		fmt.Fprintf(&ctxBuf, "[%s]\n%s\n\n", r.name, r.content)
		if !seen[r.fileID] {
			seen[r.fileID] = true
			sources = append(sources, Source{FileID: r.fileID.String(), Name: r.name})
		}
	}
	prompt := fmt.Sprintf(
		"You are a helpful assistant answering questions about the user's own documents. "+
			"Use ONLY the context below. If the answer isn't there, say you don't know. "+
			"Cite the document names in [brackets].\n\nContext:\n%s\nQuestion: %s\n\nAnswer:",
		ctxBuf.String(), question)

	answer, err := s.client.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}
	return &Answer{Answer: answer, Sources: sources}, nil
}

// StartBackfill periodically embeds files that have text but no embeddings.
func (s *Service) StartBackfill(interval time.Duration) {
	if !s.enabled {
		return
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				s.backfillOnce(context.Background())
			}
		}
	}()
}

// Stop halts the backfill loop.
func (s *Service) Stop() {
	if s.enabled {
		close(s.stop)
	}
}

func (s *Service) backfillOnce(ctx context.Context) {
	rows, err := s.q.ListFilesNeedingEmbedding(ctx, 20)
	if err != nil {
		s.log.Error("embedding backfill query failed", "err", err)
		return
	}
	for _, r := range rows {
		if err := s.embedFile(ctx, r.FileID, r.OwnerID, r.Content); err != nil {
			s.log.Warn("embed file failed", "file", r.FileID, "err", err)
		}
	}
	if len(rows) > 0 {
		s.log.Info("embedded files", "count", len(rows))
	}
}

// embedFile chunks content, embeds each chunk, and stores the vectors.
func (s *Service) embedFile(ctx context.Context, fileID, owner uuid.UUID, content string) error {
	_ = s.q.DeleteFileEmbeddings(ctx, fileID)
	chunks := chunk(content)
	for i, ch := range chunks {
		vec, err := s.client.Embed(ctx, ch)
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(vec)
		if err := s.q.InsertFileEmbedding(ctx, sftpdb.InsertFileEmbeddingParams{
			FileID: fileID, OwnerID: owner, ChunkNo: int32(i), Content: ch, Embedding: raw,
		}); err != nil {
			return err
		}
	}
	// A file with no text still gets a marker row so it isn't rescanned forever.
	if len(chunks) == 0 {
		raw, _ := json.Marshal([]float32{})
		_ = s.q.InsertFileEmbedding(ctx, sftpdb.InsertFileEmbeddingParams{
			FileID: fileID, OwnerID: owner, ChunkNo: 0, Content: "", Embedding: raw,
		})
	}
	return nil
}

// chunk splits text into ~chunkSize pieces on whitespace boundaries.
func chunk(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	var chunks []string
	var b strings.Builder
	for _, w := range words {
		if b.Len()+len(w)+1 > chunkSize {
			chunks = append(chunks, b.String())
			b.Reset()
			if len(chunks) >= maxChunks {
				break
			}
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(w)
	}
	if b.Len() > 0 && len(chunks) < maxChunks {
		chunks = append(chunks, b.String())
	}
	return chunks
}

func snippet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}
