package embedded

import (
	"context"
	"strings"

	"hardcoreai-rag/indexing"
	"hardcoreai-rag/retrieval"
	"hardcoreai-rag/storage"
)

// RAGRetriever performs automatic background reference retrieval. It is not an
// LLM-callable tool: the agent loop queries it at the start of each turn and
// injects any result as hidden context. When nothing relevant is found it
// returns ok=false so the model never sees a "no documentation" message.
//
// It satisfies the agent.Retriever interface structurally.
type RAGRetriever struct {
	db     *storage.DB
	engine *retrieval.Engine
}

// NewRAGRetriever builds a retriever over the given RAG database. Returns nil
// when db is nil so callers can simply skip installing it.
func NewRAGRetriever(db *storage.DB) *RAGRetriever {
	if db == nil {
		return nil
	}
	return &RAGRetriever{
		db:     db,
		engine: retrieval.NewEngine(db, indexing.NewEmbedder()),
	}
}

// Retrieve runs a hybrid search for the prompt and returns assembled context.
// ok is false when the database is unavailable, the search errors, or no
// chunks match — in every one of those cases the caller injects nothing.
func (r *RAGRetriever) Retrieve(ctx context.Context, query string) (string, bool) {
	if r == nil || r.engine == nil {
		return "", false
	}
	if strings.TrimSpace(query) == "" {
		return "", false
	}

	res, err := r.engine.Retrieve(ctx, query, retrieval.RetrievalOptions{
		K:         3,
		MaxTokens: 3000,
	})
	if err != nil || res == nil || len(res.Chunks) == 0 {
		return "", false
	}
	if strings.TrimSpace(res.Context) == "" {
		return "", false
	}
	return res.Context, true
}
