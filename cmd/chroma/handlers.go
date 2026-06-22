package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/DONAR-0/cmdChroma/cmd/chroma/factory"
	"github.com/DONAR-0/cmdChroma/cmd/chroma/output"
	"github.com/DONAR-0/cmdChroma/internal"
	client "github.com/DONAR-0/cmdChroma/internal/client"
	config "github.com/DONAR-0/cmdChroma/internal/config"
	"github.com/DONAR-0/cmdChroma/internal/ingest"
	"github.com/DONAR-0/cmdChroma/internal/service"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// ============ Input Validation Errors ============

var (
	// Validation errors with actionable hints
	errCollectionNameRequired = "collection name is required\n\nUsage: chroma create <collection_name>\n\nExample: chroma create my_docs"
	errInvalidNIMModel        = "invalid NIM model format: use nim://<model-id>\n\nExample: --llm-model nim://mistralai/mistral-7b-instruct-v0.3"
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

	slog.Info("operation_start", "op", "test_connection", "host", host, "port", port)

	f := factory.NewServiceFactory()

	chromaClient, err := f.CreateChromaClient(cmd)
	if err != nil {
		slog.Error("operation_failed", "op", opName, "error", err, "duration_ms", time.Since(start).Milliseconds())
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	svc := service.NewChromaService(chromaClient, nil)

	timeout := cmd.Int("timeout")

	rlCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	if err := svc.TestConnection(rlCtx); err != nil {
		// Check context deadline
		if rlCtx.Err() == context.DeadlineExceeded {
			slog.Error("operation_timeout", "op", opName, "timeout_s", timeout, "duration_ms", time.Since(start).Milliseconds())
			printer.Error("Connection timed out after %d seconds", timeout)
			printer.Info("Check if ChromaDB is running")
			printer.Info("Verify host/port: --host %s --port %s", host, port)

			return fmt.Errorf("connection timed out after %ds", timeout)
		}

		slog.Error("operation_failed", "op", opName, "error", err, "duration_ms", time.Since(start).Milliseconds())
		printer.Error("Connection failed: %v", err)
		printer.Info("Check if ChromaDB is running with: docker ps")
		printer.Info("Verify host/port: --host %s --port %s", host, port)

		return err
	}

	slog.Info("operation_complete", "op", opName, "duration_ms", time.Since(start).Milliseconds())
	printer.Success("Successfully connected to ChromaDB")
	printer.Printf("   Server: %s:%s\n", host, port)
	printer.Printf("   Tenant: %s\n", cmd.String("tenant"))
	printer.Printf("   Database: %s\n", cmd.String("database"))

	return nil
}

// handleCurrentTenants shows tenant information.
func handleCurrentTenants(ctx context.Context, cmd *cli.Command) error {
	slog.Info("Checking tenant", "tenant", cmd.String("tenant"))

	f := factory.NewServiceFactory()

	chromaClient, err := f.CreateChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	tenantExists, err := chromaClient.GetTenant(ctx)
	if err != nil {
		slog.Error("Failed to check tenant", "error", err)
		printer.Error("Tenant check failed: %v", err)

		return err
	}

	if tenantExists {
		printer.Success("Tenant: %s [exists]", cmd.String("tenant"))
	} else {
		printer.Warn("Tenant: %s [not found]", cmd.String("tenant"))
	}

	slog.Info("Tenant check complete", "exists", tenantExists)

	return nil
}

// handleListDatabases lists databases in the current tenant.
func handleListDatabases(ctx context.Context, cmd *cli.Command) error {
	slog.Info("Listing databases", "tenant", cmd.String("tenant"))

	f := factory.NewServiceFactory()

	chromaClient, err := f.CreateChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	dbs, err := chromaClient.ListDatabases(ctx)
	if err != nil {
		slog.Error("Failed to list databases", "error", err)
		printer.Error("Failed to list databases: %v", err)
		printer.Info("Verify your tenant has databases")

		return err
	}

	if len(dbs) == 0 {
		printer.Info("No databases found in this tenant.")
		printer.Info("Create a database first or check your tenant configuration.")

		return nil
	}

	printer.Printf("Databases in tenant '%s':\n", cmd.String("tenant"))
	printer.PrintTable([]string{"Name", "ID"}, toStringRows(dbs))

	slog.Info("Database listing complete", "count", len(dbs))

	return nil
}

// ===== Collections =====

// handleListCollection lists collections in the current database.
func handleListCollection(ctx context.Context, cmd *cli.Command) error {
	slog.Info("Listing collections", "tenant", cmd.String("tenant"), "database", cmd.String("database"))

	f := factory.NewServiceFactory()

	chromaClient, err := f.CreateChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	collections, err := chromaClient.ListCollections(ctx)
	if err != nil {
		slog.Error("Failed to list collections", "error", err)
		return fmt.Errorf("failed to list collections: %w\n\nHint: Verify database exists: chroma databases", err)
	}

	if len(collections) == 0 {
		printer.Info("No collections found in this database.")
		printer.Info("Hint: Create a collection first: chroma create <name>")

		return nil
	}

	printer.Info("Collections in database '%s':", cmd.String("database"))

	for _, coll := range collections {
		printer.Printf("  • %s (ID: %s)\n", coll.Name, coll.ID)
	}

	slog.Info("Collection listing complete", "count", len(collections))

	return nil
}

// handleCreateCollection creates a new collection.
func handleCreateCollection(ctx context.Context, cmd *cli.Command) error {
	start := time.Now()
	opName := "create_collection"

	collectionName := cmd.Args().Get(0)
	if err := validateCollectionName(collectionName); err != nil {
		return err
	}

	slog.Info("operation_start", "op", opName, "name", collectionName)

	f := factory.NewServiceFactory()

	chromaClient, err := f.CreateChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	// Try to create the collection
	id, err := chromaClient.CreateCollection(ctx, collectionName)
	if err != nil {
		// Check if the error is due to missing database
		errMsg := err.Error()
		isDatabaseError := strings.Contains(errMsg, "does not exist") ||
			strings.Contains(errMsg, "Database") && strings.Contains(errMsg, "not found")

		if isDatabaseError && cmd.Bool("create-db") {
			// User requested automatic database creation
			dbName := cmd.String("database")
			slog.Info("Auto-creating database", "database", dbName)

			if createErr := chromaClient.CreateDatabase(ctx, dbName); createErr != nil {
				slog.Error("Failed to create database", "error", createErr)
				return fmt.Errorf("failed to create database '%s': %w", dbName, createErr)
			}

			fmt.Printf("✅ Database '%s' created\n", dbName)

			// Retry collection creation
			id, err = chromaClient.CreateCollection(ctx, collectionName)
			if err != nil {
				slog.Error("operation_failed", "op", opName, "name", collectionName, "error", err, "duration_ms", time.Since(start).Milliseconds())
				return fmt.Errorf("failed to create collection after database creation: %w", err)
			}
		} else if isDatabaseError {
			// Provide helpful error message with available databases
			dbs, listErr := chromaClient.ListDatabases(ctx)

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
func handleDeleteCollection(ctx context.Context, cmd *cli.Command) error {
	collectionName := cmd.Args().Get(0)
	if collectionName == "" {
		return fmt.Errorf("collection name is required\n\nUsage: chroma delete <collection_name>\n\nExample: chroma delete my_collection")
	}

	slog.Info("Deleting collection", "name", collectionName)

	f := factory.NewServiceFactory()

	chromaClient, err := f.CreateChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	if err := chromaClient.DeleteCollection(ctx, collectionName); err != nil {
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
func handleListDocuments(ctx context.Context, c *cli.Command) error {
	collectionName := c.Args().Get(0)
	if collectionName == "" {
		return fmt.Errorf("collection name is required\n\nUsage: chroma records <collection_name>\n\nExample: chroma records my_collection")
	}

	slog.Info("Listing documents", "collection", collectionName, "tenant", c.String("tenant"), "database", c.String("database"))

	f := factory.NewServiceFactory()

	svc, _, cleanup, err := f.CreateChromaService(c)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer cleanup()

	docs, err := svc.GetDocuments(ctx, collectionName)
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
func handleBatchAddDocuments(ctx context.Context, c *cli.Command) error {
	collectionName := c.Args().Get(0)
	docs := c.StringSlice("doc")

	if len(docs) == 0 {
		return fmt.Errorf("no documents provided\n\nUsage: chroma add <collection> --doc \"text\"\n\nExample: chroma add my_collection --doc \"Your document\"")
	}

	// Setup service using factory
	f := factory.NewServiceFactory()

	svc, _, cleanup, err := f.CreateChromaService(c)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer cleanup()

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

		if err := svc.UpsertDocuments(ctx, collectionName, docs, ids); err != nil {
			return fmt.Errorf("failed to upsert documents: %w\n\nHint: Check collection exists and you have write permissions", err)
		}

		fmt.Printf("✅ Successfully upserted %d documents to '%s'\n", len(docs), collectionName)
		slog.Info("Documents upserted", "collection", collectionName, "count", len(docs))
	} else {
		slog.Info("Uploading batch to Chroma", "collection", collectionName, "count", len(docs))

		if err := svc.AddDocuments(ctx, collectionName, docs, ids); err != nil {
			return fmt.Errorf("failed to add documents: %w\n\nHint: Check collection exists and you have write permissions", err)
		}

		fmt.Printf("✅ Successfully added %d documents to '%s'\n", len(docs), collectionName)
		slog.Info("Documents added", "collection", collectionName, "count", len(docs))
	}

	return nil
}

// handleDeleteRecords deletes specific documents from a collection by IDs.
func handleDeleteRecords(ctx context.Context, cmd *cli.Command) error {
	collectionName := cmd.Args().Get(0)
	ids := cmd.StringSlice("id")

	if collectionName == "" {
		return fmt.Errorf("collection name is required\n\nUsage: chroma delete-records <collection> --id <id>\n\nExample: chroma delete-records my_collection --id doc-1")
	}

	if len(ids) == 0 {
		return fmt.Errorf("at least one document ID is required\n\nUsage: chroma delete-records <collection> --id <id>")
	}

	slog.Info("Deleting records", "collection", collectionName, "ids", ids)

	f := factory.NewServiceFactory()

	chromaClient, err := f.CreateChromaClient(cmd)
	if err != nil {
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}

	if err := chromaClient.DeleteRecords(ctx, collectionName, ids); err != nil {
		slog.Error("Delete records failed", "error", err)
		return fmt.Errorf("failed to delete records: %w", err)
	}

	fmt.Printf("✅ Deleted %d document(s) from '%s'\n", len(ids), collectionName)
	slog.Info("Records deleted", "collection", collectionName, "ids", ids)

	return nil
}

// ===== Search & Query =====

// handleQueryBatchInCollection performs batch semantic search.
func handleQueryBatchInCollection(ctx context.Context, c *cli.Command) error {
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

	// Setup service using factory
	f := factory.NewServiceFactory()

	svc, _, cleanup, err := f.CreateChromaService(c)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer cleanup()

	// Show loading indicator in interactive mode
	ui := output.NewUI()
	if ui.IsInteractive() && !c.Bool("no-tui") {
		printer.Info("Searching...")
	}

	response, err := svc.QueryDocuments(ctx, collectionName, queries, nResults)
	if err != nil {
		slog.Error("operation_failed", "op", opName, "collection", collectionName, "error", err, "duration_ms", time.Since(start).Milliseconds())
		return fmt.Errorf("query failed: %w\n\nHint: Check collection exists and contains documents", err)
	}

	// Display results using formatter
	formatter := output.NewFormatter(os.Stdout, output.ModeFromCLI(c))
	formatter.FormatQueryResponse(collectionName, queries, response)

	slog.Info("operation_complete", "op", opName, "collection", collectionName, "result_count", len(response.IDs[0]), "duration_ms", time.Since(start).Milliseconds())

	return nil
}

// ===== Import =====

// handleImportFileInChromaDb imports a file (JSONL or Parquet) with progress tracking.
// All batching logic is handled by the service layer.
func handleImportFileInChromaDb(ctx context.Context, c *cli.Command) error {
	collectionName := c.Args().Get(0)
	if collectionName == "" {
		return fmt.Errorf("collection name is required\n\nUsage: chroma import <collection> <file.jsonl|file.parquet|url>\n\nExample: chroma import my_collection https://example.com/data.parquet")
	}

	filePathArg := c.Args().Get(1)
	if filePathArg == "" {
		return fmt.Errorf("file path is required\n\nUsage: chroma import <collection> <file.jsonl|file.parquet|url>")
	}

	var (
		safePath string
		cleanup  func()
	)

	if ingest.IsURL(filePathArg) {
		printer.Printf("⬇ Downloading from '%s'\n", filePathArg)

		tmpPath, err := ingest.DownloadFile(ctx, filePathArg, func(downloaded, total int64) {
			if total > 0 {
				pct := float64(downloaded) / float64(total) * 100
				printer.Printf("\r   %s / %s (%.0f%%)\033[K", ingest.FormatBytes(downloaded), ingest.FormatBytes(total), pct)
			} else {
				printer.Printf("\r   %s downloaded\033[K", ingest.FormatBytes(downloaded))
			}
		})
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		printer.Print("")

		cleanup = func() {
			if err := os.Remove(tmpPath); err != nil {
				slog.Warn("Failed to remove temp file", "path", tmpPath, "error", err)
			}
		}
		safePath = tmpPath
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		safePath, err = internal.SafeJoin(cwd, filePathArg)
		if err != nil {
			return fmt.Errorf("invalid file path: %w\n\nHint: Use relative paths within the current directory", err)
		}

		cleanup = func() {}
	}

	defer cleanup()

	// Build ingest config
	cfg := &ingest.Config{
		BatchSize:      int(c.Int("batch-size")),
		ContentField:   c.String("field-content"),
		IDField:        c.String("field-id"),
		MetadataFields: c.StringSlice("field-metadata"),
		AllMetadata:    c.Bool("all-metadata"),
		Limit:          c.Int("n-ingest"),
		AutoID:         c.Bool("auto-id"),
		DedupMode:      c.String("dedup"),
		Upsert:         c.Bool("upsert"),
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}

	if cfg.ContentField == "" {
		cfg.ContentField = "text"
	}

	if cfg.IDField == "" {
		cfg.IDField = "id"
	}

	// Validate DedupMode
	switch cfg.DedupMode {
	case "none", "warn", "skip":
		// valid
	default:
		return fmt.Errorf("invalid --dedup value: %q (must be: none, warn, or skip)", cfg.DedupMode)
	}

	// Get parquet row count for progress bar
	if strings.HasSuffix(safePath, ".parquet") {
		cfg.Total = ingest.ParquetRowCount(safePath)
	}

	// Setup service using factory
	f := factory.NewServiceFactory()

	svc, _, cleanup, err := f.CreateChromaService(c)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer cleanup()

	// Build output config and progress display
	outputCfg := output.NewOutputConfig(c)
	progress := output.NewIngestProgress(outputCfg, collectionName, filepath.Base(safePath))
	cfg.OnProgress = progress.Update

	// Print import info using printer
	printer.Printf("📥 Importing from '%s' to collection '%s'\n", filepath.Base(safePath), collectionName)
	printer.Printf("   Batch size: %d\n", cfg.BatchSize)

	if cfg.Total > 0 {
		printer.Printf("   Total: %d records\n", cfg.Total)
	}

	if cfg.Limit > 0 {
		printer.Printf("   Limit: %d documents\n", cfg.Limit)
	}

	printer.Print("")

	slog.Info("Starting import", "file", safePath, "collection", collectionName)

	if err := svc.IngestRecords(ctx, collectionName, safePath, cfg); err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

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

	if _, err := validateNIMModel(model); err != nil {
		return err
	}

	nResults := c.Int("n-results")
	if nResults <= 0 {
		nResults = 3
	}

	slog.Info("operation_start", "op", opName, "collection", collectionName, "model", model)

	// Setup service and LLM using factories
	f := factory.NewServiceFactory()

	svc, _, cleanup, err := f.CreateChromaService(c)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer cleanup()

	llmFactory := factory.NewLLMProviderFactory()
	nimURL := c.String("nim-url")

	provider, err := llmFactory.CreateProvider(model, nimURL)
	if err != nil {
		return err
	}

	// We can't use ProviderInterface directly for chains,
	// but we know our providers now use LangChainAdapter.
	// For this MVP, we'll use the existing RAG pipeline logic but
	// move towards chains in the next sub-task.

	fmt.Printf("\n🤖 Querying collection '%s' with: %s\n\n", collectionName, question)

	resp, err := svc.QueryDocuments(ctx, collectionName, []string{question}, nResults)
	if err != nil {
		return fmt.Errorf("failed to retrieve context: %w", err)
	}

	// Use existing prompt logic
	distanceThreshold := c.Float64("distance-threshold")
	contextStr, hasRelevant, _ := buildContextFromResponse(resp, distanceThreshold)
	finalPrompt := buildRAGPrompt(question, contextStr, hasRelevant)

	if err := provider.Generate(ctx, finalPrompt, model, printer.Stdout()); err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	slog.Info("operation_complete", "op", opName, "duration_ms", time.Since(start).Milliseconds())

	return nil
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
		fmt.Fprintf(&sb, "[Context %d]: %s\n", i+1, doc)
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

// ============ Config Command Handlers ============

// handleConfigShow displays the effective configuration after merging all sources.
func handleConfigShow(ctx context.Context, c *cli.Command) error {
	slog.Info("Displaying effective configuration")

	// Load config from all sources
	cfg, err := config.LoadConfig(c)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine output format
	outputFormat := c.String("output")
	if outputFormat == "" {
		outputFormat = "yaml" // default format
	}

	switch outputFormat {
	case "json":
		// Output as JSON
		jsonData, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config to JSON: %w", err)
		}

		fmt.Println(string(jsonData))
	case "env":
		// Output as environment variables
		fmt.Printf("# Effective configuration as environment variables\n")
		fmt.Printf("CHROMA_HOST=%s\n", cfg.Chroma.Host)
		fmt.Printf("CHROMA_PORT=%s\n", cfg.Chroma.Port)
		fmt.Printf("CHROMA_TENANT=%s\n", cfg.Chroma.Tenant)
		fmt.Printf("CHROMA_DATABASE=%s\n", cfg.Chroma.Database)
		fmt.Printf("CHROMA_LOG_LEVEL=%s\n", cfg.Logging.Level)
		fmt.Printf("CHROMA_LOG_FORMAT=%s\n", cfg.Logging.Format)
		fmt.Printf("CHROMA_MODEL_PATH=%s\n", cfg.Model.ONNXModel)
		fmt.Printf("CHROMA_TOKENIZER_PATH=%s\n", cfg.Model.Tokenizer)
		fmt.Printf("CHROMA_ONNX_LIB=%s\n", cfg.Model.ONNXLib)
	default:
		// Output as YAML (default)
		yamlData, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config to YAML: %w", err)
		}

		fmt.Println(string(yamlData))
	}

	return nil
}

// handleConfigInit creates a configuration file with sensible defaults.
func handleConfigInit(ctx context.Context, c *cli.Command) error {
	slog.Info("Initializing configuration file")

	// Determine output path
	outputPath := c.String("output")
	global := c.Bool("global")

	// Set default output path based on flags
	if outputPath == "" {
		if global {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			outputPath = filepath.Join(homeDir, ".config", "cmdChroma", "config.yaml")
		} else {
			// Default to local (either --local specified or neither --global nor --local)
			outputPath = "./.cmdChroma.yaml"
		}
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s\nUse --output to specify a different location or remove the existing file first", outputPath)
	}

	// Create default configuration
	defaultConfig := config.ConfigFile{
		Version: "1.0",
		Chroma: config.ConfigFileChroma{
			Host:     "localhost",
			Port:     "8000",
			Tenant:   "default_tenant",
			Database: "default_database",
			Timeout:  30,
		},
		Model: config.ConfigFileModel{
			ONNXModel: "models/all-MiniLM-L6-v2/model.onnx",
			Tokenizer: "models/all-MiniLM-L6-v2/tokenizer.json",
			ONNXLib:   "models/onnx_runtime/lib/libonnxruntime.so",
		},
		Logging: config.ConfigFileLogging{
			Level:   "info",
			Format:  "text",
			Verbose: false,
		},
		Features: config.ConfigFileFeatures{
			CreateCollection: config.ConfigFileCreateCollection{
				AutoCreateDatabase: false,
			},
		},
	}

	// Create directory if needed
	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Write configuration file
	yamlData, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config to YAML: %w", err)
	}

	if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file %s: %w", outputPath, err)
	}

	printer.Success("Configuration file created at %s", outputPath)
	printer.Info("Edit this file to customize your settings")
	printer.Info("Use 'chroma config show' to view effective configuration")

	return nil
}

// ============ Helper Functions ============

// toStringRows converts a list of Database to table rows.
func toStringRows(dbs []client.Database) [][]string {
	rows := make([][]string, len(dbs))
	for i, db := range dbs {
		rows[i] = []string{db.Name, db.Id}
	}

	return rows
}

// handleDoctor runs diagnostic checks and reports system status.
func handleDoctor(ctx context.Context, c *cli.Command) error {
	// Header
	printer.Print("🔍 Running diagnostics...")
	printer.Print(strings.Repeat("=", 60))
	printer.Print("")

	// 1. Show configuration
	printer.Info("Configuration:")
	printer.Printf("  Host: %s\n", c.String("host"))
	printer.Printf("  Port: %s\n", c.String("port"))
	printer.Printf("  Tenant: %s\n", c.String("tenant"))
	printer.Printf("  Database: %s\n", c.String("database"))
	printer.Print("")

	// 2. Check model files
	printer.Info("Checking model files...")

	_, currentFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(currentFile)
	projectRoot := filepath.Join(dir, "..", "..")

	modelPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "model.onnx")
	tokenizerPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "tokenizer.json")
	libPath := filepath.Join(projectRoot, "models", "onnx_runtime", "lib", "libonnxruntime.so.1")

	checkFile := func(path, desc string) {
		if _, err := os.Stat(path); err == nil {
			printer.Success("  ✓ %s: present", desc)
		} else {
			printer.Error("  ✗ %s: NOT FOUND (%s)", desc, path)
		}
	}

	checkFile(modelPath, "Model file (model.onnx)")
	checkFile(tokenizerPath, "Tokenizer (tokenizer.json)")
	checkFile(libPath, "ONNX Runtime library (libonnxruntime.so.1)")
	printer.Print("")

	// 3. Environment variables
	printer.Info("Environment variables:")

	if os.Getenv("NVIDIA_API_KEY") != "" {
		printer.Success("  ✓ NVIDIA_API_KEY is set")
	} else {
		printer.Warn("  ⚠ NVIDIA_API_KEY not set (only needed for NVIDIA NIM)")
	}

	printer.Print("")

	// 4. ChromaDB connectivity
	printer.Info("Testing ChromaDB connectivity...")

	f := factory.NewServiceFactory()

	chromaClient, err := f.CreateChromaClient(c)
	if err != nil {
		printer.Error("  ✗ Failed to create client: %v", err)
	} else {
		if err := chromaClient.TestConnection(ctx); err != nil {
			printer.Error("  ✗ Connection failed: %v", err)
		} else {
			printer.Success("  ✓ Successfully connected to ChromaDB")
		}
	}

	printer.Print("")

	// Summary
	printer.Print(strings.Repeat("=", 60))
	printer.Success("Diagnostics complete!")

	if outPath := c.String("output"); outPath != "" {
		printer.Info("Note: --output flag not yet implemented; report shown above")
	}

	return nil
}
