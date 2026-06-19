// Package main — tool registration (handlers, schemas, mcp server wiring)
// lands here.
//
// T-05 lands the skeleton: a constructor that builds an mcp-go server with
// capabilities enabled but ZERO tools registered. T-06..T-09 each add tools
// via server.AddTool calls; T-11 attaches output-budget annotations; T-12
// wraps this constructor with stdio/HTTP transport; T-13 hooks the config +
// embedder injection path in main.go.
//
// tools.go owns the *server.MCPServer construction; tools_test.go hands that
// server to the in-memory transport provided by mcptest. Handler tasks register
// tools via this server through closures; they never re-enter this file.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
)

const (
	serverName        = "cmdChroma MCP"
	serverVersion     = "0.1.0"
	maxBatchSize      = 5461
	maxQueryResults   = 100
	defaultNResults   = 5
	outputBudgetChars = 200000
	shutdownTimeout   = 5 * time.Second
)

func buildServer(chroma client.ChromaClientInterface, embedder onnx.EmbedderInterface, mode string) *server.MCPServer {
	_ = embedder

	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
	)

	storeTool := mcp.NewTool("store_documents",
		mcp.WithDescription("Store text documents as embeddings in a ChromaDB collection"),
		mcp.WithInputSchema[StoreDocumentsInput](),
		mcp.WithOutputSchema[StoreDocumentsOutput](),
	)
	s.AddTool(storeTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args StoreDocumentsInput) (StoreDocumentsOutput, error) {
			return handleStoreDocuments(ctx, chroma, args)
		},
	))

	queryTool := withOutputBudget(mcp.NewTool("query_documents",
		mcp.WithDescription("Search semantically across stored documents in a collection"),
		mcp.WithInputSchema[QueryDocumentsInput](),
		mcp.WithOutputSchema[QueryDocumentsOutput](),
	), outputBudgetChars)
	s.AddTool(queryTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args QueryDocumentsInput) (QueryDocumentsOutput, error) {
			return handleQueryDocuments(ctx, chroma, args)
		},
	))

	collectionListTool := withOutputBudget(mcp.NewTool("collection_list",
		mcp.WithDescription("List all collections accessible to the configured tenant/database"),
		mcp.WithInputSchema[CollectionListInput](),
		mcp.WithOutputSchema[CollectionListOutput](),
	), outputBudgetChars)
	s.AddTool(collectionListTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args CollectionListInput) (CollectionListOutput, error) {
			return handleCollectionList(ctx, chroma)
		},
	))

	collectionCreateTool := mcp.NewTool("collection_create",
		mcp.WithDescription("Create a new empty collection"),
		mcp.WithInputSchema[CollectionCreateInput](),
		mcp.WithOutputSchema[CollectionCreateOutput](),
	)
	s.AddTool(collectionCreateTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args CollectionCreateInput) (CollectionCreateOutput, error) {
			return handleCollectionCreate(chroma, args)
		},
	))

	collectionDeleteTool := mcp.NewTool("collection_delete",
		mcp.WithDescription("Permanently delete a collection and all its data"),
		mcp.WithInputSchema[CollectionDeleteInput](),
		mcp.WithOutputSchema[CollectionDeleteOutput](),
	)
	s.AddTool(collectionDeleteTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args CollectionDeleteInput) (CollectionDeleteOutput, error) {
			return handleCollectionDelete(chroma, args)
		},
	))

	collectionStatsTool := withOutputBudget(mcp.NewTool("collection_stats",
		mcp.WithDescription("Get document count and sample IDs for a collection"),
		mcp.WithInputSchema[CollectionStatsInput](),
		mcp.WithOutputSchema[CollectionStatsOutput](),
	), outputBudgetChars)
	s.AddTool(collectionStatsTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args CollectionStatsInput) (CollectionStatsOutput, error) {
			return handleCollectionStats(ctx, chroma, args)
		},
	))

	forgetTool := mcp.NewTool("forget",
		mcp.WithDescription("Delete specific records from a collection by ID, or clear all records"),
		mcp.WithInputSchema[ForgetInput](),
		mcp.WithOutputSchema[ForgetOutput](),
	)
	s.AddTool(forgetTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args ForgetInput) (ForgetOutput, error) {
			return handleForget(ctx, chroma, args)
		},
	))

	if mode == "memory" {
		registerMemoryTools(s, chroma)
	}

	return s
}

func withOutputBudget(t mcp.Tool, maxChars int) mcp.Tool {
	t.Meta = &mcp.Meta{
		AdditionalFields: map[string]any{
			"anthropic/maxResultSizeChars": maxChars,
		},
	}

	return t
}

func registerMemoryTools(s *server.MCPServer, chroma client.ChromaClientInterface) {
	storeMemTool := mcp.NewTool("store_memory",
		mcp.WithDescription("Store a piece of knowledge for future retrieval"),
		mcp.WithInputSchema[StoreMemoryInput](),
		mcp.WithOutputSchema[StoreMemoryOutput](),
	)
	s.AddTool(storeMemTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args StoreMemoryInput) (StoreMemoryOutput, error) {
			return handleStoreMemory(ctx, chroma, args)
		},
	))

	searchMemTool := withOutputBudget(mcp.NewTool("search_memories",
		mcp.WithDescription("Search semantically across stored knowledge, optionally filtered by type"),
		mcp.WithInputSchema[SearchMemoriesInput](),
		mcp.WithOutputSchema[SearchMemoriesOutput](),
	), outputBudgetChars)
	s.AddTool(searchMemTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args SearchMemoriesInput) (SearchMemoriesOutput, error) {
			return handleSearchMemories(ctx, chroma, args)
		},
	))

	storeCodeTool := mcp.NewTool("store_code_snippet",
		mcp.WithDescription("Index a reusable code snippet with language and description metadata"),
		mcp.WithInputSchema[StoreCodeSnippetInput](),
		mcp.WithOutputSchema[StoreCodeSnippetOutput](),
	)
	s.AddTool(storeCodeTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args StoreCodeSnippetInput) (StoreCodeSnippetOutput, error) {
			return handleStoreCodeSnippet(ctx, chroma, args)
		},
	))

	searchCodeTool := withOutputBudget(mcp.NewTool("search_code",
		mcp.WithDescription("Find code snippets by semantic meaning, optionally filtered by language"),
		mcp.WithInputSchema[SearchCodeInput](),
		mcp.WithOutputSchema[SearchCodeOutput](),
	), outputBudgetChars)
	s.AddTool(searchCodeTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args SearchCodeInput) (SearchCodeOutput, error) {
			return handleSearchCode(ctx, chroma, args)
		},
	))

	getSessionTool := mcp.NewTool("get_session",
		mcp.WithDescription("Retrieve a previously saved session by its ID"),
		mcp.WithInputSchema[GetSessionInput](),
		mcp.WithOutputSchema[GetSessionOutput](),
	)
	s.AddTool(getSessionTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args GetSessionInput) (GetSessionOutput, error) {
			return handleGetSession(ctx, chroma, args)
		},
	))
}

func handleStoreDocuments(ctx context.Context, chroma client.ChromaClientInterface, in StoreDocumentsInput) (StoreDocumentsOutput, error) {
	if in.CollectionID == "" {
		return StoreDocumentsOutput{}, fmt.Errorf("collection_id is required")
	}

	if len(in.Documents) == 0 {
		return StoreDocumentsOutput{}, fmt.Errorf("documents must be non-empty")
	}

	ids := in.IDs
	if len(ids) == 0 {
		ids = make([]string, len(in.Documents))
		for i := range in.Documents {
			ids[i] = uuid.New().String()[:21]
		}
	}

	if len(ids) != len(in.Documents) {
		return StoreDocumentsOutput{}, fmt.Errorf("ids length (%d) must match documents length (%d)", len(ids), len(in.Documents))
	}

	if len(in.Metadatas) > 0 && len(in.Metadatas) != len(in.Documents) {
		return StoreDocumentsOutput{}, fmt.Errorf("metadatas length (%d) must match documents length (%d)", len(in.Metadatas), len(in.Documents))
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, in.CollectionID)
	if err != nil {
		return StoreDocumentsOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	var addErr error

	if len(in.Metadatas) > 0 {
		for start := 0; start < len(in.Documents); start += maxBatchSize {
			end := start + maxBatchSize
			if end > len(in.Documents) {
				end = len(in.Documents)
			}

			if addErr = chroma.AddBatchGeneric(ctx, resolvedID, in.Documents[start:end], ids[start:end], in.Metadatas[start:end]); addErr != nil {
				break
			}
		}
	} else {
		for start := 0; start < len(in.Documents); start += maxBatchSize {
			end := start + maxBatchSize
			if end > len(in.Documents) {
				end = len(in.Documents)
			}

			if addErr = chroma.AddBatch(ctx, resolvedID, in.Documents[start:end], ids[start:end]); addErr != nil {
				break
			}
		}
	}

	if addErr != nil {
		return StoreDocumentsOutput{}, fmt.Errorf("store failed: %v", addErr)
	}

	return StoreDocumentsOutput{Count: len(in.Documents), IDs: ids}, nil
}

func handleQueryDocuments(ctx context.Context, chroma client.ChromaClientInterface, in QueryDocumentsInput) (QueryDocumentsOutput, error) {
	if in.CollectionID == "" {
		return QueryDocumentsOutput{}, fmt.Errorf("collection_id is required")
	}

	if len(in.QueryTexts) == 0 {
		return QueryDocumentsOutput{}, fmt.Errorf("query_texts must be non-empty")
	}

	nResults := in.NResults
	if nResults <= 0 {
		nResults = defaultNResults
	}

	if nResults > maxQueryResults {
		nResults = maxQueryResults
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, in.CollectionID)
	if err != nil {
		return QueryDocumentsOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	start := time.Now()

	resp, err := chroma.QueryBatch(ctx, resolvedID, in.QueryTexts, nResults)
	if err != nil {
		return QueryDocumentsOutput{}, fmt.Errorf("query failed: %v", err)
	}

	durationMs := time.Since(start).Milliseconds()

	out := QueryDocumentsOutput{
		IDs:        resp.IDs,
		Documents:  resp.Documents,
		Metadatas:  resp.Metadatas,
		Distances:  make([][]float64, len(resp.Distances)),
		DurationMs: durationMs,
	}
	for i, dd := range resp.Distances {
		out.Distances[i] = make([]float64, len(dd))
		for j, d := range dd {
			out.Distances[i][j] = float64(d)
		}
	}

	return out, nil
}

func handleCollectionList(ctx context.Context, chroma client.ChromaClientInterface) (CollectionListOutput, error) {
	cols, err := chroma.ListCollections()
	if err != nil {
		return CollectionListOutput{}, fmt.Errorf("list collections failed: %v", err)
	}

	out := CollectionListOutput{
		Collections: make([]CollectionSummary, len(cols)),
	}
	for i, c := range cols {
		out.Collections[i] = CollectionSummary{
			Name: c.Name,
			ID:   c.ID,
		}
	}

	return out, nil
}

func handleCollectionCreate(chroma client.ChromaClientInterface, in CollectionCreateInput) (CollectionCreateOutput, error) {
	if in.Name == "" {
		return CollectionCreateOutput{}, fmt.Errorf("name is required")
	}

	id, err := chroma.CreateCollection(in.Name)
	if err != nil {
		return CollectionCreateOutput{}, fmt.Errorf("create collection failed: %v", err)
	}

	return CollectionCreateOutput{ID: id, Name: in.Name}, nil
}

func handleCollectionDelete(chroma client.ChromaClientInterface, in CollectionDeleteInput) (CollectionDeleteOutput, error) {
	if in.Name == "" {
		return CollectionDeleteOutput{}, fmt.Errorf("name is required")
	}

	if err := chroma.DeleteCollection(in.Name); err != nil {
		return CollectionDeleteOutput{}, fmt.Errorf("delete collection failed: %v", err)
	}

	return CollectionDeleteOutput{Deleted: true, Name: in.Name}, nil
}

func handleCollectionStats(ctx context.Context, chroma client.ChromaClientInterface, in CollectionStatsInput) (CollectionStatsOutput, error) {
	if in.CollectionID == "" {
		return CollectionStatsOutput{}, fmt.Errorf("collection_id is required")
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, in.CollectionID)
	if err != nil {
		return CollectionStatsOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	records, err := chroma.ListDocuments(resolvedID)
	if err != nil {
		return CollectionStatsOutput{}, fmt.Errorf("list documents failed: %v", err)
	}

	count := len(records.IDs)

	out := CollectionStatsOutput{
		Collection: resolvedID,
		Count:      count,
	}
	if count > 0 {
		if count > 5 {
			out.SampleIDs = records.IDs[:5]
		} else {
			out.SampleIDs = records.IDs
		}
	}

	return out, nil
}

func handleForget(ctx context.Context, chroma client.ChromaClientInterface, in ForgetInput) (ForgetOutput, error) {
	if in.CollectionID == "" {
		return ForgetOutput{}, fmt.Errorf("collection_id is required")
	}

	if len(in.IDs) == 0 && !in.All {
		return ForgetOutput{}, fmt.Errorf("provide either ids or set all=true")
	}

	if len(in.IDs) > 0 && in.All {
		return ForgetOutput{}, fmt.Errorf("provide either ids or all=true, not both")
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, in.CollectionID)
	if err != nil {
		return ForgetOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	var idsToDelete []string

	if in.All {
		records, err := chroma.ListDocuments(resolvedID)
		if err != nil {
			return ForgetOutput{}, fmt.Errorf("list documents failed: %v", err)
		}

		idsToDelete = records.IDs
	} else {
		idsToDelete = in.IDs
	}

	if len(idsToDelete) == 0 {
		return ForgetOutput{DeletedCount: 0, Mode: "ids"}, nil
	}

	if err := chroma.DeleteRecords(resolvedID, idsToDelete); err != nil {
		return ForgetOutput{}, fmt.Errorf("delete records failed: %v", err)
	}

	mode := "ids"
	if in.All {
		mode = "all"
	}

	return ForgetOutput{DeletedCount: len(idsToDelete), Mode: mode}, nil
}
