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
)

// chromaClient is the subset of *client.ChromaClient methods used by MCP handlers.
type chromaClient interface {
	ResolveCollectionID(ctx context.Context, input string) (string, error)
	AddBatch(ctx context.Context, collectionID string, docs []string, ids []string) error
	AddBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error
	QueryBatch(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*client.QueryResponse, error)
	ListCollections(ctx context.Context) ([]client.Collection, error)
	CreateCollection(ctx context.Context, name string) (string, error)
	DeleteCollection(ctx context.Context, name string) error
	ListDocuments(ctx context.Context, collectionID string) (*client.GetRecordsResponse, error)
	DeleteRecords(ctx context.Context, collectionID string, ids []string) error
}

// textEmbedder is the subset of *onnx.Embedder methods used by MCP handlers.
type textEmbedder interface {
	Embed(text string) ([]float32, error)
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	Close()
}

const (
	serverName        = "cmdChroma MCP"
	serverVersion     = "0.1.0"
	maxBatchSize      = 5461
	maxQueryResults   = 100
	defaultNResults   = 5
	outputBudgetChars = 200000
	shutdownTimeout   = 5 * time.Second
)

func buildServer(chroma chromaClient, embedder textEmbedder, mode string) *server.MCPServer {
	_ = embedder

	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
	)

	storeTool := mcp.NewTool("store_documents",
		mcp.WithDescription("Store text documents as embeddings in a ChromaDB collection"),
		mcp.WithToolTitle("Store Documents"),
		mcp.WithString("collection_id", mcp.Required(), mcp.Description("Target collection name or ID")),
		mcp.WithArray("documents", mcp.Required(), mcp.Description("Texts to embed and store"), mcp.WithStringItems(), mcp.MinItems(1)),
		mcp.WithArray("ids", mcp.Description("Optional explicit IDs; one per document"), mcp.WithStringItems()),
		mcp.WithArray("metadatas", mcp.Description("Optional metadata, parallel to documents"), mcp.Items(map[string]any{"type": "object", "additionalProperties": true})),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("StoreDocumentsOutput")),
	)
	s.AddTool(storeTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args StoreDocumentsInput) (StoreDocumentsOutput, error) {
			return handleStoreDocuments(ctx, chroma, args)
		},
	))

	queryTool := withOutputBudget(mcp.NewTool("query_documents",
		mcp.WithDescription("Search semantically across stored documents in a collection"),
		mcp.WithToolTitle("Query Documents"),
		mcp.WithString("collection_id", mcp.Required(), mcp.Description("Target collection name or ID")),
		mcp.WithArray("query_texts", mcp.Required(), mcp.Description("One or more natural-language queries"), mcp.WithStringItems(), mcp.MinItems(1)),
		mcp.WithInteger("n_results", mcp.Description("Maximum hits to return per query"), mcp.DefaultNumber(5), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("QueryDocumentsOutput")),
	), outputBudgetChars)
	s.AddTool(queryTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args QueryDocumentsInput) (QueryDocumentsOutput, error) {
			return handleQueryDocuments(ctx, chroma, args)
		},
	))

	collectionListTool := withOutputBudget(mcp.NewTool("collection_list",
		mcp.WithDescription("List all collections accessible to the configured tenant/database"),
		mcp.WithToolTitle("List Collections"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("CollectionListOutput")),
	), outputBudgetChars)
	s.AddTool(collectionListTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args CollectionListInput) (CollectionListOutput, error) {
			return handleCollectionList(ctx, chroma)
		},
	))

	collectionCreateTool := mcp.NewTool("collection_create",
		mcp.WithDescription("Create a new empty collection"),
		mcp.WithToolTitle("Create Collection"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Collection name (1-64 chars)"), mcp.MinLength(1), mcp.MaxLength(64)),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("CollectionCreateOutput")),
	)
	s.AddTool(collectionCreateTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args CollectionCreateInput) (CollectionCreateOutput, error) {
			return handleCollectionCreate(ctx, chroma, args)
		},
	))

	collectionDeleteTool := mcp.NewTool("collection_delete",
		mcp.WithDescription("Permanently delete a collection and all its data"),
		mcp.WithToolTitle("Delete Collection"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Collection name to permanently delete"), mcp.MinLength(1)),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("CollectionDeleteOutput")),
	)
	s.AddTool(collectionDeleteTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args CollectionDeleteInput) (CollectionDeleteOutput, error) {
			return handleCollectionDelete(ctx, chroma, args)
		},
	))

	collectionStatsTool := withOutputBudget(mcp.NewTool("collection_stats",
		mcp.WithDescription("Get document count and sample IDs for a collection"),
		mcp.WithToolTitle("Collection Stats"),
		mcp.WithString("collection_id", mcp.Required(), mcp.Description("Target collection name or ID")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("CollectionStatsOutput")),
	), outputBudgetChars)
	s.AddTool(collectionStatsTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args CollectionStatsInput) (CollectionStatsOutput, error) {
			return handleCollectionStats(ctx, chroma, args)
		},
	))

	forgetTool := mcp.NewTool("forget",
		mcp.WithDescription("Delete specific records from a collection by ID, or clear all records"),
		mcp.WithToolTitle("Forget Documents"),
		mcp.WithString("collection_id", mcp.Required(), mcp.Description("Target collection name or ID")),
		mcp.WithArray("ids", mcp.Description("Specific record IDs to delete"), mcp.WithStringItems(), mcp.MinItems(1)),
		mcp.WithBoolean("all", mcp.Description("Set true to delete every record in the collection")),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("ForgetOutput")),
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

func registerMemoryTools(s *server.MCPServer, chroma chromaClient) {
	storeMemTool := mcp.NewTool("store_memory",
		mcp.WithDescription("Store a piece of knowledge for future retrieval"),
		mcp.WithToolTitle("Store Memory"),
		mcp.WithString("content", mcp.Required(), mcp.Description("The knowledge content")),
		mcp.WithString("type", mcp.Description("Knowledge type — narrows search filters"), mcp.Enum("decision", "error_solution", "fact", "gotcha", "pattern", "session", "snippet")),
		mcp.WithArray("tags", mcp.Description("Searchable tags"), mcp.WithStringItems()),
		mcp.WithString("collection", mcp.Description("Collection name (default: mcp_memory)")),
		mcp.WithString("id", mcp.Description("Optional ID (auto-generated if empty)")),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("StoreMemoryOutput")),
	)
	s.AddTool(storeMemTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args StoreMemoryInput) (StoreMemoryOutput, error) {
			return handleStoreMemory(ctx, chroma, args)
		},
	))

	searchMemTool := withOutputBudget(mcp.NewTool("search_memories",
		mcp.WithDescription("Search semantically across stored knowledge, optionally filtered by type"),
		mcp.WithToolTitle("Search Memories"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural language search query")),
		mcp.WithInteger("n_results", mcp.Description("Maximum hits to return"), mcp.DefaultNumber(5), mcp.Min(1), mcp.Max(100)),
		mcp.WithString("filter_type", mcp.Description("Optional type filter, e.g. decision|pattern|fact")),
		mcp.WithString("collection", mcp.Description("Collection name (default: mcp_memory)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("SearchMemoriesOutput")),
	), outputBudgetChars)
	s.AddTool(searchMemTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args SearchMemoriesInput) (SearchMemoriesOutput, error) {
			return handleSearchMemories(ctx, chroma, args)
		},
	))

	storeCodeTool := mcp.NewTool("store_code_snippet",
		mcp.WithDescription("Index a reusable code snippet with language and description metadata"),
		mcp.WithToolTitle("Store Code Snippet"),
		mcp.WithString("code", mcp.Required(), mcp.Description("The source code to store")),
		mcp.WithString("language", mcp.Description("Programming language")),
		mcp.WithString("description", mcp.Description("Human-readable description")),
		mcp.WithArray("tags", mcp.Description("Searchable tags"), mcp.WithStringItems()),
		mcp.WithString("collection", mcp.Description("Collection name (default: mcp_memory)")),
		mcp.WithString("id", mcp.Description("Optional ID (auto-generated if empty)")),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("StoreCodeSnippetOutput")),
	)
	s.AddTool(storeCodeTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args StoreCodeSnippetInput) (StoreCodeSnippetOutput, error) {
			return handleStoreCodeSnippet(ctx, chroma, args)
		},
	))

	searchCodeTool := withOutputBudget(mcp.NewTool("search_code",
		mcp.WithDescription("Find code snippets by semantic meaning, optionally filtered by language"),
		mcp.WithToolTitle("Search Code Snippets"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Semantic search query")),
		mcp.WithInteger("n_results", mcp.Description("Maximum hits to return"), mcp.DefaultNumber(5), mcp.Min(1), mcp.Max(100)),
		mcp.WithString("language", mcp.Description("Optional language filter")),
		mcp.WithString("collection", mcp.Description("Collection name (default: mcp_memory)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("SearchCodeOutput")),
	), outputBudgetChars)
	s.AddTool(searchCodeTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args SearchCodeInput) (SearchCodeOutput, error) {
			return handleSearchCode(ctx, chroma, args)
		},
	))

	getSessionTool := mcp.NewTool("get_session",
		mcp.WithDescription("Retrieve a previously saved session by its ID"),
		mcp.WithToolTitle("Get Session"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Session ID to retrieve")),
		mcp.WithString("collection", mcp.Description("Collection name (default: mcp_memory)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithRawOutputSchema(outputJSONSchema("GetSessionOutput")),
	)
	s.AddTool(getSessionTool, mcp.NewStructuredToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args GetSessionInput) (GetSessionOutput, error) {
			return handleGetSession(ctx, chroma, args)
		},
	))
}

func handleStoreDocuments(ctx context.Context, chroma chromaClient, in StoreDocumentsInput) (StoreDocumentsOutput, error) {
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

func handleQueryDocuments(ctx context.Context, chroma chromaClient, in QueryDocumentsInput) (QueryDocumentsOutput, error) {
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

func handleCollectionList(ctx context.Context, chroma chromaClient) (CollectionListOutput, error) {
	cols, err := chroma.ListCollections(ctx)
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

func handleCollectionCreate(ctx context.Context, chroma chromaClient, in CollectionCreateInput) (CollectionCreateOutput, error) {
	if in.Name == "" {
		return CollectionCreateOutput{}, fmt.Errorf("name is required")
	}

	id, err := chroma.CreateCollection(ctx, in.Name)
	if err != nil {
		return CollectionCreateOutput{}, fmt.Errorf("create collection failed: %v", err)
	}

	return CollectionCreateOutput{ID: id, Name: in.Name}, nil
}

func handleCollectionDelete(ctx context.Context, chroma chromaClient, in CollectionDeleteInput) (CollectionDeleteOutput, error) {
	if in.Name == "" {
		return CollectionDeleteOutput{}, fmt.Errorf("name is required")
	}

	if err := chroma.DeleteCollection(ctx, in.Name); err != nil {
		return CollectionDeleteOutput{}, fmt.Errorf("delete collection failed: %v", err)
	}

	return CollectionDeleteOutput{Deleted: true, Name: in.Name}, nil
}

func handleCollectionStats(ctx context.Context, chroma chromaClient, in CollectionStatsInput) (CollectionStatsOutput, error) {
	if in.CollectionID == "" {
		return CollectionStatsOutput{}, fmt.Errorf("collection_id is required")
	}

	resolvedID, err := chroma.ResolveCollectionID(ctx, in.CollectionID)
	if err != nil {
		return CollectionStatsOutput{}, fmt.Errorf("failed to resolve collection: %v", err)
	}

	records, err := chroma.ListDocuments(ctx, resolvedID)
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

func handleForget(ctx context.Context, chroma chromaClient, in ForgetInput) (ForgetOutput, error) {
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
		records, err := chroma.ListDocuments(ctx, resolvedID)
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

	if err := chroma.DeleteRecords(ctx, resolvedID, idsToDelete); err != nil {
		return ForgetOutput{}, fmt.Errorf("delete records failed: %v", err)
	}

	mode := "ids"
	if in.All {
		mode = "all"
	}

	return ForgetOutput{DeletedCount: len(idsToDelete), Mode: mode}, nil
}
