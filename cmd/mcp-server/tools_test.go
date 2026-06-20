package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
)

// =============================================================================
// T-05 acceptance: an empty-capability server must respond cleanly to the
// JSON-RPC `tools/list` round-trip. This is the contract every future tool
// addition (T-06..T-09) builds on: if this breaks, the rest of the handlers
// cannot be exercised through the SDK's client surface.
//
// NOTE on the mcptest API in mcp-go v0.55: there is no `NewClientServer`
// shorthand — the supported entry point is `UnstartedServer` +
// `AddServerOptions` + `Start` + `Close`. The resulting `*mcptest.Server`
// owns its own internal `*server.MCPServer` (always at version "1.0.0");
// that's why the name/version assertions live alongside the constructor
// tests below rather than inside this round-trip.
// =============================================================================

// TestServer_ToolsList_Empty verifies that a server constructed with tool
// capabilities enabled but zero tools registered responds to the standard
// MCP list method with no error and an empty tool catalog. Hosts like
// Claude Code treat "method-not-found" as a handshake failure, not as an
// empty result — so this empty-list shape is the actual contract.
func TestServer_ToolsList_NonEmpty(t *testing.T) {
	srv := buildServer(&mockChromaClient{}, newTestEmbedder(), "")

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

	if len(tools.Tools) < 7 {
		t.Fatalf("server exposes %d tools, want >= 7 after T-09 registration", len(tools.Tools))
	}

	toolNames := map[string]bool{}
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}

	expected := []string{"store_documents", "query_documents", "collection_list", "collection_create", "collection_delete", "collection_stats", "forget"}
	for _, name := range expected {
		if !toolNames[name] {
			t.Errorf("%s not found in tools/list", name)
		}
	}
}

// TestBuildServer_DoesNotPanic guards against a future refactor slipping
// heavy init (goroutines, network calls, channels) into the constructor.
// buildServer MUST stay cheap and side-effect-free so main.go (T-13) can
// construct one server at startup and tear it down on signal.
func TestBuildServer_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("buildServer panicked: %v", r)
		}
	}()

	s := buildServer(&mockChromaClient{}, newTestEmbedder(), "")

	if s == nil {
		t.Fatal("buildServer returned nil *server.MCPServer")
	}
}

// TestBuildServer_AcceptsAnyIFaceImpl — compile-time + runtime tripwire
// that buildServer accepts the recording mocks. Catches future interface
// drift between `internal/client.ChromaClientInterface`,
// `internal/onnx.EmbedderInterface`, and the constructor signature.
func TestBuildServer_AcceptsAnyIFaceImpl(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("buildServer rejected mock impls: %v", r)
		}
	}()

	_ = buildServer(&mockChromaClient{}, newTestEmbedder(), "")
}

// Annotation matrix for spec compliance (tools.go)
type annotationExpect struct {
	readOnly    bool
	destructive bool
	idempotent  bool
	openWorld   bool
}

// expectedAnnotations specifies the correct hint values for each tool.
// See docs/mcp_server/researchv2.md Appendix A (corrected for openWorldHint=false).
var expectedAnnotations = map[string]annotationExpect{
	// Generic tools
	"store_documents":   {false, true, false, false},
	"query_documents":   {true, false, true, false},
	"collection_list":   {true, false, true, false},
	"collection_create": {false, true, false, false},
	"collection_delete": {false, true, false, false},
	"collection_stats":  {true, false, true, false},
	"forget":            {false, true, false, false},
	// Memory-mode tools
	"store_memory":       {false, true, false, false},
	"search_memories":    {true, false, true, false},
	"store_code_snippet": {false, true, false, false},
	"search_code":        {true, false, true, false},
	"get_session":        {true, false, true, false},
}

// TestAnnotations_MatchMatrix verifies that each tool's annotations match the matrix.
func TestAnnotations_MatchMatrix(t *testing.T) {
	modes := []string{"", "memory"}
	for _, mode := range modes {
		t.Run("mode="+mode, func(t *testing.T) {
			srv := buildServer(&mockChromaClient{}, newTestEmbedder(), mode)

			serverTools := srv.ListTools()
			for _, st := range serverTools {
				tool := st.Tool

				exp, ok := expectedAnnotations[tool.Name]
				if !ok {
					t.Errorf("tool %q not in annotation matrix", tool.Name)
					continue
				}
				// Note: tool.Annotations is a struct (not a pointer) in mcp-go v0.55.0
				// Each hint field is a *bool.
				if got := tool.Annotations.ReadOnlyHint; *got != exp.readOnly {
					t.Errorf("%s.ReadOnlyHint: want %v, got %v", tool.Name, exp.readOnly, *got)
				}

				if got := tool.Annotations.DestructiveHint; *got != exp.destructive {
					t.Errorf("%s.DestructiveHint: want %v, got %v", tool.Name, exp.destructive, *got)
				}

				if got := tool.Annotations.IdempotentHint; *got != exp.idempotent {
					t.Errorf("%s.IdempotentHint: want %v, got %v", tool.Name, exp.idempotent, *got)
				}

				if got := tool.Annotations.OpenWorldHint; *got != exp.openWorld {
					t.Errorf("%s.OpenWorldHint: want %v, got %v", tool.Name, exp.openWorld, *got)
				}
			}
		})
	}
}

// TestTitles_Present verifies that each tool has a non-empty title matching the expected map.
func TestTitles_Present(t *testing.T) {
	expectedTitles := map[string]string{
		"store_documents":    "Store Documents",
		"query_documents":    "Query Documents",
		"collection_list":    "List Collections",
		"collection_create":  "Create Collection",
		"collection_delete":  "Delete Collection",
		"collection_stats":   "Collection Stats",
		"forget":             "Forget Documents",
		"store_memory":       "Store Memory",
		"search_memories":    "Search Memories",
		"store_code_snippet": "Store Code Snippet",
		"search_code":        "Search Code Snippets",
		"get_session":        "Get Session",
	}

	modes := []string{"", "memory"}
	for _, mode := range modes {
		t.Run("mode="+mode, func(t *testing.T) {
			srv := buildServer(&mockChromaClient{}, newTestEmbedder(), mode)

			serverTools := srv.ListTools()
			for _, st := range serverTools {
				tool := st.Tool

				title := tool.Title
				if title == "" {
					t.Errorf("%s.Title is empty", tool.Name)
					continue
				}

				if expected, ok := expectedTitles[tool.Name]; ok {
					if title != expected {
						t.Errorf("%s.Title: want %q, got %q", tool.Name, expected, title)
					}
				} else {
					t.Errorf("%s.Tool not in expectedTitles map", tool.Name)
				}
			}
		})
	}
}

// TestOutputSchema_NotEmpty verifies that each tool's RawOutputSchema is set and can be unmarshaled to a non-empty JSON object.
func TestOutputSchema_NotEmpty(t *testing.T) {
	modes := []string{"", "memory"}
	for _, mode := range modes {
		t.Run("mode="+mode, func(t *testing.T) {
			srv := buildServer(&mockChromaClient{}, newTestEmbedder(), mode)

			serverTools := srv.ListTools()
			for _, st := range serverTools {
				tool := st.Tool

				raw := tool.RawOutputSchema
				if raw == nil {
					t.Errorf("%s.RawOutputSchema is nil", tool.Name)
					continue
				}

				if len(raw) == 0 {
					t.Errorf("%s.RawOutputSchema is empty", tool.Name)
					continue
				}

				var v map[string]any
				if err := json.Unmarshal(raw, &v); err != nil {
					t.Errorf("%s.RawOutputSchema failed to unmarshal: %v", tool.Name, err)
					continue
				}

				if len(v) == 0 {
					t.Errorf("%s.RawOutputSchema unmarshaled to empty object", tool.Name)
				}
			}
		})
	}
}
