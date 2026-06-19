package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"

	client "github.com/DONAR-0/cmdChroma/internal/client"
)

func callForget(t *testing.T, chroma client.ChromaClientInterface, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	ms := mcptest.NewUnstartedServer(t)
	ms.AddTool(mcp.NewTool("forget",
		mcp.WithDescription("Delete records from a collection"),
		mcp.WithInputSchema[ForgetInput](),
		mcp.WithOutputSchema[ForgetOutput](),
	), mcp.NewTypedToolHandler(
		func(ctx context.Context, req mcp.CallToolRequest, args ForgetInput) (*mcp.CallToolResult, error) {
			out, err := handleForget(ctx, chroma, args)
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
			Name:      "forget",
			Arguments: args,
		},
	})
}

func TestForget(t *testing.T) {
	cases := []struct {
		name    string
		chroma  *mockChromaClient
		args    map[string]any
		wantErr bool
		assert  func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult)
	}{
		{
			name: "delete_by_ids",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-f",
			},
			args: map[string]any{
				"collection_id": "test_col",
				"ids":           []any{"id1", "id2"},
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[ForgetOutput](t, res)
				if out.DeletedCount != 2 {
					t.Errorf("DeletedCount = %d, want 2", out.DeletedCount)
				}

				if out.Mode != "ids" {
					t.Errorf("Mode = %q, want %q", out.Mode, "ids")
				}

				if len(chroma.DeleteRecordsCalls) != 1 {
					t.Fatalf("DeleteRecordsCalls = %d, want 1", len(chroma.DeleteRecordsCalls))
				}
			},
		},
		{
			name: "delete_all",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-f",
				ListDocsResult: &client.GetRecordsResponse{
					IDs: []string{"a", "b", "c"},
				},
			},
			args: map[string]any{
				"collection_id": "test_col",
				"all":           true,
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[ForgetOutput](t, res)
				if out.DeletedCount != 3 {
					t.Errorf("DeletedCount = %d, want 3", out.DeletedCount)
				}

				if out.Mode != "all" {
					t.Errorf("Mode = %q, want %q", out.Mode, "all")
				}

				if len(chroma.DeleteRecordsCalls) != 1 {
					t.Fatalf("DeleteRecordsCalls = %d, want 1", len(chroma.DeleteRecordsCalls))
				}
			},
		},
		{
			name: "missing_collection_id",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-f",
			},
			args: map[string]any{
				"ids": []any{"id1"},
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name: "missing_ids_and_not_all",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-f",
			},
			args: map[string]any{
				"collection_id": "test_col",
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name: "both_ids_and_all_rejected",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-f",
			},
			args: map[string]any{
				"collection_id": "test_col",
				"ids":           []any{"id1"},
				"all":           true,
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name: "empty_ids_returns_zero",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "col-uuid-f",
				ListDocsResult:            &client.GetRecordsResponse{},
			},
			args: map[string]any{
				"collection_id": "test_col",
				"all":           true,
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[ForgetOutput](t, res)
				if out.DeletedCount != 0 {
					t.Errorf("DeletedCount = %d, want 0", out.DeletedCount)
				}

				if len(chroma.DeleteRecordsCalls) != 0 {
					t.Errorf("DeleteRecordsCalls = %d, want 0", len(chroma.DeleteRecordsCalls))
				}
			},
		},
		{
			name: "resolve_collection_error",
			chroma: &mockChromaClient{
				ResolveCollectionIDErr: fmt.Errorf("not found"),
			},
			args: map[string]any{
				"collection_id": "missing",
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
			res, err := callForget(t, tc.chroma, tc.args)
			if err != nil {
				t.Fatalf("CallTool error: %v", err)
			}

			tc.assert(t, tc.chroma, res)
		})
	}
}

func TestForget_SchemaPresence(t *testing.T) {
	ms := mcptest.NewUnstartedServer(t)
	ms.AddServerOptions(server.WithToolCapabilities(true))
	ms.AddTool(mcp.NewTool("forget",
		mcp.WithInputSchema[ForgetInput](),
		mcp.WithOutputSchema[ForgetOutput](),
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

	found := false

	for _, tool := range tools.Tools {
		if tool.Name == "forget" {
			found = true

			raw, _ := json.Marshal(tool.InputSchema)
			if !json.Valid(raw) {
				t.Errorf("forget InputSchema not valid JSON: %s", raw)
			}
		}
	}

	if !found {
		t.Error("forget not found in tools/list")
	}
}
