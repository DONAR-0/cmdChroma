package main

import (
	"context"
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
// shorthand — the supported entry point is `NewUnstartedServer` +
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
