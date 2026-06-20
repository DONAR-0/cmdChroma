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

	client "github.com/DONAR-0/cmdChroma/internal/client"
)

func callQueryDocuments(t *testing.T, chroma client.ChromaClientInterface, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	ms := mcptest.NewUnstartedServer(t)
	queryTool := mcp.NewTool("query_documents",
		mcp.WithDescription("Search semantically across stored documents in a collection"),
		mcp.WithInputSchema[QueryDocumentsInput](),
		mcp.WithOutputSchema[QueryDocumentsOutput](),
	)
	ms.AddTool(queryTool, mcp.NewTypedToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args QueryDocumentsInput) (*mcp.CallToolResult, error) {
			out, err := handleQueryDocuments(ctx, chroma, args)
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
			Name:      "query_documents",
			Arguments: args,
		},
	})
}

func TestQueryDocuments(t *testing.T) {
	happyResp := &client.QueryResponse{
		IDs:       [][]string{{"id1", "id2"}},
		Documents: [][]string{{"doc a", "doc b"}},
		Metadatas: [][]map[string]any{{{}, {}}},
		Distances: [][]float32{{0.1, 0.2}},
	}

	cases := []struct {
		name    string
		chroma  *mockChromaClient
		args    map[string]any
		wantErr bool
		assert  func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult)
	}{
		{
			name: "happy_path",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-q",
				QueryResult:               happyResp,
			},
			args: map[string]any{
				"collection_id": "test_col",
				"query_texts":   []any{"hello"},
				"n_results":     5,
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[QueryDocumentsOutput](t, res)
				if len(out.IDs) != 1 {
					t.Errorf("len(IDs) = %d, want 1", len(out.IDs))
				}

				if len(out.Distances[0]) != 2 {
					t.Errorf("len(Distances[0]) = %d, want 2", len(out.Distances[0]))
				}

				if out.DurationMs < 0 {
					t.Error("DurationMs should be >= 0")
				}

				if len(chroma.QueryCalls) != 1 {
					t.Errorf("QueryCalls = %d, want 1", len(chroma.QueryCalls))
				}
			},
		},
		{
			name: "missing_collection_id",
			chroma: &mockChromaClient{
				QueryResult: happyResp,
			},
			args: map[string]any{
				"query_texts": []any{"hello"},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name: "empty_query_texts",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-q",
			},
			args: map[string]any{
				"collection_id": "test_col",
				"query_texts":   []any{},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name: "n_results_default_when_zero",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-q",
				QueryResult:               happyResp,
			},
			args: map[string]any{
				"collection_id": "test_col",
				"query_texts":   []any{"hello"},
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if len(chroma.QueryCalls) != 1 {
					t.Fatalf("QueryCalls = %d, want 1", len(chroma.QueryCalls))
				}

				if chroma.QueryCalls[0].NResults != defaultNResults {
					t.Errorf("NResults = %d, want %d", chroma.QueryCalls[0].NResults, defaultNResults)
				}
			},
		},
		{
			name: "n_results_clamped_to_max",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-q",
				QueryResult:               happyResp,
			},
			args: map[string]any{
				"collection_id": "test_col",
				"query_texts":   []any{"hello"},
				"n_results":     9999,
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if len(chroma.QueryCalls) != 1 {
					t.Fatalf("QueryCalls = %d, want 1", len(chroma.QueryCalls))
				}

				if chroma.QueryCalls[0].NResults != maxQueryResults {
					t.Errorf("NResults = %d, want %d", chroma.QueryCalls[0].NResults, maxQueryResults)
				}
			},
		},
		{
			name: "resolve_collection_error",
			chroma: &mockChromaClient{
				ResolveCollectionIDErr: fmt.Errorf("collection not found"),
			},
			args: map[string]any{
				"collection_id": "missing",
				"query_texts":   []any{"hello"},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name: "query_batch_error",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-q",
				QueryErr:                  fmt.Errorf("chroma query error"),
			},
			args: map[string]any{
				"collection_id": "test_col",
				"query_texts":   []any{"hello"},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name: "empty_result_set",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-q",
				QueryResult: &client.QueryResponse{
					IDs:       [][]string{{}},
					Documents: [][]string{{}},
					Distances: [][]float32{{}},
				},
			},
			args: map[string]any{
				"collection_id": "test_col",
				"query_texts":   []any{"nothing"},
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[QueryDocumentsOutput](t, res)
				if len(out.IDs[0]) != 0 {
					t.Errorf("len(out.IDs[0]) = %d, want 0", len(out.IDs[0]))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := callQueryDocuments(t, tc.chroma, tc.args)
			if err != nil {
				t.Fatalf("CallTool error: %v", err)
			}

			tc.assert(t, tc.chroma, res)
		})
	}
}

func TestQueryDocuments_Concurrent(t *testing.T) {
	cm := &concurrentQueryMock{}
	cm.ResolveCollectionIDResult = "col-uuid-q"
	cm.QueryResult = &client.QueryResponse{
		IDs:       [][]string{{"id1"}},
		Documents: [][]string{{"doc a"}},
		Distances: [][]float32{{0.1}},
	}

	var wg sync.WaitGroup

	errCh := make(chan error, 50)

	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := handleQueryDocuments(context.Background(), cm, QueryDocumentsInput{
				CollectionID: "test_col",
				QueryTexts:   []string{"hello"},
				NResults:     5,
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

type concurrentQueryMock struct {
	mockChromaClient
}

func (m *concurrentQueryMock) ResolveCollectionID(_ context.Context, input string) (string, error) {
	return "col-uuid-q", nil
}

func (m *concurrentQueryMock) QueryBatch(_ context.Context, _ string, _ []string, _ int) (*client.QueryResponse, error) {
	return &client.QueryResponse{
		IDs:       [][]string{{"id1"}},
		Documents: [][]string{{"doc a"}},
		Distances: [][]float32{{0.1}},
	}, nil
}

func TestQueryDocuments_SchemaPresence(t *testing.T) {
	ms := mcptest.NewUnstartedServer(t)
	ms.AddServerOptions(server.WithToolCapabilities(true))
	ms.AddTool(mcp.NewTool("query_documents",
		mcp.WithDescription("Search semantically across stored documents in a collection"),
		mcp.WithInputSchema[QueryDocumentsInput](),
		mcp.WithOutputSchema[QueryDocumentsOutput](),
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
		if tool.Name == "query_documents" {
			found = true

			raw, _ := json.Marshal(tool.InputSchema)
			if !json.Valid(raw) {
				t.Errorf("query_documents InputSchema is not valid JSON: %s", raw)
			}
		}
	}

	if !found {
		t.Error("query_documents not found in tools/list")
	}
}
