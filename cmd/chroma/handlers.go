package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DONAR-0/cmdChroma/internal"
	client "github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/ingest"
	"github.com/DONAR-0/cmdChroma/internal/llm"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
	"github.com/DONAR-0/cmdChroma/internal/service"
	"github.com/urfave/cli/v3"
)

// ============ Input Validation Errors ============

var (
	// Validation errors with actionable hints
	errCollectionNameRequired = "collection name is required\n\nUsage: chroma create <collection_name>\n\nExample: chroma create my_docs"
	errInvalidNIMModel        = "invalid NIM model format: use nim://<model-id>\n\nExample: --llm-model nim://mistralai/mistral-7b-instruct-v0.3"
	errNIMAPIKeyMissing       = "NVIDIA_API_KEY environment variable is required for NIM\n\nSet it with: export NVIDIA_API_KEY=your-key"
)

// validateNIMModel validates and extracts model ID from nim:// prefix.
func validateNIMModel(model string) (modelID string, err error) {
	if model == "" {
		return "", fmt.Errorf("model is required")
	}

	if !strings.HasPrefix(model, "nim://") {
		return model, nil // Not NIM, return as-is
	}

	modelID = strings.TrimPrefix(model, "nim://")
	if modelID == "" {
		return "", errors.New(errInvalidNIMModel)
	}

	return modelID, nil
}

// validateCollectionName checks if collection name is valid.
func validateCollectionName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New(errCollectionNameRequired)
	}

	return nil
}

// ============ Command Handlers ============
// Handlers are organized by functional domain.

// ===== Connection & Tenant =====

// handleTestConnection tests connectivity to ChromaDB.
func handleTestConnection(ctx context.Context, cmd *cli.Command) error {
	start := time.Now()
	opName := "test_connection"

	host := cmd.String("host")
	port := cmd.String("port")

	slog.Info("operation_start", "op", opName, "host", host, "port", port)

	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		slog.Error("operation_failed", "op", opName, "error", err, "duration_ms", time.Since(start).Milliseconds())
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	svc := service.NewChromaService(chromaClient, nil)

	timeout := cmd.Int("timeout")

	rlCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	if err := svc.TestConnection(); err != nil {
		// Check context deadline
		if rlCtx.Err() == context.DeadlineExceeded {
			slog.Error("operation_timeout", "op", opName, "timeout_s", timeout, "duration_ms", time.Since(start).Milliseconds())
			return fmt.Errorf("connection timed out after %ds\n\nHints:\n  • Check if ChromaDB is running\n  • Verify host/port: --host %s --port %s", timeout, host, port)
		}

		slog.Error("operation_failed", "op", opName, "error", err, "duration_ms", time.Since(start).Milliseconds())

		return fmt.Errorf("connection failed: %w\n\nHints:\n  • Is ChromaDB running? Try: docker ps\n  • Check host/port: --host %s --port %s\n  • Verify network connectivity", err, host, port)
	}

	slog.Info("operation_complete", "op", opName, "duration_ms", time.Since(start).Milliseconds())
	fmt.Println("✅ Successfully connected to ChromaDB")
	fmt.Printf("   Server: %s:%s\n", host, port)
	fmt.Printf("   Tenant: %s\n", cmd.String("tenant"))
	fmt.Printf("   Database: %s\n", cmd.String("database"))

	return nil
}

// handleCurrentTenants shows tenant information.
func handleCurrentTenants(ctx context.Context, cmd *cli.Command) error {
	slog.Info("Checking tenant", "tenant", cmd.String("tenant"))

	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	tenantExists, err := chromaClient.GetTenant()
	if err != nil {
		slog.Error("Failed to check tenant", "error", err)
		return fmt.Errorf("tenant check failed: %w", err)
	}

	status := "✅ exists"
	if !tenantExists {
		status = "❌ not found"
	}

	fmt.Printf("Tenant: %s [%s]\n", cmd.String("tenant"), status)
	slog.Info("Tenant check complete", "exists", tenantExists)

	return nil
}

// handleListDatabases lists databases in the current tenant.
func handleListDatabases(ctx context.Context, cmd *cli.Command) error {
	slog.Info("Listing databases", "tenant", cmd.String("tenant"))

	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	dbs, err := chromaClient.ListDatabases()
	if err != nil {
		slog.Error("Failed to list databases", "error", err)
		return fmt.Errorf("failed to list databases: %w\n\nHint: Verify your tenant has databases: chroma databases --tenant default_tenant", err)
	}

	if len(dbs) == 0 {
		fmt.Println("No databases found in this tenant.")
		fmt.Println("Hint: Create a database first or check your tenant configuration.")

		return nil
	}

	fmt.Printf("Databases in tenant '%s':\n", cmd.String("tenant"))

	for _, db := range dbs {
		fmt.Printf("  • %s (ID: %s)\n", db.Name, db.Id)
	}

	slog.Info("Database listing complete", "count", len(dbs))

	return nil
}

// ===== Collections =====

// handleListCollection lists collections in the current database.
func handleListCollection(_ context.Context, cmd *cli.Command) error {
	slog.Info("Listing collections", "tenant", cmd.String("tenant"), "database", cmd.String("database"))

	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	collections, err := chromaClient.ListCollections()
	if err != nil {
		slog.Error("Failed to list collections", "error", err)
		return fmt.Errorf("failed to list collections: %w\n\nHint: Verify database exists: chroma databases", err)
	}

	if len(collections) == 0 {
		fmt.Println("No collections found in this database.")
		fmt.Println("Hint: Create a collection first: chroma create <name>")

		return nil
	}

	fmt.Printf("Collections in database '%s':\n", cmd.String("database"))

	for _, coll := range collections {
		fmt.Printf("  • %s (ID: %s)\n", coll.Name, coll.ID)
	}

	slog.Info("Collection listing complete", "count", len(collections))

	return nil
}

// handleCreateCollection creates a new collection.
func handleCreateCollection(_ context.Context, cmd *cli.Command) error {
	start := time.Now()
	opName := "create_collection"

	collectionName := cmd.Args().Get(0)
	if err := validateCollectionName(collectionName); err != nil {
		return err
	}

	slog.Info("operation_start", "op", opName, "name", collectionName)

	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	// Try to create the collection
	id, err := chromaClient.CreateCollection(collectionName)
	if err != nil {
		// Check if the error is due to missing database
		errMsg := err.Error()
		isDatabaseError := strings.Contains(errMsg, "does not exist") ||
			strings.Contains(errMsg, "Database") && strings.Contains(errMsg, "not found")

		if isDatabaseError && cmd.Bool("create-db") {
			// User requested automatic database creation
			dbName := cmd.String("database")
			slog.Info("Auto-creating database", "database", dbName)

			if createErr := chromaClient.CreateDatabase(dbName); createErr != nil {
				slog.Error("Failed to create database", "error", createErr)
				return fmt.Errorf("failed to create database '%s': %w", dbName, createErr)
			}

			fmt.Printf("✅ Database '%s' created\n", dbName)

			// Retry collection creation
			id, err = chromaClient.CreateCollection(collectionName)
			if err != nil {
				slog.Error("operation_failed", "op", opName, "name", collectionName, "error", err, "duration_ms", time.Since(start).Milliseconds())
				return fmt.Errorf("failed to create collection after database creation: %w", err)
			}
		} else if isDatabaseError {
			// Provide helpful error message with available databases
			dbs, listErr := chromaClient.ListDatabases()

			var hint string
			if listErr == nil && len(dbs) > 0 {
				dbList := make([]string, len(dbs))
				for i, db := range dbs {
					dbList[i] = db.Name
				}

				hint = fmt.Sprintf("\n\nAvailable databases in tenant '%s':\n  • %s\n\nUse --database <name> to specify an existing database, or --create-db to create it automatically.",
					cmd.String("tenant"), strings.Join(dbList, "\n  • "))
			} else {
				hint = "\n\nHint: Run 'chroma databases' to list available databases, or use --create-db to create the database automatically."
			}

			slog.Error("operation_failed", "op", opName, "name", collectionName, "error", err, "duration_ms", time.Since(start).Milliseconds())

			return fmt.Errorf("database '%s' does not exist in tenant '%s'%s", cmd.String("database"), cmd.String("tenant"), hint)
		} else {
			slog.Error("operation_failed", "op", opName, "name", collectionName, "error", err, "duration_ms", time.Since(start).Milliseconds())
			return fmt.Errorf("failed to create collection: %w\n\nHint: Check if collection already exists: chroma collections", err)
		}
	}

	slog.Info("operation_complete", "op", opName, "name", collectionName, "id", id, "duration_ms", time.Since(start).Milliseconds())
	fmt.Printf("✅ Collection '%s' created (ID: %s)\n", collectionName, id)

	return nil
}

// handleDeleteCollection deletes a collection.
func handleDeleteCollection(_ context.Context, cmd *cli.Command) error {
	collectionName := cmd.Args().Get(0)
	if collectionName == "" {
		return fmt.Errorf("collection name is required\n\nUsage: chroma delete <collection_name>\n\nExample: chroma delete my_collection")
	}

	slog.Info("Deleting collection", "name", collectionName)

	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	if err := chromaClient.DeleteCollection(collectionName); err != nil {
		slog.Error("Collection deletion failed", "error", err)
		// Check if it's a "not found" error (404)
		if strings.Contains(err.Error(), "404") && (strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found")) {
			return fmt.Errorf("collection '%s' not found", collectionName)
		}

		return fmt.Errorf("failed to delete collection: %w\n\nHint: Verify collection exists and you have delete permissions", err)
	}

	fmt.Printf("✅ Collection '%s' deleted\n", collectionName)
	slog.Info("Collection deleted", "name", collectionName)

	return nil
}

// ===== Documents =====

// handleListDocuments lists documents in a collection.
func handleListDocuments(_ context.Context, c *cli.Command) error {
	collectionName := c.Args().Get(0)
	if collectionName == "" {
		return fmt.Errorf("collection name is required\n\nUsage: chroma records <collection_name>\n\nExample: chroma records my_collection")
	}

	slog.Info("Listing documents", "collection", collectionName, "tenant", c.String("tenant"), "database", c.String("database"))

	client, err := createChromaClient(c)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	// Resolve collection name to ID
	targetID, err := client.GetIDByName(collectionName)
	if err != nil {
		targetID = collectionName
		slog.Info("Collection name not found, using as-is", "name", collectionName)
	}

	docs, err := client.ListDocuments(targetID)
	if err != nil {
		slog.Error("Failed to list documents", "error", err)
		return fmt.Errorf("failed to list documents: %w\n\nHint: Verify collection exists and you have read permissions", err)
	}

	if len(docs.IDs) == 0 {
		fmt.Printf("Collection '%s' is empty.\n", collectionName)
		return nil
	}

	fmt.Printf("\n📄 Documents in collection '%s' (%d total):\n\n", collectionName, len(docs.IDs))

	for i := 0; i < len(docs.IDs); i++ {
		fmt.Printf("ID:       %s\n", docs.IDs[i])

		if len(docs.Documents) > i {
			content := docs.Documents[i]
			if len(content) > 100 {
				content = content[:100] + "..."
			}

			fmt.Printf("Content:  %s\n", content)
		}

		if len(docs.Metadatas) > i && docs.Metadatas[i] != nil {
			fmt.Printf("Metadata: %v\n", docs.Metadatas[i])
		}

		fmt.Println(strings.Repeat("-", 40))
	}

	slog.Info("Document listing complete", "collection", collectionName, "count", len(docs.IDs))

	return nil
}

// handleBatchAddDocuments adds documents to a collection.
func handleBatchAddDocuments(_ context.Context, c *cli.Command) error {
	collectionName := c.Args().Get(0)
	docs := c.StringSlice("doc")

	if len(docs) == 0 {
		return fmt.Errorf("no documents provided\n\nUsage: chroma add <collection> --doc \"text\"\n\nExample: chroma add my_collection --doc \"Your document\"")
	}

	// Setup client
	client, err := createChromaClient(c)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	// Load AI Engine
	embedder, err := initEmbedder(c)
	if err != nil {
		return fmt.Errorf("failed to initialize embedding engine: %w\n\nHint: Verify model files exist", err)
	}

	svc := service.NewChromaService(client, embedder)
	defer svc.Close()

	// Handle IDs: use provided ones or generate auto IDs
	ids := c.StringSlice("id")
	if len(ids) > 0 {
		// User provided custom IDs
		if len(ids) != len(docs) {
			return fmt.Errorf("number of IDs (%d) must match number of documents (%d)", len(ids), len(docs))
		}

		slog.Info("Using custom document IDs", "ids", ids)
	} else {
		// Generate auto IDs
		ids = make([]string, len(docs))
		for i := range docs {
			ids[i] = fmt.Sprintf("id-%d-%d", time.Now().UnixNano(), i)
		}
	}

	// Check if upsert flag is set
	upsert := c.Bool("upsert")

	// Execute
	if upsert {
		slog.Info("Upserting batch to Chroma", "collection", collectionName, "count", len(docs))

		if err := svc.UpsertDocuments(collectionName, docs, ids); err != nil {
			return fmt.Errorf("failed to upsert documents: %w\n\nHint: Check collection exists and you have write permissions", err)
		}

		fmt.Printf("✅ Successfully upserted %d documents to '%s'\n", len(docs), collectionName)
		slog.Info("Documents upserted", "collection", collectionName, "count", len(docs))
	} else {
		slog.Info("Uploading batch to Chroma", "collection", collectionName, "count", len(docs))

		if err := svc.AddDocuments(collectionName, docs, ids); err != nil {
			return fmt.Errorf("failed to add documents: %w\n\nHint: Check collection exists and you have write permissions", err)
		}

		fmt.Printf("✅ Successfully added %d documents to '%s'\n", len(docs), collectionName)
		slog.Info("Documents added", "collection", collectionName, "count", len(docs))
	}

	return nil
}

// handleDeleteRecords deletes specific documents from a collection by IDs.
func handleDeleteRecords(_ context.Context, cmd *cli.Command) error {
	collectionName := cmd.Args().Get(0)
	ids := cmd.StringSlice("id")

	if collectionName == "" {
		return fmt.Errorf("collection name is required\n\nUsage: chroma delete-records <collection> --id <id>\n\nExample: chroma delete-records my_collection --id doc-1")
	}

	if len(ids) == 0 {
		return fmt.Errorf("at least one document ID is required\n\nUsage: chroma delete-records <collection> --id <id>")
	}

	slog.Info("Deleting records", "collection", collectionName, "ids", ids)

	chromaClient, err := createChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	if err := chromaClient.DeleteRecords(collectionName, ids); err != nil {
		slog.Error("Delete records failed", "error", err)
		return fmt.Errorf("failed to delete records: %w", err)
	}

	fmt.Printf("✅ Deleted %d document(s) from '%s'\n", len(ids), collectionName)
	slog.Info("Records deleted", "collection", collectionName, "ids", ids)

	return nil
}

// ===== Search & Query =====

// handleQueryBatchInCollection performs batch semantic search.
func handleQueryBatchInCollection(_ context.Context, c *cli.Command) error {
	start := time.Now()
	opName := "query"

	queries := c.StringSlice("query")
	if len(queries) == 0 {
		return fmt.Errorf("at least one query is required\n\nUsage: chroma query <collection> --query \"search terms\"\n\nExample: chroma query my_collection --query \"What is Go?\"")
	}

	collectionName := c.Args().Get(0)
	if err := validateCollectionName(collectionName); err != nil {
		return err
	}

	nResults := c.Int("n-results")
	if nResults <= 0 {
		nResults = 5
	}

	slog.Info("operation_start", "op", opName, "collection", collectionName, "query_count", len(queries), "n_results", nResults)

	// Setup client
	client, err := createChromaClient(c)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	embedder, err := initEmbedder(c)
	if err != nil {
		return fmt.Errorf("failed to initialize embedding engine: %w\n\nHint: Verify model files exist or use --model-path, --tokenizer-path, --onnx-lib flags", err)
	}

	svc := service.NewChromaService(client, embedder)
	defer svc.Close()

	response, err := svc.QueryDocuments(collectionName, queries, nResults)
	if err != nil {
		slog.Error("operation_failed", "op", opName, "collection", collectionName, "error", err, "duration_ms", time.Since(start).Milliseconds())
		return fmt.Errorf("query failed: %w\n\nHint: Check collection exists and contains documents", err)
	}

	// Display results
	fmt.Printf("\n🔍 Search results for collection '%s':\n\n", collectionName)

	for i, originalQuery := range queries {
		fmt.Printf("Query %d: %s\n", i+1, originalQuery)
		fmt.Println(strings.Repeat("-", 60))

		for j := 0; j < len(response.IDs[i]); j++ {
			fmt.Printf("  [%d] Distance: %.4f\n", j+1, response.Distances[i][j])
			fmt.Printf("      ID: %s\n", response.IDs[i][j])

			if len(response.Documents[i]) > j {
				content := response.Documents[i][j]
				if len(content) > 150 {
					content = content[:150] + "..."
				}

				fmt.Printf("      Content: %s\n\n", content)
			}
		}

		if i < len(queries)-1 {
			fmt.Println(strings.Repeat("=", 60) + "\n")
		}
	}

	slog.Info("operation_complete", "op", opName, "collection", collectionName, "result_count", len(response.IDs[0]), "duration_ms", time.Since(start).Milliseconds())

	return nil
}

// ===== Import =====

// handleImportFileInChromaDb imports a file (JSONL or Parquet) with progress tracking.
func handleImportFileInChromaDb(_ context.Context, c *cli.Command) error {
	collectionName := c.Args().Get(0)
	if collectionName == "" {
		return fmt.Errorf("collection name is required\n\nUsage: chroma import <collection> <file.jsonl|file.parquet>\n\nExample: chroma import my_collection data.jsonl")
	}

	fp := c.Args().Get(1)
	if fp == "" {
		return fmt.Errorf("file path is required\n\nUsage: chroma import <collection> <file.jsonl|file.parquet>")
	}

	// Validate path
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	safePath, err := internal.SafeJoin(cwd, fp)
	if err != nil {
		return fmt.Errorf("invalid file path: %w\n\nHint: Use relative paths within the current directory", err)
	}

	// Build ingest config
	cfg := &ingest.Config{
		BatchSize:      int(c.Int("batch-size")),
		ContentField:   c.String("field-content"),
		IDField:        c.String("field-id"),
		MetadataFields: c.StringSlice("field-metadata"),
		AllMetadata:    c.Bool("all-metadata"),
		Limit:          c.Int("n-ingest"),
	}
	// Apply defaults
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}

	if cfg.ContentField == "" {
		cfg.ContentField = "text"
	}

	if cfg.IDField == "" {
		cfg.IDField = "id"
	}

	// Setup service
	client, err := createChromaClient(c)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	embedder, err := initEmbedder(c)
	if err != nil {
		return fmt.Errorf("failed to initialize embedding engine: %w", err)
	}

	svc := service.NewChromaService(client, embedder)
	defer svc.Close()

	// Resolve collection ID
	collectionID, err := client.ResolveCollectionID(collectionName)
	if err != nil {
		return fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	// Determine file type by extension
	ext := strings.ToLower(filepath.Ext(safePath))

	processor := ingest.NewProcessor(cfg)

	var (
		recordsChan <-chan *ingest.Record
		errChan     <-chan error
	)

	switch ext {
	case ".jsonl":
		recordsChan, errChan = processor.ProcessJSONL(safePath)
	case ".parquet":
		recordsChan, errChan = processor.ProcessParquet(safePath)
	default:
		return fmt.Errorf("unsupported file format: %s (supported: .jsonl, .parquet)", ext)
	}

	// Start import with progress
	fmt.Printf("📥 Importing from '%s' to collection '%s'\n", filepath.Base(safePath), collectionName)
	fmt.Printf("   Batch size: %d\n", cfg.BatchSize)

	if cfg.Limit > 0 {
		fmt.Printf("   Limit: %d documents\n", cfg.Limit)
	}

	fmt.Println()

	startTime := time.Now()

	slog.Info("Starting import", "file", safePath, "collection", collectionName, "format", ext)

	// Batch accumulation with progress tracking
	var (
		docs          []string
		ids           []string
		metas         []map[string]any
		batchIdx      int
		totalUploaded int
		progressN     = 10 // log progress every N documents processed
		nextProgress  = progressN
	)

	for record := range recordsChan {
		docs = append(docs, record.Content)
		ids = append(ids, record.ID)
		metas = append(metas, record.Metadata)
		batchIdx++

		// Current total processed (including current batch)
		currentTotal := totalUploaded + batchIdx

		// Progress update every N documents
		if currentTotal >= nextProgress && batchIdx < cfg.BatchSize {
			slog.Info("Progress", "total_processed", currentTotal, "batch_accumulated", batchIdx)

			nextProgress += progressN
		}

		if batchIdx >= cfg.BatchSize {
			if err := client.AddBatchGeneric(collectionID, docs, ids, metas); err != nil {
				return fmt.Errorf("batch upload failed at document %d: %w", totalUploaded, err)
			}

			totalUploaded += len(docs)
			slog.Info("Batch uploaded", "batch_size", len(docs), "total_uploaded", totalUploaded)
			docs, ids, metas = nil, nil, nil
			batchIdx = 0
			nextProgress = totalUploaded + progressN // set next milestone
		}
	}

	// Final batch
	if len(docs) > 0 {
		if err := client.AddBatchGeneric(collectionID, docs, ids, metas); err != nil {
			return fmt.Errorf("final batch upload failed at document %d: %w", totalUploaded, err)
		}

		totalUploaded += len(docs)
		slog.Info("Final batch uploaded", "batch_size", len(docs), "total_uploaded", totalUploaded)
	}

	// Check for errors from processor
	if err, ok := <-errChan; ok && err != nil {
		return fmt.Errorf("ingestion error: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n✅ Import completed in %s\n", elapsed.Round(time.Second))
	slog.Info("Import successful", "elapsed", elapsed, "total_documents", totalUploaded)

	return nil
}

// ===== RAG Chat ==========

// handleChat performs RAG-based question answering.
func handleChat(ctx context.Context, c *cli.Command) error {
	start := time.Now()
	opName := "chat"

	collectionName := c.Args().Get(0)
	if err := validateCollectionName(collectionName); err != nil {
		return err
	}

	question := c.Args().Get(1)
	if strings.TrimSpace(question) == "" {
		return fmt.Errorf("question is required\n\nUsage: chroma chat <collection> \"your question\"")
	}

	// Get model and validate
	model := c.String("llm-model")
	if model == "" {
		model = "qwen:0.5b"
	}

	// Validate model format
	if _, err := validateNIMModel(model); err != nil {
		return err
	}

	// Get n-results with validation
	nResults := c.Int("n-results")
	if nResults <= 0 {
		nResults = 3
	}

	distanceThreshold := c.Float64("distance-threshold")
	if distanceThreshold < 0 {
		return fmt.Errorf("distance-threshold must be >= 0")
	}

	slog.Info("operation_start",
		"op", opName,
		"collection", collectionName,
		"model", model,
		"n_results", nResults,
		"distance_threshold", distanceThreshold)

	// Setup client and embedder
	client, err := createChromaClient(c)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	embedder, err := initEmbedder(c)
	if err != nil {
		return fmt.Errorf("failed to initialize embedding engine: %w", err)
	}

	svc := service.NewChromaService(client, embedder)
	defer svc.Close()

	fmt.Printf("\n🤖 Querying collection '%s' with: %s\n\n", collectionName, question)

	// Query for context
	resp, err := svc.QueryDocuments(collectionName, []string{question}, nResults)
	if err != nil {
		slog.Error("operation_failed", "op", opName, "stage", "query", "error", err, "duration_ms", time.Since(start).Milliseconds())
		return fmt.Errorf("failed to retrieve context: %w\n\nHint: Make sure collection exists and has documents", err)
	}

	// Build context from response
	contextStr, hasRelevant, bestDistance := buildContextFromResponse(resp, distanceThreshold)

	if !hasRelevant {
		fmt.Println("⚠️  No relevant documents found. The LLM will be instructed to say so.")
	} else {
		fmt.Println("💭 Generating answer...")
	}

	// Build prompt based on whether we have context
	finalPrompt := buildRAGPrompt(question, contextStr, hasRelevant)

	// Get LLM provider
	provider, err := getLLMProvider(c, model)
	if err != nil {
		slog.Error("operation_failed", "op", opName, "stage", "provider_init", "error", err, "duration_ms", time.Since(start).Milliseconds())
		return err
	}

	// Generate response
	if err := provider.Generate(ctx, finalPrompt, model); err != nil {
		slog.Error("operation_failed", "op", opName, "stage", "generate", "error", err, "duration_ms", time.Since(start).Milliseconds())

		if strings.HasPrefix(model, "nim://") {
			return fmt.Errorf("NIM generation failed: %w\n\nHints:\n  • Ensure NVIDIA_API_KEY is set\n  • Verify model name (e.g., nim://mistralai/mistral-7b-instruct)\n  • Check your NIM endpoint URL", err)
		}

		return fmt.Errorf("LLM generation failed: %w\n\nHints:\n  • Is Ollama running? Start it with: ollama serve\n  • Pull the model: ollama pull %s\n  • Check: http://localhost:11434", err, model)
	}

	slog.Info("operation_complete",
		"op", opName,
		"has_context", hasRelevant,
		"best_distance", bestDistance,
		"duration_ms", time.Since(start).Milliseconds())

	return nil
}

// ============ RAG Helpers ============

// getLLMProvider creates the appropriate LLM provider based on model prefix.
func getLLMProvider(c *cli.Command, model string) (llm.ProviderInterface, error) {
	if strings.HasPrefix(model, "nim://") {
		nimURL := c.String("nim-url")

		provider, err := llm.NewNIMProvider(nimURL, "")
		if err != nil {
			// Provide helpful error message for missing API key
			if strings.Contains(err.Error(), "API key is required") {
				return nil, errors.New(errNIMAPIKeyMissing)
			}

			return nil, err
		}

		fmt.Println("\n🤖 Using NVIDIA NIM for generation")

		return provider, nil
	}

	// Default: Ollama provider
	provider := llm.NewProvider("")

	fmt.Println("\n🤖 Using Ollama for generation")

	return provider, nil
}

// buildContextFromResponse extracts relevant context from query response.
func buildContextFromResponse(resp *client.QueryResponse, distanceThreshold float64) (context string, hasRelevant bool, bestDistance float64) {
	var sb strings.Builder

	if len(resp.Documents) == 0 || len(resp.Documents[0]) == 0 {
		return "", false, 0
	}

	// Check distance threshold if set
	if distanceThreshold > 0 {
		bestDistance = float64(resp.Distances[0][0])
		if bestDistance > distanceThreshold {
			slog.Info("results_excluded_by_threshold",
				"best_distance", bestDistance,
				"threshold", distanceThreshold)

			return "", false, bestDistance
		}
	}

	// All results pass threshold (or no threshold set)
	if len(resp.Distances) > 0 && len(resp.Distances[0]) > 0 {
		bestDistance = float64(resp.Distances[0][0])
	}

	fmt.Printf("📚 Found %d relevant documents:\n\n", len(resp.Documents[0]))

	for i, doc := range resp.Documents[0] {
		fmt.Printf("  [%d] Distance: %.4f\n", i+1, resp.Distances[0][i])
		fmt.Printf("      Content: %s\n\n", doc)
		sb.WriteString(fmt.Sprintf("[Context %d]: %s\n", i+1, doc))
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()

	return sb.String(), true, bestDistance
}

// buildRAGPrompt creates the prompt based on available context.
func buildRAGPrompt(question, context string, hasContext bool) string {
	if !hasContext || context == "" {
		return fmt.Sprintf(`The user asked: "%s"

No relevant documents were found in the knowledge base.

Instructions: Respond with a clear statement that the answer cannot be found in the provided context. Do not guess or provide information from general knowledge. Simply indicate that the query does not match any available context.`,
			question)
	}

	return fmt.Sprintf(`Use the following context to answer the question.
If the context doesn't contain relevant information, say so.

Context:
%s

Question: %s

Provide a clear, concise answer:`,
		context, question)
}

// ============ Private Helpers ============

// initEmbedder initializes the ONNX embedder with path resolution.
func initEmbedder(c *cli.Command) (*onnx.Embedder, error) {
	modelPath, tokenizerPath, onnxLibPath, err := resolveAIPaths(c)
	if err != nil {
		return nil, err
	}

	slog.Info("Loading AI embedding engine", "model", modelPath)

	embedder, err := onnx.NewEmbedder(modelPath, tokenizerPath, onnxLibPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedder: %w", err)
	}

	return embedder, nil
}

// resolveAIPaths determines the paths for model files with fallbacks.
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

	// Validate files exist if using default paths
	if modelPath != "" {
		if _, err := os.Stat(modelPath); err != nil {
			return "", "", "", fmt.Errorf("model file not found: %s\n\nHint: Use --model-path to specify the correct location or run the setup script", modelPath)
		}
	}

	return modelPath, tokenizerPath, onnxLibPath, nil
}
