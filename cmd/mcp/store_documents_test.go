package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
)

func callStoreDocuments(t *testing.T, chroma chromaClient, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	ms := mcptest.NewUnstartedServer(t)
	storeTool := mcp.NewTool("store_documents",
		mcp.WithDescription("Store text documents as embeddings in a ChromaDB collection"),
		mcp.WithInputSchema[StoreDocumentsInput](),
		mcp.WithOutputSchema[StoreDocumentsOutput](),
	)
	ms.AddTool(storeTool, mcp.NewTypedToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args StoreDocumentsInput) (*mcp.CallToolResult, error) {
			out, err := handleStoreDocuments(ctx, chroma, args)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultStructuredOnly(out), nil
		},
	))

	if err := ms.Start(t.Context()); err != nil {
		t.Fatalf("mcptest server start: %v", err)
	}

	t.Cleanup(ms.Close)

	return ms.Client().CallTool(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "store_documents",
			Arguments: args,
		},
	})
}

func firstText(res *mcp.CallToolResult) string {
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}

	return ""
}

func structuredOutput[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()

	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}

	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}

	return out
}

func TestStoreDocuments(t *testing.T) {
	cases := []struct {
		name    string
		chroma  *mockChromaClient
		args    map[string]any
		wantErr bool
		assert  func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult)
	}{
		{
			name:   "happy_path",
			chroma: &mockChromaClient{ResolveCollectionIDResult: "col-uuid-1"},
			args: map[string]any{
				"collection_id": "test_col",
				"documents":     []any{"hello", "world"},
				"ids":           []any{"id1", "id2"},
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[StoreDocumentsOutput](t, res)
				if out.Count != 2 {
					t.Errorf("Count = %d, want 2", out.Count)
				}

				if len(chroma.AddBatchCalls) != 1 {
					t.Fatalf("AddBatchCalls = %d, want 1", len(chroma.AddBatchCalls))
				}

				call := chroma.AddBatchCalls[0]
				if call.CollectionID != "col-uuid-1" {
					t.Errorf("CollectionID = %q, want %q", call.CollectionID, "col-uuid-1")
				}

				if len(call.Documents) != 2 {
					t.Errorf("len(Documents) = %d, want 2", len(call.Documents))
				}
			},
		},
		{
			name:   "with_metadatas",
			chroma: &mockChromaClient{ResolveCollectionIDResult: "col-uuid-2"},
			args: map[string]any{
				"collection_id": "test_col",
				"documents":     []any{"doc1"},
				"ids":           []any{"id1"},
				"metadatas":     []any{map[string]any{"source": "test"}},
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				if len(chroma.AddBatchGenericCalls) != 1 {
					t.Fatalf("AddBatchGenericCalls = %d, want 1", len(chroma.AddBatchGenericCalls))
				}

				call := chroma.AddBatchGenericCalls[0]
				if len(call.Metadatas) != 1 {
					t.Errorf("len(Metadatas) = %d, want 1", len(call.Metadatas))
				}
			},
		},
		{
			name:   "auto_generate_ids",
			chroma: &mockChromaClient{ResolveCollectionIDResult: "col-uuid-3"},
			args: map[string]any{
				"collection_id": "test_col",
				"documents":     []any{"doc a", "doc b"},
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[StoreDocumentsOutput](t, res)
				if out.Count != 2 {
					t.Errorf("Count = %d, want 2", out.Count)
				}

				if len(out.IDs) != 2 {
					t.Errorf("len(IDs) = %d, want 2", len(out.IDs))
				}

				for _, id := range out.IDs {
					if len(id) != 21 {
						t.Errorf("auto-generated ID %q has length %d, want 21", id, len(id))
					}
				}
			},
		},
		{
			name:   "missing_collection_id",
			chroma: &mockChromaClient{},
			args: map[string]any{
				"documents": []any{"doc1"},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name:   "empty_documents",
			chroma: &mockChromaClient{},
			args: map[string]any{
				"collection_id": "test_col",
				"documents":     []any{},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name:   "ids_length_mismatch",
			chroma: &mockChromaClient{ResolveCollectionIDResult: "col-uuid-4"},
			args: map[string]any{
				"collection_id": "test_col",
				"documents":     []any{"doc1", "doc2"},
				"ids":           []any{"id1"},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}

				if len(chroma.AddBatchCalls) != 0 {
					t.Errorf("AddBatchCalls = %d, want 0 (should not call ChromaDB on validation failure)", len(chroma.AddBatchCalls))
				}
			},
		},
		{
			name:   "metadatas_length_mismatch",
			chroma: &mockChromaClient{ResolveCollectionIDResult: "col-uuid-5"},
			args: map[string]any{
				"collection_id": "test_col",
				"documents":     []any{"doc1"},
				"metadatas":     []any{map[string]any{"a": 1}, map[string]any{"b": 2}},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name:   "resolve_collection_error",
			chroma: &mockChromaClient{ResolveCollectionIDErr: fmt.Errorf("not found")},
			args: map[string]any{
				"collection_id": "missing_col",
				"documents":     []any{"doc1"},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name:   "add_batch_error",
			chroma: &mockChromaClient{ResolveCollectionIDResult: "col-uuid-6", AddBatchErr: fmt.Errorf("chroma server error")},
			args: map[string]any{
				"collection_id": "test_col",
				"documents":     []any{"doc1"},
				"ids":           []any{"id1"},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := callStoreDocuments(t, tc.chroma, tc.args)
			if err != nil {
				t.Fatalf("CallTool error: %v", err)
			}

			tc.assert(t, tc.chroma, res)
		})
	}
}

func TestStoreDocuments_Concurrent(t *testing.T) {
	cm := &concurrentMock{}
	cm.ResolveCollectionIDResult = "col-uuid-conc"

	var wg sync.WaitGroup

	errCh := make(chan error, 50)

	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := handleStoreDocuments(context.Background(), cm, StoreDocumentsInput{
				CollectionID: "test_col",
				Documents:    []string{"doc"},
				IDs:          []string{"id"},
			})
			if err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent handler error: %v", err)
	}
}

// concurrentMock shadows AddBatch and ResolveCollectionID for
// concurrency-safe testing under 50-goroutine load.
type concurrentMock struct {
	mockChromaClient
}

func (m *concurrentMock) ResolveCollectionID(_ context.Context, input string) (string, error) {
	return "col-uuid-conc", nil
}

func (m *concurrentMock) AddBatch(_ context.Context, _ string, _ []string, _ []string) error {
	return nil
}

func TestStoreDocuments_Batching(t *testing.T) {
	chroma := &mockChromaClient{ResolveCollectionIDResult: "col-uuid-batch"}
	docs := make([]string, maxBatchSize+100)
	ids := make([]string, maxBatchSize+100)

	for i := range docs {
		docs[i] = fmt.Sprintf("doc-%d", i)
		ids[i] = fmt.Sprintf("id-%d", i)
	}

	out, err := handleStoreDocuments(context.Background(), chroma, StoreDocumentsInput{
		CollectionID: "test_col",
		Documents:    docs,
		IDs:          ids,
	})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if out.Count != maxBatchSize+100 {
		t.Errorf("Count = %d, want %d", out.Count, maxBatchSize+100)
	}

	if len(chroma.AddBatchCalls) != 2 {
		t.Fatalf("AddBatchCalls = %d, want 2 (batched)", len(chroma.AddBatchCalls))
	}

	if len(chroma.AddBatchCalls[0].Documents) != maxBatchSize {
		t.Errorf("first batch size = %d, want %d", len(chroma.AddBatchCalls[0].Documents), maxBatchSize)
	}

	if len(chroma.AddBatchCalls[1].Documents) != 100 {
		t.Errorf("second batch size = %d, want 100", len(chroma.AddBatchCalls[1].Documents))
	}
}

func TestStoreDocuments_SchemaPresence(t *testing.T) {
	ms := mcptest.NewUnstartedServer(t)
	ms.AddServerOptions(server.WithToolCapabilities(true))
	ms.AddTool(mcp.NewTool("store_documents",
		mcp.WithDescription("Store text documents as embeddings in a ChromaDB collection"),
		mcp.WithInputSchema[StoreDocumentsInput](),
		mcp.WithOutputSchema[StoreDocumentsOutput](),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, nil
	})

	if err := ms.Start(t.Context()); err != nil {
		t.Fatalf("mcptest server start: %v", err)
	}

	t.Cleanup(ms.Close)

	tools, err := ms.Client().ListTools(t.Context(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	found := false

	for _, tool := range tools.Tools {
		if tool.Name == "store_documents" {
			found = true

			raw, _ := json.Marshal(tool.InputSchema)
			if !json.Valid(raw) {
				t.Errorf("store_documents InputSchema is not valid JSON: %s", raw)
			}
		}
	}

	if !found {
		t.Error("store_documents not found in tools/list")
	}
}
