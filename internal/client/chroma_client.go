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
	"net"
	"net/http"
	"time"

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
	countDocuments       = "%s/api/v2/tenants/%s/databases/%s/collections/%s/count"
	queryEndpoint        = "%s/api/v2/tenants/%s/databases/%s/collections/%s/query"
	batchAdd             = "%s/api/v2/tenants/%s/databases/%s/collections/%s/add"
	batchUpsert          = "%s/api/v2/tenants/%s/databases/%s/collections/%s/upsert"
	deleteCollection     = "%s/api/v2/tenants/%s/databases/%s/collections/%s"
	deleteRecords        = "%s/api/v2/tenants/%s/databases/%s/collections/%s/delete"
)

// ============ Core Types ============

var _ embedder = (*onnx.Embedder)(nil)

// embedder is the subset of *onnx.Embedder used by *ChromaClient.
type embedder interface {
	Embed(text string) ([]float32, error)
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	Close()
}

type ChromaClient struct {
	URL      string
	Tenant   string
	Database string
	client   *http.Client
	Embedder embedder
}

// ============ Data Transfer Types ============

type (
	CreateCollectionRequest struct {
		Name        string         `json:"name"`
		Metadata    map[string]any `json:"metadata"`
		GetOrCreate bool           `json:"get_or_create"`
	}

	Collection struct {
		ID        string         `json:"id"`
		Name      string         `json:"name"`
		Tenant    string         `json:"tenant"`
		Database  string         `json:"database"`
		Metadata  map[string]any `json:"metadata"`
		Dimension *int           `json:"dimension"`
		Config    map[string]any `json:"configuration_json"`
	}

	Database struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Tenant string `json:"tenant"`
	}

	GetRecordsRequest struct {
		Tenant   string   `json:"tenant"`
		Database string   `json:"database"`
		IDs      []string `json:"ids,omitempty"`
		Include  []string `json:"include"`
		Limit    *int     `json:"limit"`
		Offset   *int     `json:"offset"`
	}

	GetRecordsResponse struct {
		IDs       []string         `json:"ids"`
		Documents []string         `json:"documents"`
		Metadatas []map[string]any `json:"metadatas"`
	}

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
		client: &http.Client{
			Transport: &http.Transport{
				Proxy:       http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
			},
		},
	}
}

// ============ Connection & Health ============

func (c *ChromaClient) TestConnection(ctx context.Context) error {
	endpoint := fmt.Sprintf(testEndpoint, c.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to ChromaDB at %s: %w", c.URL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("deferred close failed", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed: %d, response: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ============ Tenant & Database ============

func (c *ChromaClient) GetTenant(ctx context.Context) (bool, error) {
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

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
}

func (c *ChromaClient) ListDatabases(ctx context.Context) ([]Database, error) {
	endpoint := fmt.Sprintf(listDatabases, c.URL, c.Tenant)

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

	var databases []Database
	if err := json.NewDecoder(resp.Body).Decode(&databases); err != nil {
		return nil, fmt.Errorf("failed to decode databases: %w", err)
	}

	return databases, nil
}

func (c *ChromaClient) CreateDatabase(ctx context.Context, name string) error {
	endpoint := fmt.Sprintf(createDatabase, c.URL, c.Tenant)
	payload := map[string]any{"name": name}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
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
		return nil, fmt.Errorf("failed to list collections: status %d, body: %s", resp.StatusCode, string(body))
	}

	var collections []Collection
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return nil, fmt.Errorf("failed to decode collections: %w", err)
	}

	return collections, nil
}

func (c *ChromaClient) CreateCollection(ctx context.Context, name string) (string, error) {
	endpoint := fmt.Sprintf(listCreateCollection, c.URL, c.Tenant, c.Database)
	payload := CreateCollectionRequest{Name: name, GetOrCreate: true}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
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
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, nil
}

// ============ Documents ============

func (c *ChromaClient) ListDocuments(ctx context.Context, collectionID string) (*GetRecordsResponse, error) {
	endpoint := fmt.Sprintf(listDocuments, c.URL, c.Tenant, c.Database, collectionID)
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

	return &result, nil
}

func (c *ChromaClient) CountDocuments(ctx context.Context, collectionID string) (int64, error) {
	endpoint := fmt.Sprintf(countDocuments, c.URL, c.Tenant, c.Database, collectionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("count request failed: %w", err)
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("count failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var count int64
	if err := json.NewDecoder(resp.Body).Decode(&count); err != nil {
		return 0, fmt.Errorf("failed to decode count response: %w", err)
	}

	return count, nil
}

func (c *ChromaClient) GetIDByName(ctx context.Context, name string) (string, error) {
	endpoint := fmt.Sprintf(listCreateCollection, c.URL, c.Tenant, c.Database)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return name, nil
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return name, nil
	}
	defer MustClose(resp.Body.Close)

	var collections []Collection
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return name, nil
	}

	for _, col := range collections {
		if col.Name == name {
			return col.ID, nil
		}
	}

	return name, nil
}

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

	return "", fmt.Errorf("collection %q not found", input)
}

func (c *ChromaClient) SetEmbedder(e embedder) {
	c.Embedder = e
}

func (c *ChromaClient) sendBatch(ctx context.Context, endpoint string, documents []string, ids []string, metadatas []map[string]any) error {
	if len(documents) == 0 {
		return nil
	}

	embeddings, err := c.Embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	payload := map[string]any{
		"ids":        ids,
		"embeddings": embeddings,
		"metadatas":  metadatas,
		"documents":  documents,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal batch payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chromadb error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ============ Deletion ============

func (c *ChromaClient) DeleteCollection(ctx context.Context, name string) error {
	collectionID, err := c.ResolveCollectionID(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to resolve collection ID: %w", err)
	}

	// Try deleting by UUID first
	endpoint := fmt.Sprintf(deleteCollection, c.URL, c.Tenant, c.Database, collectionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err == nil {
		resp, err := c.client.Do(req)
		if err == nil {
			defer MustClose(resp.Body.Close)

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
				return nil
			}

			if resp.StatusCode != http.StatusNotFound {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("chromadb API returned status %d for UUID endpoint %q: %s", resp.StatusCode, endpoint, string(body))
			}
		}
	}

	// Fallback: Try deleting by name if UUID returned 404 or request failed
	endpoint = fmt.Sprintf(deleteCollection, c.URL, c.Tenant, c.Database, name)

	req, err = http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err == nil {
		resp, err := c.client.Do(req)
		if err == nil {
			defer MustClose(resp.Body.Close)

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
				return nil
			}

			body, _ := io.ReadAll(resp.Body)

			return fmt.Errorf("chromadb API returned status %d for name endpoint %q: %s", resp.StatusCode, endpoint, string(body))
		}
	}

	return fmt.Errorf("failed to delete collection %q using both UUID and name", name)
}

func (c *ChromaClient) DeleteRecords(ctx context.Context, collectionID string, ids []string) error {
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

// ============ Querying ============

func (c *ChromaClient) QueryBatch(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*QueryResponse, error) {
	vectors, err := c.Embedder.EmbedDocuments(ctx, queryTexts)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"query_embeddings": vectors,
		"n_results":        nResults,
		"include":          []string{"documents", "metadatas", "distances"},
	}
	endpoint := fmt.Sprintf(queryEndpoint, c.URL, c.Tenant, c.Database, collectionID)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer MustClose(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed: %d, %s", resp.StatusCode, string(body))
	}

	var result QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ============ Batch Operations ============

func (c *ChromaClient) AddBatch(ctx context.Context, collectionID string, docs []string, ids []string) error {
	return c.AddBatchGeneric(ctx, collectionID, docs, ids, nil)
}

func (c *ChromaClient) AddBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	endpoint := fmt.Sprintf(batchAdd, c.URL, c.Tenant, c.Database, collectionID)
	return c.sendBatch(ctx, endpoint, documents, ids, metadatas)
}

func (c *ChromaClient) UpsertBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	endpoint := fmt.Sprintf(batchUpsert, c.URL, c.Tenant, c.Database, collectionID)
	return c.sendBatch(ctx, endpoint, documents, ids, metadatas)
}

// ============ Metadata Helpers ============

func sanitizeValue(val any) any {
	if val == nil {
		return nil
	}

	switch val.(type) {
	case string, int, int32, int64, float32, float64, bool:
		return val
	default:
		return fmt.Sprint(val)
	}
}

func sanitizeMetadataForChroma(metadatas []map[string]any) []map[string]any {
	if metadatas == nil {
		return nil
	}

	out := make([]map[string]any, len(metadatas))
	for i, m := range metadatas {
		sanitized := make(map[string]any, len(m))
		for k, v := range m {
			sanitized[k] = sanitizeValue(v)
		}

		out[i] = sanitized
	}

	return out
}

func chromaMetadataKeyHint(body []byte) string {
	// Parse "metadatas[N].<key>:" from ChromaDB validation errors
	s := string(body)
	prefix := "metadatas["

	idx := 0
	for idx < len(s) && s[idx] != '[' {
		idx++
	}

	if idx >= len(s) {
		return ""
	}
	// skip past "metadatas["
	rest := s[len(prefix):]
	// skip digits and "]"
	i := 0
	for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
		i++
	}

	if i >= len(rest) || rest[i] != ']' {
		return ""
	}

	i++ // skip ']'
	if i >= len(rest) || rest[i] != '.' {
		return ""
	}

	i++ // skip '.'

	start := i
	for i < len(rest) && rest[i] != ':' && rest[i] != ' ' {
		i++
	}

	if i == start {
		return ""
	}

	return rest[start:i]
}

func wrapChromaError(statusCode int, body []byte) error {
	if statusCode == 422 {
		if key := chromaMetadataKeyHint(body); key != "" {
			return fmt.Errorf("chroma 422: metadata field %q is invalid (exclude-field): %s", key, string(body))
		}

		return fmt.Errorf("chroma 422: metadata value must be strings, numbers, or booleans: %s", string(body))
	}

	return fmt.Errorf("chroma %d: %s", statusCode, string(body))
}
