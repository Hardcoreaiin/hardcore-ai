package embedded

import (
	"context"
	"fmt"
	"os"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"

	"hardcoreai-rag/indexing"
	"hardcoreai-rag/retrieval"
	"hardcoreai-rag/storage"
)

type ragQueryArgs struct {
	Query      string `tool:"query" desc:"semantic search query for STM32 reference manuals and datasheets"`
	ChipFamily string `tool:"chip_family" desc:"optional filter by chip family (e.g. STM32F4, STM32F7, STM32H7). Use empty string if not filtering."`
}

func RegisterRAGQuery(r *tools.Registry, db *storage.DB) {
	tools.Register(r, "rag_query",
		"Perform semantic search query across all ingested STM32 manuals/datasheets to retrieve technical reference information (e.g. register descriptions, offset, configurations).",
		func(ctx context.Context, a ragQueryArgs) (string, []tools.Artifact, error) {
			if db == nil {
				return "Error: RAG database is not initialized.", nil, nil
			}

			embedder := indexing.NewEmbedder()
			engine := retrieval.NewEngine(db, embedder)

			opts := retrieval.RetrievalOptions{
				K:          3,
				ChipFamily: a.ChipFamily,
				MaxTokens:  3000,
			}

			res, err := engine.Retrieve(ctx, a.Query, opts)
			if err != nil {
				// Write the error to a file in the workspace
				_ = os.WriteFile("rag_error.log", []byte(fmt.Sprintf("Query: %q, ChipFamily: %q, Err: %v\n", a.Query, a.ChipFamily, err)), 0644)
				return "", nil, fmt.Errorf("rag query failed: %w", err)
			}

			if len(res.Chunks) == 0 {
				return "No relevant documentation found for query: " + a.Query, nil, nil
			}

			return res.Context, nil, nil
		})
}
