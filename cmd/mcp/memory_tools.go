package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func handleStoreMemory(ctx context.Context, chroma chromaClient, in StoreMemoryInput) (StoreMemoryOutput, error) {
	if in.Content == "" {
		return StoreMemoryOutput{}, fmt.Errorf("content is required")
	}

	collection := in.Collection
	if collection == "" {
		collection = DefaultCollection
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, collection)
	if err != nil {
		return StoreMemoryOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	id := in.ID
	if id == "" {
		id = uuid.New().String()[:21]
	}

	meta := map[string]any{}
	if in.Type != "" {
		meta["type"] = in.Type
	}

	if len(in.Tags) > 0 {
		meta["tags"] = in.Tags
	}

	if err := chroma.AddBatchGeneric(ctx, resolvedID, []string{in.Content}, []string{id}, []map[string]any{meta}); err != nil {
		return StoreMemoryOutput{}, fmt.Errorf("store memory failed: %v", err)
	}

	return StoreMemoryOutput{ID: id, Count: 1}, nil
}

func handleSearchMemories(ctx context.Context, chroma chromaClient, in SearchMemoriesInput) (SearchMemoriesOutput, error) {
	if in.Query == "" {
		return SearchMemoriesOutput{}, fmt.Errorf("query is required")
	}

	collection := in.Collection
	if collection == "" {
		collection = DefaultCollection
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, collection)
	if err != nil {
		return SearchMemoriesOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	nResults := in.NResults
	if nResults <= 0 {
		nResults = defaultNResults
	}

	if nResults > maxQueryResults {
		nResults = maxQueryResults
	}

	resp, err := chroma.QueryBatch(ctx, resolvedID, []string{in.Query}, nResults)
	if err != nil {
		return SearchMemoriesOutput{}, fmt.Errorf("search failed: %v", err)
	}

	var results []MemoryResult

	if len(resp.IDs) > 0 {
		for i, id := range resp.IDs[0] {
			var doc string
			if len(resp.Documents[0]) > i {
				doc = resp.Documents[0][i]
			}

			var meta map[string]any
			if len(resp.Metadatas[0]) > i {
				meta = resp.Metadatas[0][i]
			}

			memType, _ := meta["type"].(string)
			if in.FilterType != "" && memType != in.FilterType {
				continue
			}

			tags := extractStringSlice(meta, "tags")

			var score float64
			if len(resp.Distances[0]) > i {
				score = float64(resp.Distances[0][i])
			}

			results = append(results, MemoryResult{
				ID:      id,
				Content: doc,
				Type:    memType,
				Tags:    tags,
				Score:   score,
			})
		}
	}

	return SearchMemoriesOutput{Results: results}, nil
}

func handleStoreCodeSnippet(ctx context.Context, chroma chromaClient, in StoreCodeSnippetInput) (StoreCodeSnippetOutput, error) {
	if in.Code == "" {
		return StoreCodeSnippetOutput{}, fmt.Errorf("code is required")
	}

	collection := in.Collection
	if collection == "" {
		collection = DefaultCollection
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, collection)
	if err != nil {
		return StoreCodeSnippetOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	id := in.ID
	if id == "" {
		id = uuid.New().String()[:21]
	}

	meta := map[string]any{}
	if in.Language != "" {
		meta["language"] = in.Language
	}

	if in.Description != "" {
		meta["description"] = in.Description
	}

	if len(in.Tags) > 0 {
		meta["tags"] = in.Tags
	}

	if err := chroma.AddBatchGeneric(ctx, resolvedID, []string{in.Code}, []string{id}, []map[string]any{meta}); err != nil {
		return StoreCodeSnippetOutput{}, fmt.Errorf("store code snippet failed: %v", err)
	}

	return StoreCodeSnippetOutput{ID: id, Count: 1}, nil
}

func handleSearchCode(ctx context.Context, chroma chromaClient, in SearchCodeInput) (SearchCodeOutput, error) {
	if in.Query == "" {
		return SearchCodeOutput{}, fmt.Errorf("query is required")
	}

	collection := in.Collection
	if collection == "" {
		collection = DefaultCollection
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, collection)
	if err != nil {
		return SearchCodeOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	nResults := in.NResults
	if nResults <= 0 {
		nResults = defaultNResults
	}

	if nResults > maxQueryResults {
		nResults = maxQueryResults
	}

	resp, err := chroma.QueryBatch(ctx, resolvedID, []string{in.Query}, nResults)
	if err != nil {
		return SearchCodeOutput{}, fmt.Errorf("code search failed: %v", err)
	}

	var results []CodeResult

	if len(resp.IDs) > 0 {
		for i, id := range resp.IDs[0] {
			var doc string
			if len(resp.Documents[0]) > i {
				doc = resp.Documents[0][i]
			}

			var meta map[string]any
			if len(resp.Metadatas[0]) > i {
				meta = resp.Metadatas[0][i]
			}

			lang, _ := meta["language"].(string)
			if in.Language != "" && lang != in.Language {
				continue
			}

			desc, _ := meta["description"].(string)

			var score float64
			if len(resp.Distances[0]) > i {
				score = float64(resp.Distances[0][i])
			}

			results = append(results, CodeResult{
				ID:          id,
				Code:        doc,
				Language:    lang,
				Description: desc,
				Score:       score,
			})
		}
	}

	return SearchCodeOutput{Results: results}, nil
}

func handleGetSession(ctx context.Context, chroma chromaClient, in GetSessionInput) (GetSessionOutput, error) {
	if in.ID == "" {
		return GetSessionOutput{}, fmt.Errorf("id is required")
	}

	collection := in.Collection
	if collection == "" {
		collection = DefaultCollection
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, collection)
	if err != nil {
		return GetSessionOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	records, err := chroma.ListDocuments(ctx, resolvedID)
	if err != nil {
		return GetSessionOutput{}, fmt.Errorf("list documents failed: %v", err)
	}

	for i, rid := range records.IDs {
		if rid == in.ID {
			var content string
			if len(records.Documents) > i {
				content = records.Documents[i]
			}

			var meta map[string]any
			if len(records.Metadatas) > i {
				meta = records.Metadatas[i]
			}

			tags := extractStringSlice(meta, "tags")

			return GetSessionOutput{
				ID:       rid,
				Content:  content,
				Tags:     tags,
				Metadata: meta,
			}, nil
		}
	}

	return GetSessionOutput{}, fmt.Errorf("session %q not found", in.ID)
}

func extractStringSlice(m map[string]any, key string) []string {
	raw, ok := m[key]
	if !ok {
		return nil
	}

	slice, ok := raw.([]any)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}

	return out
}
