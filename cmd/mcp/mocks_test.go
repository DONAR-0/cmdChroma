package main

import (
	"context"

	client "github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
)

// =============================================================================
// Recording mocks for the cmdChroma MCP server.
//
// Style follows internal/service/chroma_service_test.go: raw struct mocks with
// canned return values, no testify/mock. Recording is added per testing.md §1.1
// — every method appends a typed call record so tests can assert on call shape
// (which arguments were passed in what order).
//
// Usage in tests:
//
//	chroma := &mockChromaClient{
//	    QueryResult: &client.QueryResponse{ /* canned */ },
//	}
//	runTool(t, "query_documents", ...)
//
//	require.Equal(t, 1, len(chroma.QueryCalls))
//	require.Equal(t, "test_col", chroma.QueryCalls[0].CollectionID)
// =============================================================================

// writeCall groups any add/upsert shape — `AddBatch`, `AddBatchGeneric`, and
// `UpsertBatchGeneric` share the (collectionID, docs, ids, metadatas) tuple.
type writeCall struct {
	CollectionID string
	Documents    []string
	IDs          []string
	Metadatas    []map[string]any
}

type queryCall struct {
	CollectionID string
	QueryTexts   []string
	NResults     int
}

type listDocsCall struct {
	CollectionID string
}

type resolveCollectionIDCall struct {
	Input string
}

type createCollectionCall struct {
	Name string
}

type createDatabaseCall struct {
	Name string
}

type deleteCollectionCall struct {
	Name string
}

type deleteRecordsCall struct {
	CollectionID string
	IDs          []string
}

// noArgCounts groups methods that don't take meaningful arguments; we still
// count invocations so concurrent-test tooling can spot missed calls.
type noArgCounts struct {
	TestConnection  int
	GetTenant       int
	ListDatabases   int
	ListCollections int
	SetEmbedder     int
}

type mockChromaClient struct {
	// Recording tables (parallel slice per method).
	AddBatchCalls            []writeCall
	AddBatchGenericCalls     []writeCall
	UpsertBatchGenericCalls  []writeCall
	QueryCalls               []queryCall
	ListDocsCalls            []listDocsCall
	ResolveCollectionIDCalls []resolveCollectionIDCall
	CreateCollectionCalls    []createCollectionCall
	CreateDatabaseCalls      []createDatabaseCall
	DeleteCollectionCalls    []deleteCollectionCall
	DeleteRecordsCalls       []deleteRecordsCall
	NoArg                    noArgCounts

	// Canned responses / errors per method.
	TestConnectionErr         error
	GetTenantResult           bool
	GetTenantErr              error
	ListDatabasesResult       []client.Database
	ListDatabasesErr          error
	ListCollectionsResult     []client.Collection
	ListCollectionsErr        error
	ListDocsResult            *client.GetRecordsResponse
	ListDocsErr               error
	AddBatchErr               error
	AddBatchGenericErr        error
	UpsertBatchGenericErr     error
	QueryResult               *client.QueryResponse
	QueryErr                  error
	ResolveCollectionIDResult string
	ResolveCollectionIDErr    error
	// CreateCollectionResult defaults to "test-collection-id" when empty.
	CreateCollectionResult string
	CreateCollectionErr    error
	DeleteCollectionErr    error
	DeleteRecordsErr       error
	CreateDatabaseErr      error

	// The last embedder passed to SetEmbedder (if any).
	LastEmbedder *onnx.Embedder
}

func (m *mockChromaClient) TestConnection(_ context.Context) error {
	m.NoArg.TestConnection++
	return m.TestConnectionErr
}

func (m *mockChromaClient) GetTenant(_ context.Context) (bool, error) {
	m.NoArg.GetTenant++
	return m.GetTenantResult, m.GetTenantErr
}

func (m *mockChromaClient) ListDatabases(_ context.Context) ([]client.Database, error) {
	m.NoArg.ListDatabases++
	return m.ListDatabasesResult, m.ListDatabasesErr
}

func (m *mockChromaClient) CreateDatabase(_ context.Context, name string) error {
	m.CreateDatabaseCalls = append(m.CreateDatabaseCalls, createDatabaseCall{Name: name})
	return m.CreateDatabaseErr
}

func (m *mockChromaClient) ListCollections(_ context.Context) ([]client.Collection, error) {
	m.NoArg.ListCollections++
	return m.ListCollectionsResult, m.ListCollectionsErr
}

func (m *mockChromaClient) CountDocuments(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *mockChromaClient) CreateCollection(_ context.Context, name string) (string, error) {
	m.CreateCollectionCalls = append(m.CreateCollectionCalls, createCollectionCall{Name: name})

	id := m.CreateCollectionResult
	if id == "" {
		id = "test-collection-id"
	}

	return id, m.CreateCollectionErr
}

func (m *mockChromaClient) AddBatch(ctx context.Context, collectionID string, documents []string, ids []string) error {
	m.AddBatchCalls = append(m.AddBatchCalls, writeCall{
		CollectionID: collectionID,
		Documents:    documents,
		IDs:          ids,
	})

	return m.AddBatchErr
}

func (m *mockChromaClient) AddBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	m.AddBatchGenericCalls = append(m.AddBatchGenericCalls, writeCall{
		CollectionID: collectionID,
		Documents:    documents,
		IDs:          ids,
		Metadatas:    metadatas,
	})

	return m.AddBatchGenericErr
}

func (m *mockChromaClient) UpsertBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	m.UpsertBatchGenericCalls = append(m.UpsertBatchGenericCalls, writeCall{
		CollectionID: collectionID,
		Documents:    documents,
		IDs:          ids,
		Metadatas:    metadatas,
	})

	return m.UpsertBatchGenericErr
}

func (m *mockChromaClient) QueryBatch(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*client.QueryResponse, error) {
	m.QueryCalls = append(m.QueryCalls, queryCall{
		CollectionID: collectionID,
		QueryTexts:   queryTexts,
		NResults:     nResults,
	})

	return m.QueryResult, m.QueryErr
}

func (m *mockChromaClient) ListDocuments(_ context.Context, collectionID string) (*client.GetRecordsResponse, error) {
	m.ListDocsCalls = append(m.ListDocsCalls, listDocsCall{CollectionID: collectionID})
	return m.ListDocsResult, m.ListDocsErr
}

func (m *mockChromaClient) ResolveCollectionID(_ context.Context, input string) (string, error) {
	m.ResolveCollectionIDCalls = append(m.ResolveCollectionIDCalls, resolveCollectionIDCall{Input: input})
	return m.ResolveCollectionIDResult, m.ResolveCollectionIDErr
}

func (m *mockChromaClient) DeleteCollection(_ context.Context, name string) error {
	m.DeleteCollectionCalls = append(m.DeleteCollectionCalls, deleteCollectionCall{Name: name})
	return m.DeleteCollectionErr
}

func (m *mockChromaClient) DeleteRecords(_ context.Context, collectionID string, ids []string) error {
	m.DeleteRecordsCalls = append(m.DeleteRecordsCalls, deleteRecordsCall{
		CollectionID: collectionID,
		IDs:          ids,
	})

	return m.DeleteRecordsErr
}

func (m *mockChromaClient) SetEmbedder(e *onnx.Embedder) {
	m.NoArg.SetEmbedder++
	m.LastEmbedder = e
}

// -----------------------------------------------------------------------------
// mockChromaClient
// -----------------------------------------------------------------------------
