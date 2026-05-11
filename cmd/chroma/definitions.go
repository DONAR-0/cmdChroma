package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/donar0/cmdChroma/internal/version"
	"github.com/urfave/cli/v3"
)

// createApp builds the root CLI application with all commands and flags.
// It follows clig.dev guidelines for a consistent, user-friendly CLI experience.
func createApp() *cli.Command {
	return &cli.Command{
		Name:    "cmdChroma",
		Version: fmt.Sprintf("%s (git %s, built %s)", AppVersion, version.GitCommit, version.BuildDate),
		Usage:   "A high-performance CLI for ChromaDB with local AI embeddings",
		Description: `cmdChroma is a command-line tool for managing ChromaDB collections
with local vector embeddings using ONNX Runtime. It keeps your data and AI
processing entirely on your local machine.

Use it to:
  • Test connectivity to your ChromaDB instance
  • Create and manage collections
  • Ingest documents from JSONL or Parquet files
  • Add documents with local embedding generation
  • Perform semantic search with batch queries
  • Chat with your data using RAG (Retrieval-Augmented Generation)

Get started quickly:
  chroma ping                    # Test connection
  chroma create my_collection    # Create a collection
  chroma add my_collection -d "Your document text"  # Add a document
  chroma query my_collection -q "Search query"      # Search`,
		// Global flags available to all commands
		Flags: []cli.Flag{hostFlag, portFlag, verboseFlag, tenantFlag, databaseFlag, collectionFlag},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
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
			importJsonlCommand,
			importParquetCommand,
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

// pingCommand tests connectivity to the ChromaDB server.
var pingCommand = &cli.Command{
	Name:    "ping",
	Aliases: []string{"t", "test", "health-check"},
	Usage:   "Test connection to ChromaDB",
	Description: `Verify that cmdChroma can connect to your ChromaDB instance.

This command sends a simple request to confirm:
  • The server is running and reachable
  • Network connectivity is working
  • Authentication (if any) is configured correctly

EXAMPLES:
  # Basic connectivity test
  chroma ping

  # Test with custom host/port
  chroma ping --host localhost --port 8000

  # Test with a timeout
  chroma ping --timeout 10`,
	Action: handleTestConnection,
	Flags:  []cli.Flag{timeoutFlag},
}

// tenantCommand shows the current tenant.
var tenantCommand = &cli.Command{
	Name:    "tenant",
	Aliases: []string{"cT", "current-tenant"},
	Usage:   "Show current tenant information",
	Description: `Display information about the configured tenant.

This command shows whether the specified tenant exists in your ChromaDB instance.
It's useful for verifying your tenant configuration.

EXAMPLES:
  # Show default tenant
  chroma tenant

  # Check a specific tenant
  chroma tenant --tenant my_tenant`,
	Action: handleCurrentTenants,
}

// databasesCommand lists all databases in the current tenant.
var databasesCommand = &cli.Command{
	Name:    "databases",
	Aliases: []string{"ls-dbs", "dbs"},
	Usage:   "List all databases in the current tenant",
	Description: `List all databases accessible to the current tenant.

This command retrieves and displays all databases you can access
within the tenant context.

EXAMPLES:
  # List all databases
  chroma databases

  # List databases for a specific tenant
  chroma databases --tenant custom_tenant`,
	Action: handleListDatabases,
}

// collectionsCommand lists collections in a database.
var collectionsCommand = &cli.Command{
	Name:    "collections",
	Aliases: []string{"ls", "list", "colls", "ls-colls"},
	Usage:   "List all collections in a database",
	Description: `List all collections in the specified database.

Shows collection names and IDs for the current database context.
Useful for exploring available data before querying.

EXAMPLES:
  # List collections in default database
  chroma collections

  # List collections in a specific database and tenant
  chroma collections --database my_db --tenant my_tenant

  # Show all collections in current context
  chroma collections`,
	Action: handleListCollection,
}

// createCollectionCommand creates a new collection.
var createCollectionCommand = &cli.Command{
	Name:    "create",
	Aliases: []string{"mkdir", "mkColl"},
	Usage:   "Create a new collection",
	Description: `Create a new collection in the current database.

Collections are where your embedded documents are stored. You need to
create a collection before adding documents.

EXAMPLES:
  # Create a simple collection
  chroma create my_collection

  # Create with explicit tenant and database
  chroma create my_collection --tenant my_tenant --database my_db`,
	Action: handleCreateCollection,
}

// deleteCollectionCommand deletes an existing collection.
var deleteCollectionCommand = &cli.Command{
	Name:      "delete",
	Aliases:   []string{"rm", "del"},
	Usage:     "Delete a collection",
	ArgsUsage: "<collection_name>",
	Description: `Permanently delete a collection and all its documents.

This operation cannot be undone. The collection and all its data will be removed.

EXAMPLES:
  # Delete a collection
  chroma delete my_collection

  # Delete with explicit tenant and database
  chroma delete my_collection --tenant my_tenant --database my_db`,
	Action: handleDeleteCollection,
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

// importJsonlCommand ingests a JSONL file into a collection.
var importJsonlCommand = &cli.Command{
	Name:      "import",
	Aliases:   []string{"ingest", "jsonl-import"},
	Usage:     "Import documents from a JSONL file into a collection",
	ArgsUsage: "<collection_name> <file_path>",
	Description: `Bulk import documents from a JSONL (JSON Lines) file.

This command is optimized for large datasets and will:
  • Stream the file line by line (memory efficient)
  • Generate embeddings locally for each document
  • Upload in configurable batches
  • Show progress during import

JSONL format: Each line must be a valid JSON object.
Common schemas: Hugging Face datasets, custom exports.

EXAMPLES:
  # Basic import with default field names (text, id)
  chroma import my_collection data.jsonl

  # Import with custom field mapping
  chroma import my_collection data.jsonl \\
    --field-content "content" \\
    --field-id "uuid"

  # Import with metadata and limit
  chroma import my_collection data.jsonl \\
    --field-metadata "author" \\
    --field-metadata "category" \\
    --all-metadata \\
    --n-ingest 1000

  # Import with custom batch size
  chroma import my_collection large_data.jsonl \\
    --batch-size 500 \\
    --all-metadata`,
	Action: handleImportJsonlFileInChromaDb,
	Flags: []cli.Flag{
		nIngestDocumentFlag,
		fieldContentFlag,
		fieldIdFlag,
		fieldMetadataFlag,
		allMetadataFlag,
		batchSizeFlag,
	},
}

// importParquetCommand ingests a Parquet file into a collection.
var importParquetCommand = &cli.Command{
	Name:      "import-parquet",
	Aliases:   []string{"parquet-import"},
	Usage:     "Import documents from a Parquet file into a collection",
	ArgsUsage: "<collection_name> <file_path>",
	Description: `Bulk import documents from a Parquet file.

Similar to 'import' but optimized for Parquet format.
Efficiently reads columnar data and generates embeddings in batches.

EXAMPLES:
  # Import with explicit column mapping
  chroma import-parquet my_collection data.parquet \\
    --text-column "text_content" \\
    --id-column "record_id"

  # Import with all other columns as metadata
  chroma import-parquet my_collection data.parquet \\
    --text-column "content" \\
    --id-column "id" \\
    --all-metadata \\
    --batch-size 200`,
	Action: handleImportParquetFileInChromaDb,
	Flags: []cli.Flag{
		nIngestDocumentFlag,
		fieldMetadataFlag,
		allMetadataFlag,
		batchSizeFlag,
		parquetIDColumnFlag,
		parquetTextColumnFlag,
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
  3. Generates an answer using a local LLM (via Ollama)

Requires Ollama running locally (default: http://localhost:11434).

EXAMPLES:
  # Ask a question about your collection
  chroma chat my_collection "What are the main topics?"

  # Use a specific Ollama model
  chroma chat my_collection "Summarize this" --llm-model llama2

  # Retrieve more context for complex questions
  chroma chat my_collection "Explain in detail" --n-results 10`,
	Action: handleChat,
	Flags: []cli.Flag{
		nResultsFlag,
		llmModelFlag,
	},
}

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

	// Add flags
	docSliceFlag = &cli.StringSliceFlag{
		Name:     "doc",
		Aliases:  []string{"d", "document"},
		Usage:    "Document text to add (use multiple times for batch)",
		Required: true,
	}

	idSliceFlag = &cli.StringFlag{
		Name:    "id",
		Aliases: []string{"i"},
		Usage:   "Custom document IDs (must match number of documents)",
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

	// Parquet flags
	parquetIDColumnFlag = &cli.StringFlag{
		Name:  "id-column",
		Usage: "Parquet column name for document IDs",
	}

	parquetTextColumnFlag = &cli.StringFlag{
		Name:  "text-column",
		Usage: "Parquet column name for document text",
	}

	// Chat/RAG flags
	llmModelFlag = &cli.StringFlag{
		Name:    "llm-model",
		Aliases: []string{"model"},
		Value:   "qwen:0.5b",
		Usage:   "Ollama model to use for answer generation",
	}
)
