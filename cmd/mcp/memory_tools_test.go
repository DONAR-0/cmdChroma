package main

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"

	client "github.com/DONAR-0/cmdChroma/internal/client"
)

func callStoreMemory(t *testing.T, chroma chromaClient, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	ms := mcptest.NewUnstartedServer(t)
	ms.AddTool(mcp.NewTool("store_memory",
		mcp.WithInputSchema[StoreMemoryInput](),
		mcp.WithOutputSchema[StoreMemoryOutput](),
	), mcp.NewTypedToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args StoreMemoryInput) (*mcp.CallToolResult, error) {
			out, err := handleStoreMemory(ctx, chroma, args)
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
			Name:      "store_memory",
			Arguments: args,
		},
	})
}

func callSearchMemories(t *testing.T, chroma chromaClient, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	ms := mcptest.NewUnstartedServer(t)
	ms.AddTool(mcp.NewTool("search_memories",
		mcp.WithInputSchema[SearchMemoriesInput](),
		mcp.WithOutputSchema[SearchMemoriesOutput](),
	), mcp.NewTypedToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args SearchMemoriesInput) (*mcp.CallToolResult, error) {
			out, err := handleSearchMemories(ctx, chroma, args)
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
			Name:      "search_memories",
			Arguments: args,
		},
	})
}

func callStoreCodeSnippet(t *testing.T, chroma chromaClient, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	ms := mcptest.NewUnstartedServer(t)
	ms.AddTool(mcp.NewTool("store_code_snippet",
		mcp.WithInputSchema[StoreCodeSnippetInput](),
		mcp.WithOutputSchema[StoreCodeSnippetOutput](),
	), mcp.NewTypedToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args StoreCodeSnippetInput) (*mcp.CallToolResult, error) {
			out, err := handleStoreCodeSnippet(ctx, chroma, args)
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
			Name:      "store_code_snippet",
			Arguments: args,
		},
	})
}

func callSearchCode(t *testing.T, chroma chromaClient, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	ms := mcptest.NewUnstartedServer(t)
	ms.AddTool(mcp.NewTool("search_code",
		mcp.WithInputSchema[SearchCodeInput](),
		mcp.WithOutputSchema[SearchCodeOutput](),
	), mcp.NewTypedToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args SearchCodeInput) (*mcp.CallToolResult, error) {
			out, err := handleSearchCode(ctx, chroma, args)
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
			Name:      "search_code",
			Arguments: args,
		},
	})
}

func callGetSession(t *testing.T, chroma chromaClient, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	ms := mcptest.NewUnstartedServer(t)
	ms.AddTool(mcp.NewTool("get_session",
		mcp.WithInputSchema[GetSessionInput](),
		mcp.WithOutputSchema[GetSessionOutput](),
	), mcp.NewTypedToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args GetSessionInput) (*mcp.CallToolResult, error) {
			out, err := handleGetSession(ctx, chroma, args)
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
			Name:      "get_session",
			Arguments: args,
		},
	})
}

func firstTextContent(res *mcp.CallToolResult) string {
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}

	return ""
}

func isErrorResult(res *mcp.CallToolResult) bool {
	return res.IsError
}

func parseJSONContent[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()

	raw := firstTextContent(res)
	if raw == "" {
		t.Fatal("result has no text content")
	}

	var val T
	if err := json.Unmarshal([]byte(raw), &val); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	return val
}

// mockErr returns an error that satisfies the error interface.
type mockErr string

func (e mockErr) Error() string { return string(e) }

func TestStoreMemory(t *testing.T) {
	tests := []struct {
		name      string
		chroma    *mockChromaClient
		args      map[string]any
		wantErr   bool
		wantCount int
	}{
		{
			name: "happy path",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
			},
			args: map[string]any{
				"content": "The Go scheduler uses M:N threading",
				"type":    "fact",
				"tags":    []any{"go", "scheduler"},
			},
			wantCount: 1,
		},
		{
			name: "explicit id",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
			},
			args: map[string]any{
				"content": "Use context.Background() in tests",
				"type":    "pattern",
				"id":      "my-id-001",
			},
			wantCount: 1,
		},
		{
			name: "custom collection",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "custom-col-uuid",
			},
			args: map[string]any{
				"content":    "Custom collection memory",
				"collection": "my_knowledge_base",
			},
			wantCount: 1,
		},
		{
			name: "missing content",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
			},
			args: map[string]any{
				"type": "fact",
			},
			wantErr: true,
		},
		{
			name: "resolve collection error",
			chroma: &mockChromaClient{
				ResolveCollectionIDErr: mockErr("collection not found"),
			},
			args: map[string]any{
				"content": "test",
			},
			wantErr: true,
		},
		{
			name: "store error",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
				AddBatchGenericErr:        mockErr("add failed"),
			},
			args: map[string]any{
				"content": "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := callStoreMemory(t, tt.chroma, tt.args)
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}

			if tt.wantErr {
				if !isErrorResult(res) {
					t.Errorf("expected IsError=true, got text=%q", firstTextContent(res))
				}

				return
			}

			if isErrorResult(res) {
				t.Fatalf("unexpected error: %s", firstTextContent(res))
			}

			out := parseJSONContent[StoreMemoryOutput](t, res)
			if out.Count != tt.wantCount {
				t.Errorf("Count = %d, want %d", out.Count, tt.wantCount)
			}

			if out.ID == "" {
				t.Error("ID should not be empty")
			}
		})
	}
}

func TestStoreMemory_Recording(t *testing.T) {
	chroma := &mockChromaClient{ResolveCollectionIDResult: "col-uuid"}

	_, err := callStoreMemory(t, chroma, map[string]any{
		"content": "test memory",
		"type":    "fact",
		"tags":    []any{"tag1", "tag2"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	if len(chroma.AddBatchGenericCalls) != 1 {
		t.Fatalf("AddBatchGenericCalls = %d, want 1", len(chroma.AddBatchGenericCalls))
	}

	call := chroma.AddBatchGenericCalls[0]
	if len(call.Documents) != 1 || call.Documents[0] != "test memory" {
		t.Errorf("Documents = %v, want [test memory]", call.Documents)
	}

	if len(call.Metadatas) != 1 {
		t.Fatalf("Metadatas = %d, want 1", len(call.Metadatas))
	}

	if call.Metadatas[0]["type"] != "fact" {
		t.Errorf("type metadata = %v, want fact", call.Metadatas[0]["type"])
	}
}

func TestSearchMemories(t *testing.T) {
	tests := []struct {
		name    string
		chroma  *mockChromaClient
		args    map[string]any
		wantErr bool
		wantN   int
	}{
		{
			name: "happy path",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
				QueryResult: &client.QueryResponse{
					IDs:       [][]string{{"mem-1", "mem-2"}},
					Documents: [][]string{{"doc1", "doc2"}},
					Metadatas: [][]map[string]any{{{}, {}}},
					Distances: [][]float32{{0.1, 0.2}},
				},
			},
			args:  map[string]any{"query": "how does Go schedule?"},
			wantN: 2,
		},
		{
			name: "missing query",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
			},
			args:    map[string]any{},
			wantErr: true,
		},
		{
			name: "empty result set",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
				QueryResult: &client.QueryResponse{
					IDs:       [][]string{{}},
					Documents: [][]string{{}},
					Metadatas: [][]map[string]any{{{}}},
					Distances: [][]float32{{}},
				},
			},
			args:  map[string]any{"query": "nothing"},
			wantN: 0,
		},
		{
			name: "filter by type",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
				QueryResult: &client.QueryResponse{
					IDs:       [][]string{{"mem-1", "mem-2"}},
					Documents: [][]string{{"decision text", "fact text"}},
					Metadatas: [][]map[string]any{{{"type": "decision"}, {"type": "fact"}}},
					Distances: [][]float32{{0.1, 0.2}},
				},
			},
			args:  map[string]any{"query": "test", "filter_type": "fact"},
			wantN: 1,
		},
		{
			name: "query error",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
				QueryErr:                  mockErr("query failed"),
			},
			args:    map[string]any{"query": "test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := callSearchMemories(t, tt.chroma, tt.args)
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}

			if tt.wantErr {
				if !isErrorResult(res) {
					t.Errorf("expected IsError=true, got text=%q", firstTextContent(res))
				}

				return
			}

			if isErrorResult(res) {
				t.Fatalf("unexpected error: %s", firstTextContent(res))
			}

			out := parseJSONContent[SearchMemoriesOutput](t, res)
			if len(out.Results) != tt.wantN {
				t.Errorf("len(Results) = %d, want %d", len(out.Results), tt.wantN)
			}
		})
	}
}

func TestStoreCodeSnippet(t *testing.T) {
	tests := []struct {
		name    string
		chroma  *mockChromaClient
		args    map[string]any
		wantErr bool
	}{
		{
			name: "happy path",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
			},
			args: map[string]any{
				"code":        "func main() {}",
				"language":    "go",
				"description": "Entry point",
			},
		},
		{
			name: "missing code",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
			},
			args: map[string]any{
				"language": "go",
			},
			wantErr: true,
		},
		{
			name: "with tags",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
			},
			args: map[string]any{
				"code": "import \"fmt\"",
				"tags": []any{"import", "fmt"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := callStoreCodeSnippet(t, tt.chroma, tt.args)
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}

			if tt.wantErr {
				if !isErrorResult(res) {
					t.Errorf("expected IsError=true, got text=%q", firstTextContent(res))
				}

				return
			}

			if isErrorResult(res) {
				t.Fatalf("unexpected error: %s", firstTextContent(res))
			}

			out := parseJSONContent[StoreCodeSnippetOutput](t, res)
			if out.Count != 1 {
				t.Errorf("Count = %d, want 1", out.Count)
			}

			if out.ID == "" {
				t.Error("ID should not be empty")
			}
		})
	}
}

func TestSearchCode(t *testing.T) {
	tests := []struct {
		name    string
		chroma  *mockChromaClient
		args    map[string]any
		wantErr bool
		wantN   int
	}{
		{
			name: "happy path",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
				QueryResult: &client.QueryResponse{
					IDs:       [][]string{{"snippet-1", "snippet-2"}},
					Documents: [][]string{{"func foo() {}", "func bar() {}"}},
					Metadatas: [][]map[string]any{{{}, {}}},
					Distances: [][]float32{{0.1, 0.2}},
				},
			},
			args:  map[string]any{"query": "find function"},
			wantN: 2,
		},
		{
			name: "missing query",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
			},
			args:    map[string]any{},
			wantErr: true,
		},
		{
			name: "filter by language",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
				QueryResult: &client.QueryResponse{
					IDs:       [][]string{{"s-1", "s-2"}},
					Documents: [][]string{{"python code", "go code"}},
					Metadatas: [][]map[string]any{{{"language": "python"}, {"language": "go"}}},
					Distances: [][]float32{{0.1, 0.2}},
				},
			},
			args:  map[string]any{"query": "code", "language": "go"},
			wantN: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := callSearchCode(t, tt.chroma, tt.args)
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}

			if tt.wantErr {
				if !isErrorResult(res) {
					t.Errorf("expected IsError=true, got text=%q", firstTextContent(res))
				}

				return
			}

			if isErrorResult(res) {
				t.Fatalf("unexpected error: %s", firstTextContent(res))
			}

			out := parseJSONContent[SearchCodeOutput](t, res)
			if len(out.Results) != tt.wantN {
				t.Errorf("len(Results) = %d, want %d", len(out.Results), tt.wantN)
			}
		})
	}
}

func TestGetSession(t *testing.T) {
	tests := []struct {
		name    string
		chroma  *mockChromaClient
		args    map[string]any
		wantErr bool
	}{
		{
			name: "happy path",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
				ListDocsResult: &client.GetRecordsResponse{
					IDs:       []string{"session-1", "session-2"},
					Documents: []string{"session content", "other"},
					Metadatas: []map[string]any{{"tags": []any{"important"}}, {}},
				},
			},
			args: map[string]any{"id": "session-1"},
		},
		{
			name: "missing id",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
			},
			args:    map[string]any{},
			wantErr: true,
		},
		{
			name: "not found",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid",
				ListDocsResult: &client.GetRecordsResponse{
					IDs:       []string{"other-id"},
					Documents: []string{"other"},
					Metadatas: []map[string]any{{}},
				},
			},
			args:    map[string]any{"id": "missing-id"},
			wantErr: true,
		},
		{
			name: "resolve error",
			chroma: &mockChromaClient{
				ResolveCollectionIDErr: mockErr("bad collection"),
			},
			args:    map[string]any{"id": "session-1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := callGetSession(t, tt.chroma, tt.args)
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}

			if tt.wantErr {
				if !isErrorResult(res) {
					t.Errorf("expected IsError=true, got text=%q", firstTextContent(res))
				}

				return
			}

			if isErrorResult(res) {
				t.Fatalf("unexpected error: %s", firstTextContent(res))
			}

			out := parseJSONContent[GetSessionOutput](t, res)
			if out.ID != "session-1" {
				t.Errorf("ID = %q, want session-1", out.ID)
			}

			if out.Content == "" {
				t.Error("Content should not be empty")
			}
		})
	}
}

func TestStoreMemory_Concurrent(t *testing.T) {
	cm := &concurrentMockSession{}
	cm.ResolveCollectionIDResult = "col-uuid"

	var wg sync.WaitGroup

	errCh := make(chan error, 50)

	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := handleStoreMemory(context.Background(), cm, StoreMemoryInput{
				Content: "test",
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

type concurrentMockSession struct {
	mockChromaClient
}

func (m *concurrentMockSession) ResolveCollectionID(_ context.Context, input string) (string, error) {
	return "col-uuid-conc", nil
}

func (m *concurrentMockSession) AddBatchGeneric(_ context.Context, _ string, _ []string, _ []string, _ []map[string]any) error {
	return nil
}

func TestMemory_SchemaPresence(t *testing.T) {
	tools := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"store_memory", func(t *testing.T) {
			ms := mcptest.NewUnstartedServer(t)
			ms.AddServerOptions(server.WithToolCapabilities(true))
			ms.AddTool(mcp.NewTool("store_memory",
				mcp.WithInputSchema[StoreMemoryInput](),
				mcp.WithOutputSchema[StoreMemoryOutput](),
			), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return nil, nil
			})

			if err := ms.Start(t.Context()); err != nil {
				t.Fatalf("mcptest start: %v", err)
			}

			t.Cleanup(ms.Close)

			tools, err := ms.Client().ListTools(t.Context(), mcp.ListToolsRequest{})
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}

			for _, tool := range tools.Tools {
				if tool.Name == "store_memory" {
					raw, _ := json.Marshal(tool.InputSchema)
					if !json.Valid(raw) {
						t.Errorf("store_memory InputSchema not valid JSON: %s", raw)
					}

					return
				}
			}

			t.Error("store_memory not found in tools/list")
		}},
		{"search_memories", func(t *testing.T) {
			ms := mcptest.NewUnstartedServer(t)
			ms.AddServerOptions(server.WithToolCapabilities(true))
			ms.AddTool(mcp.NewTool("search_memories",
				mcp.WithInputSchema[SearchMemoriesInput](),
				mcp.WithOutputSchema[SearchMemoriesOutput](),
			), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return nil, nil
			})

			if err := ms.Start(t.Context()); err != nil {
				t.Fatalf("mcptest start: %v", err)
			}

			t.Cleanup(ms.Close)

			tools, err := ms.Client().ListTools(t.Context(), mcp.ListToolsRequest{})
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}

			for _, tool := range tools.Tools {
				if tool.Name == "search_memories" {
					raw, _ := json.Marshal(tool.InputSchema)
					if !json.Valid(raw) {
						t.Errorf("search_memories InputSchema not valid JSON: %s", raw)
					}

					return
				}
			}

			t.Error("search_memories not found in tools/list")
		}},
		{"store_code_snippet", func(t *testing.T) {
			ms := mcptest.NewUnstartedServer(t)
			ms.AddServerOptions(server.WithToolCapabilities(true))
			ms.AddTool(mcp.NewTool("store_code_snippet",
				mcp.WithInputSchema[StoreCodeSnippetInput](),
				mcp.WithOutputSchema[StoreCodeSnippetOutput](),
			), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return nil, nil
			})

			if err := ms.Start(t.Context()); err != nil {
				t.Fatalf("mcptest start: %v", err)
			}

			t.Cleanup(ms.Close)

			tools, err := ms.Client().ListTools(t.Context(), mcp.ListToolsRequest{})
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}

			for _, tool := range tools.Tools {
				if tool.Name == "store_code_snippet" {
					raw, _ := json.Marshal(tool.InputSchema)
					if !json.Valid(raw) {
						t.Errorf("store_code_snippet InputSchema not valid JSON: %s", raw)
					}

					return
				}
			}

			t.Error("store_code_snippet not found in tools/list")
		}},
		{"search_code", func(t *testing.T) {
			ms := mcptest.NewUnstartedServer(t)
			ms.AddServerOptions(server.WithToolCapabilities(true))
			ms.AddTool(mcp.NewTool("search_code",
				mcp.WithInputSchema[SearchCodeInput](),
				mcp.WithOutputSchema[SearchCodeOutput](),
			), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return nil, nil
			})

			if err := ms.Start(t.Context()); err != nil {
				t.Fatalf("mcptest start: %v", err)
			}

			t.Cleanup(ms.Close)

			tools, err := ms.Client().ListTools(t.Context(), mcp.ListToolsRequest{})
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}

			for _, tool := range tools.Tools {
				if tool.Name == "search_code" {
					raw, _ := json.Marshal(tool.InputSchema)
					if !json.Valid(raw) {
						t.Errorf("search_code InputSchema not valid JSON: %s", raw)
					}

					return
				}
			}

			t.Error("search_code not found in tools/list")
		}},
		{"get_session", func(t *testing.T) {
			ms := mcptest.NewUnstartedServer(t)
			ms.AddServerOptions(server.WithToolCapabilities(true))
			ms.AddTool(mcp.NewTool("get_session",
				mcp.WithInputSchema[GetSessionInput](),
				mcp.WithOutputSchema[GetSessionOutput](),
			), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return nil, nil
			})

			if err := ms.Start(t.Context()); err != nil {
				t.Fatalf("mcptest start: %v", err)
			}

			t.Cleanup(ms.Close)

			tools, err := ms.Client().ListTools(t.Context(), mcp.ListToolsRequest{})
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}

			for _, tool := range tools.Tools {
				if tool.Name == "get_session" {
					raw, _ := json.Marshal(tool.InputSchema)
					if !json.Valid(raw) {
						t.Errorf("get_session InputSchema not valid JSON: %s", raw)
					}

					return
				}
			}

			t.Error("get_session not found in tools/list")
		}},
	}

	for _, tt := range tools {
		t.Run(tt.name, tt.fn)
	}
}

func TestMemoryTools_ListedInMemoryMode(t *testing.T) {
	srv := buildServer(&mockChromaClient{}, "memory")

	ms := mcptest.NewUnstartedServer(t)
	ms.AddServerOptions(server.WithToolCapabilities(true))

	for _, st := range srv.ListTools() {
		ms.AddTool(st.Tool, st.Handler)
	}

	if err := ms.Start(t.Context()); err != nil {
		t.Fatalf("mcptest server start: %v", err)
	}

	t.Cleanup(ms.Close)

	tools, err := ms.Client().ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools round-trip failed: %v", err)
	}

	memoryNames := []string{"store_memory", "search_memories", "store_code_snippet", "search_code", "get_session"}

	toolNames := map[string]bool{}
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}

	for _, name := range memoryNames {
		if !toolNames[name] {
			t.Errorf("%s not found in tools/list when mode=memory", name)
		}
	}
}

func TestMemoryTools_NotListedInGenericMode(t *testing.T) {
	srv := buildServer(&mockChromaClient{}, "")

	ms := mcptest.NewUnstartedServer(t)
	ms.AddServerOptions(server.WithToolCapabilities(true))

	for _, st := range srv.ListTools() {
		ms.AddTool(st.Tool, st.Handler)
	}

	if err := ms.Start(t.Context()); err != nil {
		t.Fatalf("mcptest server start: %v", err)
	}

	t.Cleanup(ms.Close)

	tools, err := ms.Client().ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools round-trip failed: %v", err)
	}

	memoryNames := []string{"store_memory", "search_memories", "store_code_snippet", "search_code", "get_session"}

	toolNames := map[string]bool{}
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}

	for _, name := range memoryNames {
		if toolNames[name] {
			t.Errorf("%s should NOT be in tools/list when mode=\"\"", name)
		}
	}
}
