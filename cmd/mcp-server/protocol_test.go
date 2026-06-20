package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"

	client "github.com/DONAR-0/cmdChroma/internal/client"
)

func TestProtocol_ToolsListRoundTrip(t *testing.T) {
	srv := buildServer(&mockChromaClient{}, newTestEmbedder(), "")

	ms := mcptest.NewUnstartedServer(t)
	ms.AddServerOptions(server.WithToolCapabilities(true))

	for _, st := range srv.ListTools() {
		ms.AddTool(st.Tool, st.Handler)
	}

	if err := ms.Start(t.Context()); err != nil {
		t.Fatalf("mcptest start: %v", err)
	}

	t.Cleanup(ms.Close)

	tools, err := ms.Client().ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	if len(tools.Tools) < 7 {
		t.Fatalf("expected >=7 tools, got %d", len(tools.Tools))
	}
}

func TestProtocol_ToolCallRoundTrip(t *testing.T) {
	chroma := &mockChromaClient{
		ResolveCollectionIDResult: "col-uuid",
	}
	srv := buildServer(chroma, newTestEmbedder(), "memory")

	ms := mcptest.NewUnstartedServer(t)
	ms.AddServerOptions(server.WithToolCapabilities(true))

	for _, st := range srv.ListTools() {
		ms.AddTool(st.Tool, st.Handler)
	}

	if err := ms.Start(t.Context()); err != nil {
		t.Fatalf("mcptest start: %v", err)
	}

	t.Cleanup(ms.Close)

	// Call collection_list (zero-arg tool)
	res, err := ms.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "collection_list",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	if res.IsError {
		t.Fatalf("collection_list returned error: %s", firstTextContent(res))
	}
}

func TestProtocol_StoreAndQueryRoundTrip(t *testing.T) {
	chroma := &mockChromaClient{
		ResolveCollectionIDResult: "col-uuid",
	}
	embed := newTestEmbedder()
	srv := buildServer(chroma, embed, "")

	ms := mcptest.NewUnstartedServer(t)
	ms.AddServerOptions(server.WithToolCapabilities(true))

	for _, st := range srv.ListTools() {
		ms.AddTool(st.Tool, st.Handler)
	}

	if err := ms.Start(t.Context()); err != nil {
		t.Fatalf("mcptest start: %v", err)
	}

	t.Cleanup(ms.Close)

	_, err := ms.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "store_documents",
			Arguments: map[string]any{
				"collection_id": "test",
				"documents":     []any{"hello"},
				"ids":           []any{"id1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	chroma.QueryResult = &client.QueryResponse{
		IDs:       [][]string{{"id1"}},
		Documents: [][]string{{"hello"}},
		Metadatas: [][]map[string]any{{{}}},
		Distances: [][]float32{{0.0}},
	}

	qRes, err := ms.Client().CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "query_documents",
			Arguments: map[string]any{
				"collection_id": "test",
				"query_texts":   []any{"hello"},
				"n_results":     5,
			},
		},
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if qRes.IsError {
		t.Fatalf("query returned error: %s", firstTextContent(qRes))
	}
}

func TestProtocol_Negative_UnknownMethod(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":1,"method":"tools/nope"}`

	code := sendRawJSONRPC(t, raw)
	if code != -32601 {
		t.Errorf("want code -32601 (MethodNotFound), got %d", code)
	}
}

func TestProtocol_Negative_UnknownTool(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"store_doc"}}`

	code := sendRawJSONRPC(t, raw)
	if code != -32602 {
		t.Errorf("want code -32602 (InvalidParams), got %d", code)
	}
}

func TestProtocol_Negative_BadParams(t *testing.T) {
	// params is not an object triggers Invalid Request (-32600) in mcp-go
	raw := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":123}`

	code := sendRawJSONRPC(t, raw)
	if code != -32600 {
		t.Errorf("want code -32600 (InvalidRequest), got %d", code)
	}
}

func sendRawJSONRPC(t *testing.T, requestJSON string) int {
	t.Helper()

	chroma := &mockChromaClient{
		ResolveCollectionIDResult: "col-uuid",
	}
	srv := buildServer(chroma, newTestEmbedder(), "")
	stdioSrv := server.NewStdioServer(srv)

	stdinRead, stdinWrite := io.Pipe()
	stdoutRead, stdoutWrite := io.Pipe()

	errCh := make(chan error, 1)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		errCh <- stdioSrv.Listen(ctx, stdinRead, stdoutWrite)
	}()

	// Write the request
	_, _ = stdinWrite.Write([]byte(requestJSON + "\n"))

	// Read the response (single JSON line)
	var resp struct {
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	}

	dec := json.NewDecoder(stdoutRead)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}

	cancel()

	_ = stdinWrite.Close()

	if resp.Error == nil {
		t.Fatal("expected error in response, got none")
	}

	return resp.Error.Code
}

func TestProtocol_HTTP_RoundTrip(t *testing.T) {
	chroma := &mockChromaClient{
		ResolveCollectionIDResult: "col-uuid",
		QueryResult: &client.QueryResponse{
			IDs:       [][]string{{"id1"}},
			Documents: [][]string{{"hello"}},
			Metadatas: [][]map[string]any{{{}}},
			Distances: [][]float32{{0.0}},
		},
	}
	srv := buildServer(chroma, newTestEmbedder(), "")
	httpSrv := server.NewStreamableHTTPServer(srv)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpSrv.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)

	payload := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"0.0.0"}}}`

	resp, err := ts.Client().Post(ts.URL+"/", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want 200 or 202", resp.StatusCode)
	}

	var initResp struct {
		Result *json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&initResp); err != nil {
		t.Fatalf("json decode: %v", err)
	}

	if initResp.Error != nil {
		t.Fatalf("initialize error: %d %s", initResp.Error.Code, initResp.Error.Message)
	}

	if initResp.Result == nil {
		t.Fatal("initialize: expected result, got nil")
	}
}
