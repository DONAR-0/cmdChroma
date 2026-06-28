package service

import (
	"context"
	"fmt"

	"github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
)

// ChromaStore implements langchaingo.VectorStore using ChromaClientInterface.
type ChromaStore struct {
	client       client.ChromaClientInterface
	collectionID string
}

// NewChromaStore creates a new ChromaStore backed by the given client and collection.
func NewChromaStore(c client.ChromaClientInterface, collectionID string) *ChromaStore {
	return &ChromaStore{
		client:       c,
		collectionID: collectionID,
	}
}

// AddDocuments stores documents in ChromaDB with embeddings.
func (cs *ChromaStore) AddDocuments(ctx context.Context, docs []schema.Document, options ...vectorstores.Option) ([]string, error) {
	var (
		contents  []string
		ids       []string
		metadatas []map[string]any
	)

	for i, doc := range docs {
		contents = append(contents, doc.PageContent)

		id, ok := doc.Metadata["id"].(string)
		if !ok {
			id = fmt.Sprintf("doc_%d", i)
		}

		ids = append(ids, id)
		metadatas = append(metadatas, doc.Metadata)
	}

	err := cs.client.AddBatchGeneric(ctx, cs.collectionID, contents, ids, metadatas)

	return ids, err
}

// SimilaritySearch finds similar documents by vector distance.
func (cs *ChromaStore) SimilaritySearch(ctx context.Context, query string, numDocuments int, options ...vectorstores.Option) ([]schema.Document, error) {
	resp, err := cs.client.QueryBatch(ctx, cs.collectionID, []string{query}, numDocuments)
	if err != nil {
		return nil, err
	}

	var docs []schema.Document

	if len(resp.Documents) > 0 {
		for i := 0; i < len(resp.Documents[0]); i++ {
			docs = append(docs, schema.Document{
				PageContent: resp.Documents[0][i],
				Metadata:    resp.Metadatas[0][i],
			})
		}
	}

	return docs, nil
}

// RunRetrievalQA handles the RAG pipeline using LangChainGo chains.
func RunRetrievalQA(ctx context.Context, llm llms.Model, store vectorstores.VectorStore, query string) (string, error) {
	retriever := vectorstores.ToRetriever(store, 3)

	chain := chains.NewRetrievalQAFromLLM(llm, retriever)

	result, err := chains.Call(ctx, chain, map[string]any{
		"query": query,
	})
	if err != nil {
		return "", err
	}

	return result["text"].(string), nil
}
