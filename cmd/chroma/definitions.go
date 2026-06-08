package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DONAR-0/cmdChroma/internal/version"
	"github.com/urfave/cli/v3"
)

// ============ Application Setup ============

// createApp builds the root CLI application with all commands and flags.
// It follows clig.dev guidelines for a consistent, user-friendly CLI experience.
func createApp() *cli.Command {
	return &cli.Command{
		Name:        "cmdChroma",
		Version:     fmt.Sprintf("%s (git %s, built %s)", AppVersion, version.GitCommit, version.BuildDate),
		Usage:       "A high-performance CLI for ChromaDB with local AI embeddings",
		Description: AppDescription,
		// Global flags available to all commands
		Flags: []cli.Flag{hostFlag, portFlag, verboseFlag, logLevelFlag, logFormatFlag, tenantFlag, databaseFlag, collectionFlag},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			// Determine log level from flags
			level := slog.LevelInfo

			// Parse log-level flag
			logLevel := c.String("log-level")
			switch logLevel {
			case "debug":
				level = slog.LevelDebug
			case "warn":
				level = slog.LevelWarn
			case "error":
				level = slog.LevelError
			}

			// verbose flag overrides to debug
			if c.Bool("verbose") {
				level = slog.LevelDebug
			}

			// Parse log-format flag
			format := c.String("log-format")
			if format != "text" && format != "json" {
				format = "text" // default fallback
			}

			// Initialize the logger
			InitLogger(level, format)

			if c.Bool("verbose") {
				slog.Info("Verbose logging enabled", "version", AppVersion)
			}

			return ctx, nil
		},
		Commands: []*cli.Command{
			pingCommand,
			tenantCommand,
			databasesCommand,
			collectionsCommand,
			createCollectionCommand,
			deleteCollectionCommand,
			deleteRecordsCommand,
			recordsCommand,
			addCommand,
			queryCommand,
			importCommand,
			chatCommand,
		},

		// Default action when no command is provided: show help
		Action: func(ctx context.Context, c *cli.Command) error {
			fmt.Println()
			fmt.Println("Welcome to cmdChroma! 🚀")
			fmt.Println("A CLI for ChromaDB with local AI embeddings")
			fmt.Println()
			fmt.Println("Quick start:")
			fmt.Println("  chroma ping                    # Test connection to ChromaDB")
			fmt.Println("  chroma create <collection>     # Create a new collection")
			fmt.Println("  chroma add <collection> -d ... # Add documents")
			fmt.Println("  chroma query <collection> -q ... # Search documents")
			fmt.Println()
			fmt.Println("Run 'chroma --help' for more information or 'chroma <command> --help'")
			fmt.Println("for command-specific help.")
			fmt.Println()

			return cli.ShowAppHelp(c)
		},
	}
}

// ============ Command Definitions ============

// pingCommand tests connectivity to the ChromaDB server.
var pingCommand = &cli.Command{
	Name:        "ping",
	Aliases:     []string{"t", "test", "health-check"},
	Usage:       "Test connection to ChromaDB",
	Description: PingCmdDescription,
	Action:      handleTestConnection,
	Flags:       []cli.Flag{timeoutFlag},
}

// tenantCommand shows the current tenant.
var tenantCommand = &cli.Command{
	Name:        "tenant",
	Aliases:     []string{"cT", "current-tenant"},
	Usage:       "Show current tenant information",
	Description: TenantCmdDescription,
	Action:      handleCurrentTenants,
}

// databasesCommand lists all databases in the current tenant.
var databasesCommand = &cli.Command{
	Name:        "databases",
	Aliases:     []string{"ls-dbs", "dbs"},
	Usage:       "List all databases in the current tenant",
	Description: DatabasesCmdDescription,
	Action:      handleListDatabases,
}

// collectionsCommand lists collections in a database.
var collectionsCommand = &cli.Command{
	Name:        "collections",
	Aliases:     []string{"ls", "list", "colls", "ls-colls"},
	Usage:       "List all collections in a database",
	Description: CollectionsCmdDescription,
	Action:      handleListCollection,
}

// createCollectionCommand creates a new collection.
var createCollectionCommand = &cli.Command{
	Name:        "create",
	Aliases:     []string{"mkdir", "mkColl"},
	Usage:       "Create a new collection",
	Description: CreateCollectionCmdDescription,
	Action:      handleCreateCollection,
}

// deleteCollectionCommand deletes an existing collection.
var deleteCollectionCommand = &cli.Command{
	Name:        "delete",
	Aliases:     []string{"rm", "del"},
	Usage:       "Delete a collection",
	ArgsUsage:   "<collection_name>",
	Description: DeleteCollectionCmdDescription,
	Action:      handleDeleteCollection,
}

// recordsCommand lists documents in a collection.
var recordsCommand = &cli.Command{
	Name:      "records",
	Aliases:   []string{"ls-records", "list-docs", "rs"},
	Usage:     "List all documents in a collection",
	ArgsUsage: "<collection_name>",
	Description: `List all documents stored in a collection.

Displays document IDs, content previews, and metadata.
This is useful for inspecting collection contents.

EXAMPLES:
  # List records in a collection
  chroma records my_collection

  # List with explicit tenant/database
  chroma records my_collection --tenant my_tenant --database my_db

  # Paginate through large collections
		`,
	Action: handleListDocuments,
}

// deleteRecordsCommand deletes specific documents from a collection.
var deleteRecordsCommand = &cli.Command{
	Name:      "delete-records",
	Aliases:   []string{"del-rec", "rm-rec"},
	Usage:     "Delete specific records from a collection",
	ArgsUsage: "<collection_name>",
	Description: `Delete documents from a collection by their IDs.

This operation removes one or more documents identified by their IDs.
IDs must match those assigned when the documents were added.

EXAMPLES:
  # Delete a single record
  chroma delete-records my_collection --id doc-1

  # Delete multiple records
		`,
	Action: handleDeleteRecords,
	Flags: []cli.Flag{
		idSliceFlag,
	},
}

// addCommand adds one or more documents to a collection.
var addCommand = &cli.Command{
	Name:      "add",
	Aliases:   []string{"a", "insert", "push"},
	Usage:     "Add one or more documents to a collection",
	ArgsUsage: "<collection_name>",
	Description: `Add text documents to a collection with automatic embedding.

The CLI will:
  1. Tokenize your text
  2. Generate embeddings locally using ONNX
  3. Upload documents + embeddings to ChromaDB

You can add multiple documents in a single command (batch operation)
for better performance.

EXAMPLES:
  # Add a single document
  chroma add my_collection --doc "The capital of France is Paris."

  # Add multiple documents (batch)
  chroma add my_collection \\
    --doc "Go is a statically typed, compiled language." \\
    --doc "ChromaDB is an AI-native vector database."

  # Add with custom IDs
  chroma add my_collection \\
    --doc "Document 1" --id "doc-1" \\
    --doc "Document 2" --id "doc-2"

  # Add many documents from a file (using xargs)
  cat docs.txt | xargs -I {} chroma add my_collection --doc "{}"`,
	Action: handleBatchAddDocuments,
	Flags: []cli.Flag{
		docSliceFlag,
		idSliceFlag,
		upsertFlag,
		modelOnnxFileFlag,
		tokenizerJsonFileFlag,
		onnxLibFlag,
	},
}

// queryCommand searches a collection with one or more queries.
var queryCommand = &cli.Command{
	Name:      "query",
	Aliases:   []string{"q", "search", "find"},
	Usage:     "Search for documents using natural language queries",
	ArgsUsage: "<collection_name>",
	Description: `Perform semantic search against a collection.

The CLI will:
  1. Convert your query to an embedding locally
  2. Find the most similar documents in the collection
  3. Return results with similarity scores

You can provide multiple queries to perform batch search in a single
execution, which is much faster than running single queries repeatedly.

EXAMPLES:
  # Single query
  chroma query my_collection --query "What is Go programming?"

  # Multiple queries (batch search)
  chroma query my_collection \\
    --query "How to use vectors?" \\
    --query "What is RAG?"

  # Control number of results
  chroma query my_collection --query "AI concepts" --n-results 10

  # Batch query from a file
  chroma query my_collection -q "$(cat queries.txt)"`,
	Action: handleQueryBatchInCollection,
	Flags: []cli.Flag{
		queryFlag,
		nResultsFlag,
		modelOnnxFileFlag,
		tokenizerJsonFileFlag,
		onnxLibFlag,
	},
}

// importCommand ingests a file (JSONL or Parquet) into a collection.
var importCommand = &cli.Command{
	Name:      "import",
	Aliases:   []string{"ingest", "jsonl-import"},
	Usage:     "Import documents from a file (JSONL or Parquet) into a collection",
	ArgsUsage: "<collection_name> <file_path>",
	Description: `Bulk import documents from JSONL or Parquet files.

This command is optimized for large datasets and will:
  • Stream the file line by line (JSONL) or row by row (Parquet)
  • Generate embeddings locally for each document
  • Upload in configurable batches
  • Show progress during import

JSONL format: Each line must be a valid JSON object.
Parquet format: Column-based format; use --field-content and --field-id to map columns.

EXAMPLES:
  # Import JSONL
  chroma import my_collection data.jsonl --field-content text

  # Import Parquet
  chroma import my_collection data.parquet --field-content question --field-id conversation_id --all-metadata`,
	Action: handleImportFileInChromaDb,
	Flags: []cli.Flag{
		nIngestDocumentFlag,
		fieldContentFlag,
		fieldIdFlag,
		fieldMetadataFlag,
		allMetadataFlag,
		batchSizeFlag,
	},
}

// chatCommand performs RAG-based chat using collection context.
var chatCommand = &cli.Command{
	Name:      "chat",
	Aliases:   []string{"rag", "qa"},
	Usage:     "Chat with your collection using RAG (Retrieval-Augmented Generation)",
	ArgsUsage: "<collection_name> <question>",
	Description: `Ask questions about your data using AI-powered retrieval.

This command:
  1. Finds relevant documents from the collection
  2. Builds context from search results
  3. Generates an answer using a local LLM (via Ollama) or NVIDIA NIM (cloud API)

For Ollama:
  • Requires Ollama running locally (default: http://localhost:11434)
  • Pull a model: ollama pull qwen:0.5b

For NVIDIA NIM:
  • Set NVIDIA_API_KEY environment variable
  • Use model prefix: nim://<model-id> (e.g., nim://mistralai/mistral-7b-instruct-v0.3)
  • Default endpoint: https://integrate.api.nvidia.com/v1 (override with --nim-url)

Use --distance-threshold to filter out irrelevant matches. ChromaDB distances are usually 0-100+ (lower = more similar). If the best result's distance exceeds the threshold, the LLM will be told no context was found.

EXAMPLES:
  # Ask a question about your collection (Ollama)
  chroma chat my_collection "What are the main topics?"

  # Use a specific Ollama model
  chroma chat my_collection "Summarize this" --llm-model llama2

  # Use NVIDIA NIM (requires API key)
  export NVIDIA_API_KEY="your-key"
  chroma chat my_collection "Explain this" --llm-model nim://mistralai/mistral-7b-instruct-v0.3

  # Set a distance threshold to treat high-distance results as irrelevant
  chroma chat my_collection "Explain this" --distance-threshold 20

  # Retrieve more context for complex questions
  chroma chat my_collection "Explain in detail" --n-results 10`,
	Action: handleChat,
	Flags: []cli.Flag{
		nResultsFlag,
		distanceThresholdFlag,
		llmModelFlag,
		nimURLFlag,
	},
}

// ============ Flag Definitions ============

// Flags
var (
	tenantFlag = &cli.StringFlag{
		Name:    "tenant",
		Value:   "default_tenant",
		Usage:   "ChromaDB tenant to use",
		Sources: cli.EnvVars("TENANT", "CHROMA_TENANT"),
	}

	databaseFlag = &cli.StringFlag{
		Name:    "database",
		Value:   "default_database",
		Usage:   "ChromaDB database to use",
		Sources: cli.EnvVars("DATABASE", "CHROMA_DATABASE"),
	}

	collectionFlag = &cli.StringFlag{
		Name:    "collection",
		Usage:   "Default collection name (can be overridden per command)",
		Sources: cli.EnvVars("COLLECTION", "CHROMA_COLLECTION"),
	}

	hostFlag = &cli.StringFlag{
		Name:    "host",
		Aliases: []string{"H"},
		Value:   "localhost",
		Usage:   "ChromaDB server host",
		Sources: cli.EnvVars("CHROMA_HOST"),
	}

	portFlag = &cli.StringFlag{
		Name:    "port",
		Aliases: []string{"p"},
		Value:   "8000",
		Usage:   "ChromaDB server port",
		Sources: cli.EnvVars("CHROMA_PORT"),
	}

	verboseFlag = &cli.BoolFlag{
		Name:    "verbose",
		Aliases: []string{"v"},
		Usage:   "Enable verbose logging (debug level)",
	}

	logLevelFlag = &cli.StringFlag{
		Name:    "log-level",
		Aliases: []string{"l"},
		Value:   "info",
		Usage:   "Set log level: debug, info, warn, error",
		Sources: cli.EnvVars("CHROMA_LOG_LEVEL"),
	}

	logFormatFlag = &cli.StringFlag{
		Name:    "log-format",
		Value:   "text",
		Usage:   "Log output format: text, json",
		Sources: cli.EnvVars("CHROMA_LOG_FORMAT"),
	}

	timeoutFlag = &cli.IntFlag{
		Name:    "timeout",
		Aliases: []string{"t"},
		Value:   30,
		Usage:   "Connection timeout in seconds",
	}

	// AI/Embedding flags
	modelOnnxFileFlag = &cli.StringFlag{
		Name:    "model-path",
		Usage:   "Path to the ONNX model file (model.onnx)",
		Sources: cli.EnvVars("CHROMA_MODEL_PATH"),
	}

	tokenizerJsonFileFlag = &cli.StringFlag{
		Name:    "tokenizer-path",
		Usage:   "Path to the tokenizer.json file",
		Sources: cli.EnvVars("CHROMA_TOKENIZER_PATH"),
	}

	onnxLibFlag = &cli.StringFlag{
		Name:    "onnx-lib",
		Usage:   "Path to libonnxruntime.so",
		Sources: cli.EnvVars("CHROMA_ONNX_LIB"),
	}

	// Query flags
	queryFlag = &cli.StringSliceFlag{
		Name:     "query",
		Aliases:  []string{"q"},
		Usage:    "Query text (can be used multiple times for batch search)",
		Required: true,
	}

	nResultsFlag = &cli.IntFlag{
		Name:    "n-results",
		Aliases: []string{"n"},
		Value:   5,
		Usage:   "Number of results to return per query",
	}

	distanceThresholdFlag = &cli.Float64Flag{
		Name:  "distance-threshold",
		Value: 0,
		Usage: "Maximum distance for results to be considered relevant (0 = no threshold, use all results). ChromaDB distances are typically 0-100+; lower is more similar. Useful to filter irrelevant matches.",
	}

	// Add flags
	docSliceFlag = &cli.StringSliceFlag{
		Name:     "doc",
		Aliases:  []string{"d", "document"},
		Usage:    "Document text to add (use multiple times for batch)",
		Required: true,
	}

	idSliceFlag = &cli.StringSliceFlag{
		Name:    "id",
		Aliases: []string{"i"},
		Usage:   "Custom document IDs (must match number of documents)",
	}

	upsertFlag = &cli.BoolFlag{
		Name:  "upsert",
		Usage: "Update existing documents if IDs already exist (uses upsert endpoint)",
	}

	// Import flags
	nIngestDocumentFlag = &cli.IntFlag{
		Name:    "n-ingest",
		Aliases: []string{"l", "limit"},
		Usage:   "Maximum number of documents to ingest",
	}

	fieldContentFlag = &cli.StringFlag{
		Name:    "field-content",
		Aliases: []string{"content-field"},
		Usage:   "JSON field name for document content (default: 'text')",
	}

	fieldIdFlag = &cli.StringFlag{
		Name:    "field-id",
		Aliases: []string{"id-field"},
		Usage:   "JSON field name for document ID (default: 'id')",
	}

	fieldMetadataFlag = &cli.StringSliceFlag{
		Name:  "field-metadata",
		Usage: "Specific JSON field to include as metadata (can repeat)",
	}

	allMetadataFlag = &cli.BoolFlag{
		Name:  "all-metadata",
		Usage: "Import all fields except content and ID as metadata",
	}

	batchSizeFlag = &cli.IntFlag{
		Name:    "batch-size",
		Aliases: []string{"b"},
		Value:   100,
		Usage:   "Number of documents to process in each batch",
	}

	// Chat/RAG flags
	llmModelFlag = &cli.StringFlag{
		Name:    "llm-model",
		Aliases: []string{"model"},
		Value:   "qwen:0.5b",
		Usage:   "LLM model to use for answer generation. For Ollama: model name (e.g., qwen:0.5b). For NVIDIA NIM: prefix with nim:// and use exact model ID from /v1/models (e.g., nim://mistralai/mistral-7b-instruct-v0.3)",
	}

	nimURLFlag = &cli.StringFlag{
		Name:  "nim-url",
		Value: "https://integrate.api.nvidia.com/v1",
		Usage: "NVIDIA NIM API base URL (requires NVIDIA_API_KEY environment variable)",
	}
)
