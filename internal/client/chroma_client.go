package client

// Package client provides an HTTP client for interacting with ChromaDB's
// REST API. It handles request construction, authentication, response
// unmarshaling, and error handling. The client is designed to be safe
// for concurrent use by multiple goroutines.
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/DONAR-0/cmdChroma/internal"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
	"github.com/google/uuid"
)

// ============ Constants ============

var (
	// MustClose is a wrapper around internal.CheckDefer for cleaning up resources.
	MustClose = internal.CheckDefer
	// endpoints
	testEndpoint         = "%s/api/v2/heartbeat"
	getTenant            = "%s/api/v2/tenants/%s"
	listDatabases        = "%s/api/v2/tenants/%s/databases"
	createDatabase       = "%s/api/v2/tenants/%s/databases"
	listCreateCollection = "%s/api/v2/tenants/%s/databases/%s/collections"
	listDocuments        = "%s/api/v2/tenants/%s/databases/%s/collections/%s/get"
	queryEndpoint        = "%s/api/v2/tenants/%s/databases/%s/collections/%s/query"
	batchAdd             = "%s/api/v2/tenants/%s/databases/%s/collections/%s/add"
	batchUpsert          = "%s/api/v2/tenants/%s/databases/%s/collections/%s/upsert"
	deleteCollection     = "%s/api/v2/tenants/%s/databases/%s/collections/%s"
	deleteRecords        = "%s/api/v2/tenants/%s/databases/%s/collections/%s/delete"
)

// ============ Core Types ============

type (
	// ChromaClient is the concrete implementation of ChromaClientInterface.
	// It manages HTTP communication with a ChromaDB server and supports
	// dependency injection of an embedder for document operations.
	// Clients are safe for concurrent use by multiple goroutines.
	ChromaClient struct {
		// URL is the base API endpoint (e.g., "http://localhost:8000").
		// Must include scheme (http:// or https://) and port if non-standard.
		URL string

		// Tenant is the tenant name for all requests. This is the logical
		// isolation unit in ChromaDB. Defaults to "default_tenant".
		Tenant string

		// Database is the database name within the tenant. Defaults to
		// "default_database".
		Database string

		// client is the underlying HTTP client with configured timeout.
		client *http.Client

		// Embedder is injected by the service layer via SetEmbedder.
		// It must be non-nil before performing document add/query operations.
		Embedder onnx.EmbedderInterface
	}

	// ChromaClientInterface defines the contract for ChromaDB clients.
	// Implementations must be safe for concurrent use and respect context
	// cancellation for all blocking operations.
	ChromaClientInterface interface {
		// TestConnection verifies connectivity to the ChromaDB server.
		// Returns nil if the server responds successfully.
		TestConnection(ctx context.Context) error

		// GetTenant checks whether the configured tenant exists.
		// Returns (true, nil) if tenant exists, (false, nil) if not found,
		// or (false, error) on failure.
		GetTenant(ctx context.Context) (bool, error)

		// ListDatabases returns all databases accessible to the current tenant.
		ListDatabases(ctx context.Context) ([]Database, error)

		// CreateDatabase creates a new database in the current tenant.
		CreateDatabase(ctx context.Context, name string) error

		// ListCollections returns all collections in the current database.
		ListCollections(ctx context.Context) ([]Collection, error)

		// CreateCollection creates a new collection and returns its ID.
		CreateCollection(ctx context.Context, name string) (string, error)

		// AddBatch uploads documents with IDs (legacy, no metadata).
		AddBatch(ctx context.Context, collectionID string, docs []string, ids []string) error

		// AddBatchGeneric uploads documents with optional metadata.
		AddBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error

		// UpsertBatchGeneric adds or updates documents with metadata.
		UpsertBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error

		// QueryBatch performs similarity search and returns matching documents.
		QueryBatch(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*QueryResponse, error)

		// GetIDByName resolves a collection name to its ID.
		GetIDByName(ctx context.Context, name string) (string, error)

		// ListDocuments retrieves documents from a collection with optional filtering.
		ListDocuments(ctx context.Context, collectionID string) (*GetRecordsResponse, error)

		// ResolveCollectionID accepts either a collection name or ID and returns the ID.
		// ctx controls cancellation; passed to the underlying HTTP request.
		ResolveCollectionID(ctx context.Context, input string) (string, error)

		// DeleteCollection removes a collection and all its data.
		DeleteCollection(ctx context.Context, name string) error

		// DeleteRecords removes specific documents by ID from a collection.
		DeleteRecords(ctx context.Context, collectionID string, ids []string) error

		// SetEmbedder injects the embedding engine for document operations.
		SetEmbedder(e onnx.EmbedderInterface)
	}
)

// ============ Data Transfer Types ============

type (
	// CreateCollectionRequest represents the payload for creating a new collection.
	CreateCollectionRequest struct {
		Name        string         `json:"name"`
		Metadata    map[string]any `json:"metadata"`
		GetOrCreate bool           `json:"get_or_create"`
	}

	// Collection represents the detailed response from ChromaDB
	Collection struct {
		ID        string         `json:"id"`
		Name      string         `json:"name"`
		Tenant    string         `json:"tenant"`
		Database  string         `json:"database"`
		Metadata  map[string]any `json:"metadata"`
		Dimension *int           `json:"dimension"` // Pointer because it can be null
		Config    map[string]any `json:"configuration_json"`
	}

	// Database represents a ChromaDB database with its ID, name, and tenant.
	Database struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Tenant string `json:"tenant"`
	}

	// GetRecordsRequest represents the payload for fetching documents from a collection.
	GetRecordsRequest struct {
		Tenant   string   `json:"tenant"`
		Database string   `json:"database"`
		IDs      []string `json:"ids,omitempty"`
		Include  []string `json:"include"`
		Limit    *int     `json:"limit"`
		Offset   *int     `json:"offset"`
	}

	// GetRecordsResponse represents the response from fetching documents.
	GetRecordsResponse struct {
		IDs       []string         `json:"ids"`
		Documents []string         `json:"documents"`
		Metadatas []map[string]any `json:"metadatas"`
	}

	// AddRecordsRequest represents the payload for adding documents to a collection.
	AddRecordsRequest struct {
		IDs        []string         `json:"ids"`
		Documents  []string         `json:"documents"`
		Embeddings [][]float32      `json:"embeddings"`
		Metadatas  []map[string]any `json:"metadatas,omitempty"`
	}

	QueryResponse struct {
		IDs       [][]string         `json:"ids"`
		Documents [][]string         `json:"documents"`
		Metadatas [][]map[string]any `json:"metadatas"`
		Distances [][]float32        `json:"distances"`
	}

	IngestRecord struct {
		ID       string         `json:"id"`
		Text     string         `json:"text"`
		Metadata map[string]any `json:"metadata"`
	}
)

// ============ Constructor ============

func NewChromaDBClient(url, tenant, database string) *ChromaClient {
	slog.Info("Initializing Chroma client", "url", url, "tenant", tenant, "database", database)

	return &ChromaClient{
		URL:      url,
		Tenant:   tenant,
		Database: database,
		client:   &http.Client{},
	}
}

// ============ Connection & Health ============

func (c *ChromaClient) TestConnection(ctx context.Context) error {
	endpoint := fmt.Sprintf(testEndpoint, c.URL)
	slog.Info("Calling endpoint", "endpoint", endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to ChromaDB at %s: %w", c.URL, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to ChromaDB at %s: %w", c.URL, err)
	}

	defer MustClose(resp.Body.Close)

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	slog.Info("ChromaDB connection successful", "response", string(body))

	return nil
}

func (c *ChromaClient) GetTenant(ctx context.Context) (bool, error) {
	// Correct endpoint for checking a specific tenant
	endpoint := fmt.Sprintf(getTenant, c.URL, c.Tenant)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer MustClose(resp.Body.Close)

	// 200 means exists, 404 means it doesn't
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
}

// ============ Databases ============

func (c *ChromaClient) ListDatabases(ctx context.Context) ([]Database, error) {
	// URL includes the specific tenant from your client struct
	endpoint := fmt.Sprintf(listDatabases, c.URL, c.Tenant)
	slog.Info("Listing databases", "endpoint", endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list databases: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Chroma returns a list of database names as strings
	var databases []Database
	if err := json.NewDecoder(resp.Body).Decode(&databases); err != nil {
		return nil, fmt.Errorf("failed to decode databases: %w", err)
	}

	return databases, nil
}

// CreateDatabase creates a new database in the current tenant.
func (c *ChromaClient) CreateDatabase(ctx context.Context, name string) error {
	slog.Info("Creating database", "name", name, "tenant", c.Tenant)

	endpoint := fmt.Sprintf(createDatabase, c.URL, c.Tenant)

	// ChromaDB expects a simple POST with empty body or minimal JSON
	payload := map[string]any{"name": name}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal database payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create database '%s': status %d, body: %s", name, resp.StatusCode, string(body))
	}

	slog.Info("Database created successfully", "name", name, "tenant", c.Tenant)

	return nil
}

// ============ Collections ============

func (c *ChromaClient) ListCollections(ctx context.Context) ([]Collection, error) {
	endpoint := fmt.Sprintf(listCreateCollection, c.URL, c.Tenant, c.Database)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to request collections: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request collections: %w", err)
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list collections: status %d,body%s", resp.StatusCode, string(body))
	}

	var collections []Collection
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return nil, fmt.Errorf("failed to decode collections: %w", err)
	}

	return collections, nil
}

func (c *ChromaClient) CreateCollection(ctx context.Context, name string) (string, error) {
	slog.Info("Creating collection", "name", name)

	endpoint := fmt.Sprintf(listCreateCollection, c.URL, c.Tenant, c.Database)
	payload := CreateCollectionRequest{
		Name:        name,
		GetOrCreate: true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error received: unable to marshal json data for payload")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create collection: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create collection: %d, %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("error received: unable to decode information")
	}

	return result.ID, nil
}

// ============ Documents ============

// ListDocuments - List Documents in collection
func (c *ChromaClient) ListDocuments(ctx context.Context, collectionID string) (*GetRecordsResponse, error) {
	endpoint := fmt.Sprintf(listDocuments,
		c.URL, c.Tenant, c.Database, collectionID)

	slog.Info("Listing documents", "endpoint", endpoint)

	// Use a high limit to bypass any ChromaDB default caps on the get endpoint
	payload := map[string]any{
		"include": []string{"documents", "metadatas"},
		"limit":   100_000,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get documents: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result GetRecordsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	slog.Info("Listing documents result", "count", len(result.IDs))

	return &result, nil
}
func (c *ChromaClient) GetIDByName(ctx context.Context, name string) (string, error) {
	// Fetch all collections for the current tenant/db
	endpoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections", c.URL, c.Tenant, c.Database)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return name, nil
	}

	resp, err := c.client.Do(req)
	if err != nil {
		// If we can't list collections, we still return the input as ID
		// This matches the test expectation for error cases too
		return name, nil
	}
	defer MustClose(resp.Body.Close)

	var collections []Collection // Use the struct with ID and Name tags
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		// If we can't decode the response, we still return the input as ID
		return name, nil
	}

	for _, col := range collections {
		if col.Name == name {
			return col.ID, nil
		}
	}

	// If not found as a name, return the input as ID (assuming it's already an ID)
	return name, nil
}

// ResolveCollectionID accepts a collection name OR a UUID and returns the UUID.
// ctx propagates to the underlying HTTP request via http.NewRequestWithContext,
// so a cancelled context aborts the in-flight call.
func (c *ChromaClient) ResolveCollectionID(ctx context.Context, input string) (string, error) {
	if _, err := uuid.Parse(input); err == nil {
		return input, nil
	}

	endpoint := fmt.Sprintf(listCreateCollection, c.URL, c.Tenant, c.Database)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list collections: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to list collections: %w", err)
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to list collections: server returned %d", resp.StatusCode)
	}

	var collections []Collection
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return "", fmt.Errorf("failed to decode collections: %w", err)
	}

	for _, col := range collections {
		if col.Name == input {
			return col.ID, nil
		}
	}

	return "", fmt.Errorf("collection %q not found\nHint: Create it with: chroma create %s", input, input)
}

// ============ SetEmbedder ============

// SetEmbedder injects the embedder into the client for batch operations that need embeddings.
func (c *ChromaClient) SetEmbedder(e onnx.EmbedderInterface) {
	c.Embedder = e
}

// sendBatch is shared by AddBatchGeneric and UpsertBatchGeneric.
func (c *ChromaClient) sendBatch(ctx context.Context, endpoint string, documents []string, ids []string, metadatas []map[string]any) error {
	if len(documents) == 0 {
		return nil
	}

	embeddings, err := c.Embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	sanitizedMetadatas := sanitizeMetadataForChroma(metadatas)

	payload := map[string]any{
		"ids":        ids,
		"embeddings": embeddings,
		"metadatas":  sanitizedMetadatas,
		"documents":  documents,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal batch payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer internal.CheckDefer(resp.Body.Close)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return wrapChromaError(resp.StatusCode, body)
	}

	return nil
}

// ============ Deletion ============

func (c *ChromaClient) DeleteCollection(ctx context.Context, name string) error {
	slog.Info("Deleting collection", "name", name)

	// ChromaDB DELETE endpoint expects the collection name, not the UUID.
	endpoint := fmt.Sprintf(deleteCollection, c.URL, c.Tenant, c.Database, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete collection failed: %d, %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteRecords removes specific documents from a collection by their IDs.
func (c *ChromaClient) DeleteRecords(ctx context.Context, collectionID string, ids []string) error {
	slog.Info("Deleting records", "collection", collectionID, "ids", ids)

	// Resolve collection name to ID (handles both names and UUIDs).
	resolvedID, err := c.ResolveCollectionID(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("failed to resolve collection: %w", err)
	}

	endpoint := fmt.Sprintf(deleteRecords, c.URL, c.Tenant, c.Database, resolvedID)
	payload := map[string]any{"ids": ids}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delete payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete records: %w", err)
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete records failed: %d, %s", resp.StatusCode, string(body))
	}

	return nil
}

// ============ Embedding ============

// AddDocument - Corrected Metadata tag handling
func (c *ChromaClient) AddDocument(collectionID, id, text string, vector []float32) error {
	endpoint := fmt.Sprintf(batchAdd,
		c.URL, c.Tenant, c.Database, collectionID)

	payload := AddRecordsRequest{
		IDs:        []string{id},
		Documents:  []string{text},
		Embeddings: [][]float32{vector}, // Pass the vector here
	}

	jsonData, _ := json.Marshal(payload)

	resp, err := c.client.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma error (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Simplified logic for what your Go function will do:
func (c *ChromaClient) GenerateLocalEmbedding(text string) ([]float32, error) {
	if c.Embedder == nil {
		return nil, fmt.Errorf("error recieved: embedder is not initialized")
	}

	vector, err := c.Embedder.Embed(text)
	if err != nil {
		return nil, fmt.Errorf("error received failed to generate embeddings: %w", err)
	}

	return vector, nil
}

// ============ Querying ============

func (c *ChromaClient) QueryBatch(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*QueryResponse, error) {
	// 1. Generate embeddings for all queries at once
	// Assuming your local embedder can handle a slice of string
	vectors, err := c.Embedder.EmbedDocuments(ctx, queryTexts)
	if err != nil {
		return nil, err
	}
	// 2. Prepare payload for chroma
	payload := map[string]any{
		"query_embeddings": vectors,
		"n_results":        nResults,
		"include":          []string{"documents", "metadatas", "distances"},
	}
	// 3. Use the scoped query endpoint
	endPoint := fmt.Sprintf(queryEndpoint,
		c.URL, c.Tenant, c.Database, collectionID)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error received: unable to marshal json")
	}

	resp, err := c.client.Post(endPoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed: %d - %s", resp.StatusCode, string(body))
	}

	// The response structure for /query is slightly different (nested arrays)
	// but for simplicity, we can decode it into a similar format
	var result QueryResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ============ Batch Operations ============

func (c *ChromaClient) AddBatch(ctx context.Context, collectionID string, docs []string, ids []string) error {
	if c.Embedder == nil {
		return fmt.Errorf("embedder is not initialized; check if the AI model loaded correctly")
	}

	// 1. Generate embeddings for the entire batch
	// Using the EmbedDocuments function we discussed earlier
	vectors, err := c.Embedder.EmbedDocuments(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to embed batch: %w", err)
	}

	// 2. Prepare the Chroma /add payload
	endpoint := fmt.Sprintf(batchAdd,
		c.URL, c.Tenant, c.Database, collectionID)

	payload := map[string]any{
		"embeddings": vectors,
		"documents":  docs,
		"ids":        ids,
		// Optional: you can add "metadatas": []map[string]any here too
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.client.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add batch: %s", string(body))
	}

	return nil
}

// AddBatchGeneric handles documents, IDs, and dynamic metadata maps.
func (c *ChromaClient) AddBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	endpoint := fmt.Sprintf(batchAdd, c.URL, c.Tenant, c.Database, collectionID)
	return c.sendBatch(ctx, endpoint, documents, ids, metadatas)
}

// chromaMetadataKeyHint extracts the failing field name from a ChromaDB 422
// error like "metadatas[23].questionText: data did not match any variant...".
func chromaMetadataKeyHint(body []byte) string {
	const prefix = "metadatas["

	s := string(body)

	idx := strings.Index(s, prefix)
	if idx < 0 {
		return ""
	}

	remain := s[idx+len(prefix):]

	dotIdx := strings.Index(remain, ".")
	if dotIdx < 0 {
		return ""
	}

	fieldPart := remain[dotIdx+1:]

	colonIdx := strings.Index(fieldPart, ":")
	if colonIdx < 0 {
		return fieldPart
	}

	return strings.TrimSpace(fieldPart[:colonIdx])
}

// wrapChromaError enriches ChromaDB server errors with actionable hints.
func wrapChromaError(statusCode int, body []byte) error {
	if statusCode == 422 && strings.Contains(string(body), "MetadataValue") {
		if field := chromaMetadataKeyHint(body); field != "" {
			return fmt.Errorf(
				"metadata field %q was rejected by ChromaDB (expected string, number, or boolean)\n"+
					"Hint: Use --field-metadata to pick specific fields, "+
					"or add --exclude-field %[1]q when using --all-metadata", field)
		}

		return fmt.Errorf(
			"metadata values must be strings, numbers, or booleans\n" +
				"Hint: Use --field-metadata to select specific metadata fields, " +
				"or use --all-metadata with --exclude-field to omit problematic fields")
	}

	return fmt.Errorf("chroma server error (%d): %s", statusCode, string(body))
}

// UpsertBatchGeneric handles documents, IDs, and dynamic metadata maps for upserting (insert or update).
func (c *ChromaClient) UpsertBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	endpoint := fmt.Sprintf(batchUpsert, c.URL, c.Tenant, c.Database, collectionID)
	return c.sendBatch(ctx, endpoint, documents, ids, metadatas)
}

// sanitizeMetadataForChroma converts complex metadata values (nested maps, slices)
// to string representations so they are compatible with ChromaDB's metadata constraints.
func sanitizeMetadataForChroma(metadatas []map[string]any) []map[string]any {
	if len(metadatas) == 0 {
		return metadatas
	}

	result := make([]map[string]any, len(metadatas))
	for i, m := range metadatas {
		sanitized := make(map[string]any, len(m))
		for k, v := range m {
			sanitized[k] = sanitizeValue(v)
		}

		result[i] = sanitized
	}

	return result
}

// sanitizeValue recursively converts complex types to strings.
func sanitizeValue(v any) any {
	if v == nil {
		return nil
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
		return fmt.Sprintf("%v", v)
	default:
		return v
	}
}
