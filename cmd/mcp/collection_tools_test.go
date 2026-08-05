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

func callCollectionTool(t *testing.T, chroma chromaClient, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()

	ms := mcptest.NewUnstartedServer(t)
	ms.AddServerOptions(server.WithToolCapabilities(true))

	switch toolName {
	case "collection_list":
		ms.AddTool(mcp.NewTool("collection_list",
			mcp.WithDescription("List all collections"),
			mcp.WithInputSchema[CollectionListInput](),
			mcp.WithOutputSchema[CollectionListOutput](),
		), mcp.NewTypedToolHandler(
			func(ctx context.Context, req mcp.CallToolRequest, _ CollectionListInput) (*mcp.CallToolResult, error) {
				out, err := handleCollectionList(ctx, chroma)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				return mcp.NewToolResultStructuredOnly(out), nil
			},
		))
	case "collection_create":
		ms.AddTool(mcp.NewTool("collection_create",
			mcp.WithDescription("Create a new collection"),
			mcp.WithInputSchema[CollectionCreateInput](),
			mcp.WithOutputSchema[CollectionCreateOutput](),
		), mcp.NewTypedToolHandler(
			func(ctx context.Context, req mcp.CallToolRequest, args CollectionCreateInput) (*mcp.CallToolResult, error) {
				out, err := handleCollectionCreate(ctx, chroma, args)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				return mcp.NewToolResultStructuredOnly(out), nil
			},
		))
	case "collection_delete":
		ms.AddTool(mcp.NewTool("collection_delete",
			mcp.WithDescription("Delete a collection"),
			mcp.WithInputSchema[CollectionDeleteInput](),
			mcp.WithOutputSchema[CollectionDeleteOutput](),
		), mcp.NewTypedToolHandler(
			func(ctx context.Context, req mcp.CallToolRequest, args CollectionDeleteInput) (*mcp.CallToolResult, error) {
				out, err := handleCollectionDelete(ctx, chroma, args)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				return mcp.NewToolResultStructuredOnly(out), nil
			},
		))
	case "collection_stats":
		ms.AddTool(mcp.NewTool("collection_stats",
			mcp.WithDescription("Get collection stats"),
			mcp.WithInputSchema[CollectionStatsInput](),
			mcp.WithOutputSchema[CollectionStatsOutput](),
		), mcp.NewTypedToolHandler(
			func(ctx context.Context, req mcp.CallToolRequest, args CollectionStatsInput) (*mcp.CallToolResult, error) {
				out, err := handleCollectionStats(ctx, chroma, args)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				return mcp.NewToolResultStructuredOnly(out), nil
			},
		))
	}

	if err := ms.Start(t.Context()); err != nil {
		t.Fatalf("mcptest server start: %v", err)
	}

	t.Cleanup(ms.Close)

	return ms.Client().CallTool(t.Context(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
}

func TestCollectionList(t *testing.T) {
	cases := []struct {
		name    string
		chroma  *mockChromaClient
		wantErr bool
		assert  func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult)
	}{
		{
			name: "happy_path",
			chroma: &mockChromaClient{
				ListCollectionsResult: []client.Collection{
					{Name: "docs", ID: "uuid-1"},
					{Name: "mem", ID: "uuid-2"},
				},
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[CollectionListOutput](t, res)
				if len(out.Collections) != 2 {
					t.Fatalf("len(Collections) = %d, want 2", len(out.Collections))
				}

				if out.Collections[0].Name != "docs" || out.Collections[0].ID != "uuid-1" {
					t.Errorf("first collection mismatch: %+v", out.Collections[0])
				}
			},
		},
		{
			name:    "empty_list",
			chroma:  &mockChromaClient{},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[CollectionListOutput](t, res)
				if len(out.Collections) != 0 {
					t.Errorf("len(Collections) = %d, want 0", len(out.Collections))
				}
			},
		},
		{
			name: "upstream_error",
			chroma: &mockChromaClient{
				ListCollectionsErr: fmt.Errorf("chroma unavailable"),
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
			res, err := callCollectionTool(t, tc.chroma, "collection_list", nil)
			if err != nil {
				t.Fatalf("CallTool error: %v", err)
			}

			tc.assert(t, tc.chroma, res)
		})
	}
}

func TestCollectionCreate(t *testing.T) {
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
				CreateCollectionResult: "new-uuid",
			},
			args: map[string]any{
				"name": "my-collection",
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[CollectionCreateOutput](t, res)
				if out.ID != "new-uuid" {
					t.Errorf("ID = %q, want %q", out.ID, "new-uuid")
				}

				if out.Name != "my-collection" {
					t.Errorf("Name = %q, want %q", out.Name, "my-collection")
				}

				if len(chroma.CreateCollectionCalls) != 1 {
					t.Fatalf("CreateCollectionCalls = %d, want 1", len(chroma.CreateCollectionCalls))
				}

				if chroma.CreateCollectionCalls[0].Name != "my-collection" {
					t.Errorf("Create name = %q, want %q", chroma.CreateCollectionCalls[0].Name, "my-collection")
				}
			},
		},
		{
			name: "empty_name_rejected",
			chroma: &mockChromaClient{
				CreateCollectionResult: "should-not-be-called",
			},
			args: map[string]any{
				"name": "",
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}

				if len(chroma.CreateCollectionCalls) != 0 {
					t.Errorf("CreateCollectionCalls = %d, want 0 (should not call ChromaDB on validation failure)", len(chroma.CreateCollectionCalls))
				}
			},
		},
		{
			name: "upstream_error",
			chroma: &mockChromaClient{
				CreateCollectionErr: fmt.Errorf("duplicate name"),
			},
			args: map[string]any{
				"name": "dup",
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
			res, err := callCollectionTool(t, tc.chroma, "collection_create", tc.args)
			if err != nil {
				t.Fatalf("CallTool error: %v", err)
			}

			tc.assert(t, tc.chroma, res)
		})
	}
}

func TestCollectionDelete(t *testing.T) {
	cases := []struct {
		name    string
		chroma  *mockChromaClient
		args    map[string]any
		wantErr bool
		assert  func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult)
	}{
		{
			name:   "happy_path",
			chroma: &mockChromaClient{},
			args: map[string]any{
				"name": "old-collection",
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[CollectionDeleteOutput](t, res)
				if !out.Deleted {
					t.Error("Deleted = false, want true")
				}

				if out.Name != "old-collection" {
					t.Errorf("Name = %q, want %q", out.Name, "old-collection")
				}

				if len(chroma.DeleteCollectionCalls) != 1 {
					t.Fatalf("DeleteCollectionCalls = %d, want 1", len(chroma.DeleteCollectionCalls))
				}
			},
		},
		{
			name:   "empty_name_rejected",
			chroma: &mockChromaClient{},
			args: map[string]any{
				"name": "",
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}

				if len(chroma.DeleteCollectionCalls) != 0 {
					t.Errorf("DeleteCollectionCalls = %d, want 0", len(chroma.DeleteCollectionCalls))
				}
			},
		},
		{
			name: "upstream_error",
			chroma: &mockChromaClient{
				DeleteCollectionErr: fmt.Errorf("collection not found"),
			},
			args: map[string]any{
				"name": "missing",
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
			res, err := callCollectionTool(t, tc.chroma, "collection_delete", tc.args)
			if err != nil {
				t.Fatalf("CallTool error: %v", err)
			}

			tc.assert(t, tc.chroma, res)
		})
	}
}

func TestCollectionStats(t *testing.T) {
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
				ResolveCollectionIDResult: "col-uuid-stats",
				ListDocsResult: &client.GetRecordsResponse{
					IDs:       []string{"a", "b", "c"},
					Documents: []string{"doc a", "doc b", "doc c"},
				},
			},
			args: map[string]any{
				"collection_id": "test-col",
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[CollectionStatsOutput](t, res)
				if out.Count != 3 {
					t.Errorf("Count = %d, want 3", out.Count)
				}

				if len(out.SampleIDs) != 3 {
					t.Errorf("len(SampleIDs) = %d, want 3", len(out.SampleIDs))
				}
			},
		},
		{
			name: "empty_collection",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "empty-uuid",
				ListDocsResult:            &client.GetRecordsResponse{},
			},
			args: map[string]any{
				"collection_id": "empty",
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if res.IsError {
					t.Fatalf("unexpected error: %s", firstText(res))
				}

				out := structuredOutput[CollectionStatsOutput](t, res)
				if out.Count != 0 {
					t.Errorf("Count = %d, want 0", out.Count)
				}
			},
		},
		{
			name: "sample_ids_truncated_to_5",
			chroma: &mockChromaClient{
				ResolveCollectionIDResult: "big-uuid",
				ListDocsResult: &client.GetRecordsResponse{
					IDs: []string{"a", "b", "c", "d", "e", "f", "g"},
				},
			},
			args: map[string]any{
				"collection_id": "big",
			},
			wantErr: false,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				out := structuredOutput[CollectionStatsOutput](t, res)
				if out.Count != 7 {
					t.Errorf("Count = %d, want 7", out.Count)
				}

				if len(out.SampleIDs) != 5 {
					t.Errorf("len(SampleIDs) = %d, want 5", len(out.SampleIDs))
				}
			},
		},
		{
			name: "missing_collection_id",
			chroma: &mockChromaClient{
				ResolveCollectionIDErr: fmt.Errorf("not found"),
			},
			args: map[string]any{
				"collection_id": "missing",
			},
			wantErr: true,
			assert: func(t *testing.T, chroma *mockChromaClient, res *mcp.CallToolResult) {
				if !res.IsError {
					t.Fatal("expected IsError=true")
				}
			},
		},
		{
			name:   "empty_collection_id",
			chroma: &mockChromaClient{},
			args: map[string]any{
				"collection_id": "",
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
			res, err := callCollectionTool(t, tc.chroma, "collection_stats", tc.args)
			if err != nil {
				t.Fatalf("CallTool error: %v", err)
			}

			tc.assert(t, tc.chroma, res)
		})
	}
}

func TestCollection_SchemaPresence(t *testing.T) {
	type schemaCase struct {
		name    string
		factory func() mcp.Tool
	}

	cases := []schemaCase{
		{"collection_list", func() mcp.Tool {
			return mcp.NewTool("collection_list", mcp.WithInputSchema[CollectionListInput](), mcp.WithOutputSchema[CollectionListOutput]())
		}},
		{"collection_create", func() mcp.Tool {
			return mcp.NewTool("collection_create", mcp.WithInputSchema[CollectionCreateInput](), mcp.WithOutputSchema[CollectionCreateOutput]())
		}},
		{"collection_delete", func() mcp.Tool {
			return mcp.NewTool("collection_delete", mcp.WithInputSchema[CollectionDeleteInput](), mcp.WithOutputSchema[CollectionDeleteOutput]())
		}},
		{"collection_stats", func() mcp.Tool {
			return mcp.NewTool("collection_stats", mcp.WithInputSchema[CollectionStatsInput](), mcp.WithOutputSchema[CollectionStatsOutput]())
		}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ms := mcptest.NewUnstartedServer(t)
			ms.AddServerOptions(server.WithToolCapabilities(true))
			ms.AddTool(tt.factory(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
				if tool.Name == tt.name {
					found = true

					raw, _ := json.Marshal(tool.InputSchema)
					if !json.Valid(raw) {
						t.Errorf("InputSchema not valid JSON: %s", raw)
					}
				}
			}

			if !found {
				t.Errorf("%s not found in tools/list", tt.name)
			}
		})
	}
}
