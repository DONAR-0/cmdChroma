package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/donar0/cmdChroma/internal"
	"github.com/donar0/cmdChroma/internal/onnx"
	"github.com/donar0/cmdChroma/internal/service"
	"github.com/urfave/cli/v3"
)

// Test Connection
func handleTestConnection(ctx context.Context, cmd *cli.Command) error {
	slog.Info("Starting connection test", "host", cmd.String("host"), "port", cmd.String("port"), "timeout", cmd.Int("timeout"))

	// Create Client with context from global flags
	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	// Create service (embedder not needed for test)
	svc := service.NewChromaService(chromaClient, nil)

	// Test the connection
	if err := svc.TestConnection(); err != nil {
		slog.Error("Connection test failed", "error", err)
		return fmt.Errorf("connection test failed: %w", err)
	}

	slog.Info("Connection test successful")
	fmt.Println("✅ Successfully connected to Chroma DB")
	return nil
}

// Current tenants
func handleCurrentTenants(ctx context.Context, cmd *cli.Command) error {
	slog.Info("Getting Current Tenant:", "Tenant:", cmd.String("tenant"))
	// Create Client
	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	// Call the method we added to your client package
	tenantExists, err := chromaClient.GetTenant()
	if err != nil {
		slog.Error("Failed to list tenants", "error", err)
		return fmt.Errorf("could not list tenants: %w", err)
	}

	slog.Info("✅ Retrieving if current tenant Exists: " + cmd.String("tenant"))
	slog.Info("Tenant Exists: " + fmt.Sprintf("%t", tenantExists))

	return nil
}

// List databases
func handleListDatabases(ctx context.Context, cmd *cli.Command) error {
	slog.Info("List All Databases:", "Tenant:", cmd.String("tenant"))

	// Create Client
	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	// Call the method we added to your client package
	dbs, err := chromaClient.ListDatabases()
	if err != nil {
		slog.Error("Failed to list Databases", "error", err)
		return fmt.Errorf("could not list Databases: %w", err)
	}
	slog.Info("✅ Successfully Retrieved Databases for tenant: " + cmd.String("tenant"))

	for _, db := range dbs {
		slog.Info("- ", "ID", db.Id, "Tenant", db.Tenant, "Name", db.Name)
	}

	return nil
}

// handleListCollection List Collection
func handleListCollection(_ context.Context, cmd *cli.Command) error {
	slog.Info("List All Collections:", "Tenant", cmd.String("tenant"), "Database", cmd.String("database"))

	// Create Client
	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}
	// Call the method we added to your client package
	collections, err := chromaClient.ListCollections()
	if err != nil {
		slog.Error("Failed to list Databases", "error", err)
		return fmt.Errorf("could not list Databases: %w", err)
	}
	slog.Info("✅ Successfully Retrieved Collections for database: " + cmd.String("database"))

	for _, collection := range collections {
		// slog.Info("- ", "ID", db.Id, "Tenant", db.Tenant, "Name", db.Name)
		slog.Info("-", "collection", collection.Name, "ID", collection.ID)
	}

	return nil
}

// handleCreateCollection Create Collection
func handleCreateCollection(_ context.Context, cmd *cli.Command) error {
	// Get Positional Argument
	collectionName := cmd.Args().Get(0)

	// Validate if name is provided
	if collectionName == "" {
		return fmt.Errorf("collectionName name is required as the first argument")
	}

	slog.Info("Creating Collection", "name", collectionName, "total_args", cmd.Args().Len())

	// Create Client
	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	id, err := chromaClient.CreateCollection(collectionName)
	if err != nil {
		return err
	}

	slog.Info("✅ Collection created", "Name ", collectionName, "Id ", id)

	return nil
}

// List Documents in collection
func handleListDocuments(_ context.Context, c *cli.Command) error {
	slog.Info("Inside handleListDocuments", "Tenant", c.String("tenant"), "Database", c.String("database"))
	input := c.Args().Get(0)
	if input == "" {
		return fmt.Errorf("argument empty Collection name not found, Please provide collection name")
	}
	client, _ := createChromaClient(c)

	// Step 1: Always try to resolve the name to an ID first
	targetID, err := client.GetIDByName(input)
	if err != nil {
		targetID = input
	}
	slog.Info("Resolved Collection Name to ID:", "Name", input, "ID", targetID)

	// Step 2: Now call the endpoint with a guaranteed UUID (hopefully)
	docs, err := client.ListDocuments(targetID)
	if err != nil {
		return fmt.Errorf("failed to list documents: %w", err)
	}

	fmt.Printf("\n--- Documents in %s ---\n", targetID)
	for i := 0; i < len(docs.IDs); i++ {
		fmt.Printf("ID:       %s\n", docs.IDs[i])

		// Check if Documents slice isn't empty
		if len(docs.Documents) > i {
			fmt.Printf("Content:  %s\n", docs.Documents[i])
		}

		// Check if Metadata exists for this record
		if len(docs.Metadatas) > i && docs.Metadatas[i] != nil {
			fmt.Printf("Metadata: %v\n", docs.Metadatas[i])
		}
		fmt.Println("-----------------------")
	}

	slog.Info(fmt.Sprintf("✅ Retrieved %d documents from %s\n", len(docs.IDs), input))
	return nil
}

func handleQueryBatchInCollection(_ context.Context, c *cli.Command) error {

	//1. Get multiple queries from the flag
	queries := c.StringSlice("query")
	if len(queries) == 0 {
		return fmt.Errorf("at least one --query flag is required")
	}
	collectionName := c.Args().Get(0)
	if collectionName == "" {
		return fmt.Errorf("collection name is required as the first argument")
	}

	// 2. Standard Client & AI Setup (Path resolution)
	client, err := createChromaClient(c)
	if err != nil {
		return err
	}

	slog.Info("Initializing AI Engine for search...")
	// 3. Load AI Engine (shared logic)
	embedder, err := initEmbedder(c)
	if err != nil {
		return err
	}
	defer embedder.Close()
	client.Embedder = embedder

	// 4. Resolve Collection ID
	targetID, err := client.GetIDByName(collectionName)
	if err != nil {
		return err
	}

	// 4. Execute Batch Query
	nResults := c.Int("n-results")
	if nResults == 0 {
		nResults = 3
	}

	slog.Info("Executing batch query...", "count", len(queries))
	response, err := client.QueryBatch(targetID, queries, nResults)
	if err != nil {
		return err
	}

	// 5. Display Results (Iterating through the nested response)
	//
	for i, originalQuery := range queries {
		fmt.Printf("\n🎯 Query: %s\n", originalQuery)
		fmt.Println(strings.Repeat("-", 40))

		// Check if we have results for this specific query
		for j := 0; j < len(response.IDs[i]); j++ {
			fmt.Printf("  [%d] (Dist: %.4f) ID: %s\n", j+1, response.Distances[i][j], response.IDs[i][j])
			fmt.Printf("      Content: %s\n\n", response.Documents[i][j])
		}
	}

	return nil
}

func handleBatchAddDocuments(_ context.Context, c *cli.Command) error {
	collectionName := c.Args().Get(0)
	docs := c.StringSlice("doc")

	if len(docs) == 0 {
		return fmt.Errorf("no documents provided. Use --doc multiple times")
	}

	// 1. Setup Client
	client, err := createChromaClient(c)
	if err != nil {
		return err
	}

	// 2. Load AI Engine (shared logic)
	embedder, err := initEmbedder(c)
	if err != nil {
		return err
	}
	defer embedder.Close()
	client.Embedder = embedder

	targetID, err := client.GetIDByName(collectionName)
	if err != nil {
		return err
	}

	// 2. Generate IDs (if not provided)
	// For datasets, it's best to generate unique IDs
	ids := make([]string, len(docs))
	for i := range docs {
		ids[i] = fmt.Sprintf("id-%d-%d", time.Now().UnixNano(), i)
	}

	// 3. Execute Batch Add
	slog.Info("Uploading batch to Chroma...", "size", len(docs))
	err = client.AddBatch(targetID, docs, ids)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Successfully added %d documents to %s\n", len(docs), collectionName)
	return nil
}

func handleImportJsonlFileInChromaDb(_ context.Context, c *cli.Command) error {
	collectionName := c.Args().Get(0)
	if collectionName == "" {
		return fmt.Errorf("collection name is not provided, required collection name")
	}

	fp := c.Args().Get(1)
	if fp == "" {
		return fmt.Errorf("filepath is not provided, Please provide jsonl file path")
	}

	// Flags
	contentKey := c.String("field-content")
	if contentKey == "" {
		contentKey = "text"
	}

	idKey := c.String("field-id")
	if idKey == "" {
		idKey = "id"
	}

	batchSize := int(c.Int("batch-size"))
	if batchSize <= 0 {
		batchSize = 100
	}

	allMetadata := c.Bool("all-metadata")
	metadataKeys := c.StringSlice("field-metadata")

	limit := c.Int("n-ingest")

	cleanPath := filepath.Clean(fp)
	// 1. Check if the path is absolute
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("security error: absolute paths are not allowed. " +
			"Please provide a path relative to the current directory")
	}

	// 2. Open the Root (Current Working Directory)
	cwd, _ := os.Getwd()
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return err
	}
	defer internal.CheckDefer(root.Close)

	// 3. Attempt to open the file
	file, err := root.Open(cleanPath)
	if err != nil {
		return fmt.Errorf("cannot open file '%s': %w", fp, err)
	}
	defer internal.CheckDefer(file.Close)

	// 3. Client & Embedder Initialization
	client, err := createChromaClient(c)
	if err != nil {
		return err
	}

	embedder, err := initEmbedder(c)
	if err != nil {
		return err
	}

	client.Embedder = embedder

	collectionID, err := client.GetIDByName(collectionName)
	if err != nil {
		return fmt.Errorf("collection '%s' not found: %w", collectionName, err)
	}

	// 4. Ingestion Loop
	scanner := bufio.NewScanner(file)
	const maxCapacity = 1 * 1024 * 1024
	scanner.Buffer(make([]byte, maxCapacity), maxCapacity)

	var (
		docs  []string
		ids   []string
		metas []map[string]any
		count = 0
	)

	slog.Info("Starting generic ingestion...", "file", fp, "content_field", contentKey)

	for scanner.Scan() {
		var rec map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			slog.Error("Failed to parse line", "error", err)
			continue
		}

		contentVal := getNestedValue(rec, contentKey)
		if contentVal == nil {
			continue
		}
		content := fmt.Sprintf("%v", contentVal)

		// Extract or Generate ID
		var currentID string
		idVal := getNestedValue(rec, idKey)
		if idVal != nil {
			currentID = fmt.Sprintf("%v", idVal)
		} else {
			// Deterministic Hash fallback: Prevents duplicates on re-runs
			hash := sha256.Sum256([]byte(content))
			currentID = hex.EncodeToString(hash[:12])
		}

		// Extract Metadata
		meta := make(map[string]any)
		if allMetadata {
			for k, v := range rec {
				if k != contentKey && k != idKey {
					meta[k] = v
				}
			}
		} else {
			for _, k := range metadataKeys {
				if v, exists := rec[k]; exists {
					meta[k] = v
				}
			}
		}

		// Accumulate
		docs = append(docs, content)
		ids = append(ids, currentID)
		metas = append(metas, meta)
		count++

		// 1. Check Limit IMMEDIATELY
		if limit > 0 && count >= limit {
			slog.Info("Limit reached", "count", count)
			// Flush what we have in the current batch before breaking
			if len(docs) > 0 {
				if err := client.AddBatchGeneric(collectionID, docs, ids, metas); err != nil {
					return err
				}
			}
			docs, ids, metas = nil, nil, nil
			break // Exit the loop
		}
		// Batch Flush
		if len(docs) >= batchSize {
			slog.Info("Sending batch to Chroma", "size", len(docs))
			if err := client.AddBatchGeneric(collectionID, docs, ids, metas); err != nil {
				return err
			}
			docs, ids, metas = nil, nil, nil
		}
	}

	// Final Flush
	if len(docs) > 0 {
		if err := client.AddBatchGeneric(collectionID, docs, ids, metas); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	slog.Info("Ingestion successful", "total", count)
	return nil
}

func getNestedValue(m map[string]any, path string) any {

	parts := strings.Split(path, ".")
	var current any = m
	for _, part := range parts {
		if next, ok := current.(map[string]any); ok {
			current = next[part]
		} else {
			return nil
		}
	}
	return current
}

func initEmbedder(c *cli.Command) (*onnx.Embedder, error) {
	modelPath, tokenizerPath, onnxLibPath, err := resolveAIPaths(c)
	if err != nil {
		return nil, err
	}

	slog.Info("Loading AI Embedding Engine...", "model", modelPath)
	embedder, err := onnx.NewEmbedder(modelPath, tokenizerPath, onnxLibPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI engine: %w", err)
	}
	return embedder, nil
}

func resolveAIPaths(c *cli.Command) (string, string, string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	binDir := filepath.Dir(ex)
	projectRoot := filepath.Join(binDir, "..")

	modelPath := c.String("model-path")
	if modelPath == "" {
		modelPath = filepath.Join(projectRoot, "models/all-MiniLM-L6-v2/model.onnx")
	}

	tokenizerPath := c.String("tokenizer-path")
	if tokenizerPath == "" {
		tokenizerPath = filepath.Join(projectRoot, "models/all-MiniLM-L6-v2/tokenizer.json")
	}

	onnxLibPath := c.String("onnx-lib")
	if onnxLibPath == "" {
		onnxLibPath = filepath.Join(projectRoot, "models/onnx_runtime/lib/libonnxruntime.so")
	}

	return modelPath, tokenizerPath, onnxLibPath, nil
}

type IngestRecord map[string]any
