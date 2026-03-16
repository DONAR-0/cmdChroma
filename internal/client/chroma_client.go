package cClient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/donar0/cmdChroma/internal"
	"github.com/donar0/cmdChroma/internal/onnx"
	"github.com/google/uuid"
)

type ChromaClient struct {
	URL, Tenant, Database string
	client                *http.Client
	Embedder              *onnx.Embedder
}

func NewChromaDBClient(url, tenant, database string) *ChromaClient {
	slog.Info("Initiating ChromaClient Client", "URL:", url, "Tenant:", tenant, "Database:", database)
	return &ChromaClient{
		URL:      url,
		Tenant:   tenant,
		Database: database,
		client:   &http.Client{},
	}
}

func (c *ChromaClient) TestConnection() error {

	endpoint := fmt.Sprintf("%s/api/v2/heartbeat", c.URL)
	slog.Info("Calling Endpoint", "endpoint", endpoint)
	resp, err := c.client.Get(endpoint)

	if err != nil {
		return fmt.Errorf("failed to connect to ChromaDB at %s: %w", c.URL, err)
	}
	defer cd(resp.Body.Close)

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d, response: %s", resp.StatusCode, string(body))
	}

	slog.Info(fmt.Sprintf("ChromaDB connection successful: %s\n", string(body)))
	return nil
}

func (c *ChromaClient) GetTenant() (bool, error) {
	// Correct endpoint for checking a specific tenant
	endpoint := fmt.Sprintf("%s/api/v2/tenants/%s", c.URL, c.Tenant)

	resp, err := c.client.Get(endpoint)
	if err != nil {
		return false, err
	}
	defer cd(resp.Body.Close)

	// 200 means exists, 404 means it doesn't
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
}

func (c *ChromaClient) ListDatabases() ([]Database, error) {
	// URL includes the specific tenant from your client struct
	endpoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases", c.URL, c.Tenant)
	slog.Info("List Databases from URL :" + endpoint)
	resp, err := c.client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer cd(resp.Body.Close)

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

func (c *ChromaClient) ListCollections() ([]Collection, error) {
	// endpoint
	endpoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections", c.URL, c.Tenant, c.Database)

	resp, err := c.client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to request collections: %w", err)
	}
	defer cd(resp.Body.Close)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list collections: status %d,body%s", resp.StatusCode, string(body))
	}

	// Decode the response into a slice of string (names)
	var collections []Collection
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return nil, fmt.Errorf("failed to decode collections: %w", err)
	}
	return collections, nil
}

func (c *ChromaClient) CreateCollection(name string) (string, error) {
	slog.Info("- Creating Collection with name:", "Name", name)
	//endpoint
	endpoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections", c.URL, c.Tenant, c.Database)
	payload := CreateCollectionRequest{
		Name:        name,
		GetOrCreate: true,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error received: unable to marshal json data for payload")
	}
	resp, err := c.client.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer cd(resp.Body.Close)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create collection: %d, %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
		// Name string `json:"name"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("error received: unable to decode information")
	}
	return result.ID, nil
}

// ListDocuments - List Documents in collection
func (c *ChromaClient) ListDocuments(collectionID string) (*GetRecordsResponse, error) {
	// CHANGE: Use the fully scoped path, just like your AddDocument function
	endpoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s/get",
		c.URL, c.Tenant, c.Database, collectionID)

	slog.Info("Listing Documents", "endpoint", endpoint)

	// When using the scoped URL above, some Chroma versions expect a simpler body
	payload := map[string]any{
		"include": []string{"documents", "metadatas"},
		// "ids": nil, // Try omitting this first to get all
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.client.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer cd(resp.Body.Close)

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

func (c *ChromaClient) ResolveCollectionID(input string) (string, error) {
	// If it's already a UUID, return it
	if _, err := uuid.Parse(input); err == nil {
		return input, nil
	}

	// Otherwise, find the ID by Name
	collections, err := c.ListCollections()
	if err != nil {
		return "", err
	}
	for _, col := range collections {
		if col.Name == input {
			return col.ID, nil
		}
	}
	return "", fmt.Errorf("collection '%s' not found", input)
}

func (c *ChromaClient) GetIDByName(name string) (string, error) {
	// Fetch all collections for the current tenant/db
	endpoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections", c.URL, c.Tenant, c.Database)
	resp, err := c.client.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer cd(resp.Body.Close)

	var collections []Collection // Use the struct with ID and Name tags
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return "", err
	}

	for _, col := range collections {
		if col.Name == name {
			return col.ID, nil
		}
	}
	return "", fmt.Errorf("collection '%s' not found", name)
}

// AddDocument - Corrected Metadata tag handling
func (c *ChromaClient) AddDocument(collectionID, id, text string, vector []float32) error {
	endpoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s/add",
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
	defer cd(resp.Body.Close)

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

// func (c *ChromaClient) Query(collectioID, queryText string, nResults int) (*QueryResponse, error) {
// 	//1. Generate the embedding for the search term
// 	vector, err := c.GenerateLocalEmbedding(queryText)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to embed query: %w", err)
// 	}
// 	//2. Use the scoped query endpoint
// 	endPoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s/query",
// 		c.URL, c.Tenant, c.Database, collectioID)
//
// 	payload := map[string]any{
// 		"query_embeddings": [][]float32{vector},
// 		"n_results":        nResults,
// 		"include":          []string{"documents", "metadatas", "distances"},
// 	}
//
// 	jsonData, err := json.Marshal(payload)
// 	if err != nil {
// 		return nil, fmt.Errorf("error received: unable to marshal json")
// 	}
//
// 	resp, err := c.client.Post(endPoint, "application/json", bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer cd(resp.Body.Close)
//
// 	if resp.StatusCode != http.StatusOK {
// 		body, _ := io.ReadAll(resp.Body)
// 		return nil, fmt.Errorf("query failed: %d - %s", resp.StatusCode, string(body))
// 	}
//
// 	// The response structure for /query is slightly different (nested arrays)
// 	// but for simplicity, we can decode it into a similar format
// 	var result QueryResponse
//
// 	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
// 		return nil, err
// 	}
//
// 	return &result, nil
// }

func (c *ChromaClient) QueryBatch(collectionId string, queryTexts []string, nResults int) (*QueryResponse, error) {
	//1. Generate embeddings for all queries at once
	// Assuming your local embedder can handle a slice of string
	vectors, err := c.Embedder.EmbedDocuments(context.Background(), queryTexts)
	if err != nil {
		return nil, err
	}
	//2. Prepare payload for chroma
	payload := map[string]any{
		"query_embeddings": vectors,
		"n_results":        nResults,
		"include":          []string{"documents", "metadatas", "distances"},
	}
	//3. Use the scoped query endpoint
	endPoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s/query",
		c.URL, c.Tenant, c.Database, collectionId)

	jsonData, err := json.Marshal(payload)

	if err != nil {
		return nil, fmt.Errorf("error received: unable to marshal json")
	}

	resp, err := c.client.Post(endPoint, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		return nil, err
	}

	defer cd(resp.Body.Close)

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

func (c *ChromaClient) AddBatch(collectionID string, docs []string, ids []string) error {
	// 1. Generate embeddings for the entire batch
	// Using the EmbedDocuments function we discussed earlier
	vectors, err := c.Embedder.EmbedDocuments(context.Background(), docs)
	if err != nil {
		return fmt.Errorf("failed to embed batch: %w", err)
	}

	// 2. Prepare the Chroma /add payload
	endpoint := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s/add",
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
	defer cd(resp.Body.Close)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add batch: %s", string(body))
	}

	return nil
}

// Json Parser struct
type (
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

	Database struct {
		Id     string `json:"id"`
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
		Embeddings [][]float32      `json:"embeddings"` // Change: No longer omitempty
		Metadatas  []map[string]any `json:"metadatas,omitempty"`
	}

	QueryResponse struct {
		IDs       [][]string         `json:"ids"`
		Documents [][]string         `json:"documents"`
		Metadatas [][]map[string]any `json:"metadatas"`
		Distances [][]float32        `json:"distances"`
	}
)

var (
	cd = internal.CheckDefer
)
