// Package main — JSON-tagged input/output struct definitions per T-04.
//
// Each tool has an Input/Output pair. Field names use snake_case via the json
// tag (matching Claude Code's wire convention) and the jsonschema tag drives
// reflection-based schema generation at registration time (T-05) via
// mark3labs/mcp-go v0.55.0 WithInputSchema[T]() / WithOutputSchema[T]().
//
// This file MUST NOT import mcp-go or internal/client — kept SDK-free so types
// remain testable without CGO and trivially portable across SDK pins.
package main

import (
	"encoding/json"
	"time"
)

// outputJSONSchema returns a raw JSON schema for the given tool name.
// Used with mcp.WithRawOutputSchema to provide structured output.
// Returns nil if no schema is defined for the name.
func outputJSONSchema(name string) json.RawMessage {
	schemas := map[string]json.RawMessage{
		"StoreDocumentsOutput":   json.RawMessage(`{"type":"object","properties":{"count":{"type":"integer"},"ids":{"type":"array","items":{"type":"string"}}},"required":["count","ids"]}`),
		"QueryDocumentsOutput":   json.RawMessage(`{"type":"object","properties":{"ids":{"type":"array","items":{"type":"array","items":{"type":"string"}}},"documents":{"type":"array","items":{"type":"array","items":{"type":"string"}}},"metadatas":{"type":"array"},"distances":{"type":"array","items":{"type":"array","items":{"type":"number"}}},"duration_ms":{"type":"integer"}},"required":["ids","documents","metadatas","distances","duration_ms"]}`),
		"CollectionListOutput":   json.RawMessage(`{"type":"object","properties":{"collections":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"id":{"type":"string"}},"required":["name","id"]}}},"required":["collections"]}`),
		"CollectionCreateOutput": json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id","name"]}`),
		"CollectionDeleteOutput": json.RawMessage(`{"type":"object","properties":{"deleted":{"type":"boolean"},"name":{"type":"string"}},"required":["deleted","name"]}`),
		"CollectionStatsOutput":  json.RawMessage(`{"type":"object","properties":{"collection":{"type":"string"},"count":{"type":"integer"},"sample_ids":{"type":"array","items":{"type":"string"}}},"required":["collection","count"]}`),
		"ForgetOutput":           json.RawMessage(`{"type":"object","properties":{"deleted_count":{"type":"integer"},"mode":{"type":"string"}},"required":["deleted_count","mode"]}`),
		"StoreMemoryOutput":      json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}}}`),
		"SearchMemoriesOutput":   json.RawMessage(`{"type":"object","properties":{"results":{"type":"array","items":{"type":"object"}}}}`),
		"StoreCodeSnippetOutput": json.RawMessage(`{"type":"object","properties":{"id":{"type":"string"}}}`),
		"SearchCodeOutput":       json.RawMessage(`{"type":"object","properties":{"results":{"type":"array","items":{"type":"object"}}}}`),
		"GetSessionOutput":       json.RawMessage(`{"type":"object","properties":{"session":{"type":"object"}}}`),
	}

	raw, ok := schemas[name]
	if !ok {
		return nil
	}

	return raw
}

// -----------------------------------------------------------------------------
// -----------------------------------------------------------------------------
// store_documents — persist docs (+ optional ids/metadatas) into a collection.
// -----------------------------------------------------------------------------

// StoreDocumentsInput is the input shape for the store_documents tool.
//
// collection_id is required; documents is required (non-empty); ids and
// metadatas are optional — when omitted the client layer is responsible for
// generating per-record ids and leaving metadatas nil. All three array fields
// must have equal length when supplied; that constraint is enforced in the
// handler (T-06), not in the schema.
type StoreDocumentsInput struct {
	CollectionID string           `json:"collection_id" jsonschema:"required,description=Target collection name or ID"`
	Documents    []string         `json:"documents" jsonschema:"required,minItems=1,description=Texts to embed and store"`
	IDs          []string         `json:"ids,omitempty" jsonschema:"description=Optional explicit IDs; one per document"`
	Metadatas    []map[string]any `json:"metadatas,omitempty" jsonschema:"description=Optional metadata, parallel to documents"`
}

// StoreDocumentsOutput is returned to the host after a successful store.
type StoreDocumentsOutput struct {
	Count int      `json:"count" jsonschema:"required,description=Number of records written"`
	IDs   []string `json:"ids" jsonschema:"required,description=IDs assigned or echoed back, parallel to documents"`
}

// -----------------------------------------------------------------------------
// query_documents — semantic search across a collection.
// -----------------------------------------------------------------------------

// QueryDocumentsInput is the input shape for query_documents. n_results
// defaults to 5 in the handler when omitted (the json 0 value would otherwise
// mean "give me all", which the upstream API does not honor).
type QueryDocumentsInput struct {
	CollectionID string   `json:"collection_id" jsonschema:"required,description=Target collection name or ID"`
	QueryTexts   []string `json:"query_texts" jsonschema:"required,minItems=1,description=One or more natural-language queries"`
	NResults     int      `json:"n_results,omitempty" jsonschema:"default=5,min=1,max=1000,description=Maximum hits to return per query"`
}

// QueryDocumentsOutput mirrors the internal/client.QueryResponse shape via
// parallel slices — query i's hits are at index i of each field. Distances
// are similarity scores; lower = closer.
type QueryDocumentsOutput struct {
	IDs        [][]string         `json:"ids" jsonschema:"required,description=Result IDs, query-indexed"`
	Documents  [][]string         `json:"documents" jsonschema:"required,description=Result document texts, query-indexed"`
	Metadatas  [][]map[string]any `json:"metadatas" jsonschema:"description=Result metadata, query-indexed"`
	Distances  [][]float64        `json:"distances" jsonschema:"description=Similarity scores per hit, lower=closer"`
	DurationMs int64              `json:"duration_ms" jsonschema:"description=Wall-clock time the request took"`
}

// -----------------------------------------------------------------------------
// collection_list — enumerate available collections.
// -----------------------------------------------------------------------------

// CollectionListInput is empty by design — list operations need no args.
// Empty input structs are legal in MCP JSON schema (it emits {}), and the
// typed-handler adapter handles a zero-value fine.
type CollectionListInput struct{}

// CollectionSummary is one row in the list response.
// Count was previously exposed here but never populated by the handler —
// callers needing a per-collection count now use collection_stats instead.
type CollectionSummary struct {
	Name string `json:"name" jsonschema:"required,description=Human-readable collection name"`
	ID   string `json:"id" jsonschema:"required,description=Stable internal collection ID"`
}

// CollectionListOutput is the response shape for collection_list.
type CollectionListOutput struct {
	Collections []CollectionSummary `json:"collections" jsonschema:"required,description=Collections visible to the configured tenant/database"`
}

// -----------------------------------------------------------------------------
// collection_create — create a fresh empty collection.
// -----------------------------------------------------------------------------

// CollectionCreateInput names the collection to create. Name must be
// non-empty (minLength=1) — empty/whitespace names produce a handler-side
// error before hitting ChromaDB.
type CollectionCreateInput struct {
	Name string `json:"name" jsonschema:"required,minLength=1,maxLength=64,description=Collection name (1-64 chars)"`
}

// CollectionCreateOutput echoes back the created collection identifiers.
type CollectionCreateOutput struct {
	ID   string `json:"id" jsonschema:"required,description=Stable internal collection ID returned by ChromaDB"`
	Name string `json:"name" jsonschema:"required,description=Name as stored"`
}

// -----------------------------------------------------------------------------
// collection_delete — destructive operation, requires explicit name.
// -----------------------------------------------------------------------------

// CollectionDeleteInput names the collection to delete.
type CollectionDeleteInput struct {
	Name string `json:"name" jsonschema:"required,minLength=1,description=Collection name to permanently delete"`
}

// CollectionDeleteOutput confirms whether the deletion actually ran.
type CollectionDeleteOutput struct {
	Deleted bool   `json:"deleted" jsonschema:"required,description=True when ChromaDB acknowledged the request"`
	Name    string `json:"name" jsonschema:"required,description=Echoed name of the deleted collection"`
}

// -----------------------------------------------------------------------------
// collection_stats — return only counts (NOT a doc dump).
// -----------------------------------------------------------------------------

// CollectionStatsInput identifies the collection to summarize.
type CollectionStatsInput struct {
	CollectionID string `json:"collection_id" jsonschema:"required,description=Target collection name or ID"`
}

// CollectionStatsOutput counts records without dumping them. Per
// claude_code_integration.md §2, this avoids the disk-persist fallback for
// large-output tool responses.
type CollectionStatsOutput struct {
	Collection  string    `json:"collection" jsonschema:"required,description=Echoed collection identifier"`
	Count       int       `json:"count" jsonschema:"required,description=Total records currently stored"`
	SampleIDs   []string  `json:"sample_ids" jsonschema:"description=Up to 5 sample IDs (first N), useful for downstream tool calls"`
	LastUpdated time.Time `json:"last_updated,omitempty" jsonschema:"description=Best-effort timestamp; empty when unknown"`
}

// -----------------------------------------------------------------------------
// forget — selective record deletion from a collection.
// -----------------------------------------------------------------------------

// ForgetInput requires either ids[] or all=true (handler-side XOR). All three
// fields are independently optional in the schema so the host can show the
// caller the tradeoff; validation lives in T-09's handler.
type ForgetInput struct {
	CollectionID string   `json:"collection_id" jsonschema:"required,description=Target collection name or ID"`
	IDs          []string `json:"ids,omitempty" jsonschema:"minItems=1,description=Specific record IDs to delete"`
	All          bool     `json:"all,omitempty" jsonschema:"description=Set true to delete every record in the collection"`
}

// ForgetOutput reports how many records were removed.
type ForgetOutput struct {
	DeletedCount int    `json:"deleted_count" jsonschema:"required,description=Records removed by this call"`
	Mode         string `json:"mode" jsonschema:"required,enum=ids|all,description=Which path was used to compute the deletion set"`
}

// -----------------------------------------------------------------------------
// Memory-mode tools (registered only when --mode=memory is set)
// -----------------------------------------------------------------------------

// StoreMemoryInput stores a fact, decision, or pattern with rich metadata.
type StoreMemoryInput struct {
	Content    string   `json:"content" jsonschema:"required,description=The knowledge content"`
	Type       string   `json:"type,omitempty" jsonschema:"enum=decision|error_solution|fact|gotcha|pattern|session|snippet,description=Knowledge type — narrows search filters"`
	Tags       []string `json:"tags,omitempty" jsonschema:"description=Searchable tags"`
	Collection string   `json:"collection,omitempty" jsonschema:"description=Collection name (default: mcp_memory)"`
	ID         string   `json:"id,omitempty" jsonschema:"description=Optional ID (auto-generated if empty)"`
}

type StoreMemoryOutput struct {
	ID    string `json:"id" jsonschema:"required,description=ID of the stored memory"`
	Count int    `json:"count" jsonschema:"required,description=Always 1 for a single store"`
}

// SearchMemoriesInput finds stored knowledge by semantic similarity.
type SearchMemoriesInput struct {
	Query      string `json:"query" jsonschema:"required,description=Natural language search query"`
	NResults   int    `json:"n_results,omitempty" jsonschema:"default=5,min=1,max=100,description=Maximum hits to return"`
	FilterType string `json:"filter_type,omitempty" jsonschema:"description=Optional type filter, e.g. decision|pattern|fact"`
	Collection string `json:"collection,omitempty" jsonschema:"description=Collection name (default: mcp_memory)"`
}

type SearchMemoriesOutput struct {
	Results []MemoryResult `json:"results" jsonschema:"required,description=Matching memories"`
}

type MemoryResult struct {
	ID      string   `json:"id" jsonschema:"required,description=Memory ID"`
	Content string   `json:"content" jsonschema:"required,description=Memory content"`
	Type    string   `json:"type,omitempty" jsonschema:"description=Knowledge type"`
	Tags    []string `json:"tags,omitempty" jsonschema:"description=Associated tags"`
	Score   float64  `json:"score,omitempty" jsonschema:"description=Similarity score (lower=closer)"`
}

// StoreCodeSnippetInput indexes a reusable code snippet.
type StoreCodeSnippetInput struct {
	Code        string   `json:"code" jsonschema:"required,description=The source code to store"`
	Language    string   `json:"language,omitempty" jsonschema:"description=Programming language"`
	Description string   `json:"description,omitempty" jsonschema:"description=Human-readable description"`
	Tags        []string `json:"tags,omitempty" jsonschema:"description=Searchable tags"`
	Collection  string   `json:"collection,omitempty" jsonschema:"description=Collection name (default: mcp_memory)"`
	ID          string   `json:"id,omitempty" jsonschema:"description=Optional ID (auto-generated if empty)"`
}

type StoreCodeSnippetOutput struct {
	ID    string `json:"id" jsonschema:"required,description=ID of the stored snippet"`
	Count int    `json:"count" jsonschema:"required,description=Always 1 for a single store"`
}

// SearchCodeInput finds code snippets by semantic meaning.
type SearchCodeInput struct {
	Query      string `json:"query" jsonschema:"required,description=Semantic search query"`
	NResults   int    `json:"n_results,omitempty" jsonschema:"default=5,min=1,max=100,description=Maximum hits to return"`
	Language   string `json:"language,omitempty" jsonschema:"description=Optional language filter"`
	Collection string `json:"collection,omitempty" jsonschema:"description=Collection name (default: mcp_memory)"`
}

type SearchCodeOutput struct {
	Results []CodeResult `json:"results" jsonschema:"required,description=Matching code snippets"`
}

type CodeResult struct {
	ID          string  `json:"id" jsonschema:"required,description=Snippet ID"`
	Code        string  `json:"code" jsonschema:"required,description=Source code"`
	Language    string  `json:"language,omitempty" jsonschema:"description=Programming language"`
	Description string  `json:"description,omitempty" jsonschema:"description=Snippet description"`
	Score       float64 `json:"score,omitempty" jsonschema:"description=Similarity score (lower=closer)"`
}

// GetSessionInput retrieves a previously saved session by ID.
type GetSessionInput struct {
	ID         string `json:"id" jsonschema:"required,description=Session ID to retrieve"`
	Collection string `json:"collection,omitempty" jsonschema:"description=Collection name (default: mcp_memory)"`
}

type GetSessionOutput struct {
	ID       string         `json:"id" jsonschema:"required,description=Session ID"`
	Content  string         `json:"content" jsonschema:"required,description=Session content"`
	Tags     []string       `json:"tags,omitempty" jsonschema:"description=Associated tags"`
	Metadata map[string]any `json:"metadata,omitempty" jsonschema:"description=Session metadata"`
}
