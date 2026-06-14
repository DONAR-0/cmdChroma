package main

// CommandDescription provides documentation for the root CLI command and subcommands.
// Descriptions follow the urfave/cli/v3 convention of mixing a short paragraph
// with examples.
const (
	// AppDescription is shown in root help (cmdChroma --help).
	AppDescription = `cmdChroma is a command-line tool for managing ChromaDB collections
with local vector embeddings using ONNX Runtime. It keeps your data and AI
processing entirely on your local machine.

Use it to:
  • Test connectivity to your ChromaDB instance
  • Create and manage collections
  • Ingest documents from JSONL files
  • Add documents with local embedding generation
  • Perform semantic search with batch queries
  • Chat with your data using RAG (Retrieval-Augmented Generation)

Get started quickly:
  chroma ping                    # Test connection
  chroma create my_collection    # Create a collection
  chroma add my_collection -d "Your document text"  # Add a document
  chroma query my_collection -q "Search query"      # Search`

	// ------------------------------------------------------------------
	// Helper / overview commands
	// ------------------------------------------------------------------

	// PingCmdDescription explains the purpose of the "ping" subcommand.
	PingCmdDescription = `Verify that cmdChroma can connect to your ChromaDB instance.

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
  chroma ping --timeout 10`

	// TenantCmdDescription provides help text for the "tenant" subcommand.
	TenantCmdDescription = `Display information about the configured tenant.

This command shows whether the specified tenant exists in your ChromaDB instance.
It's useful for verifying your tenant configuration.

EXAMPLES:
  # Show default tenant
  chroma tenant

  # Check a specific tenant
  chroma tenant --tenant my_tenant`

	// DatabasesCmdDescription documents the "databases" subcommand.
	DatabasesCmdDescription = `List all databases accessible to the current tenant.

This command retrieves and displays all databases you can access
within the tenant context.

EXAMPLES:
  # List all databases
  chroma databases

  # List databases for a specific tenant
  chroma databases --tenant custom_tenant`

	// CollectionsCmdDescription documents the "collections" subcommand.
	CollectionsCmdDescription = `List all collections in the specified database.

Shows collection names and IDs for the current database context.
Useful for exploring available data before querying.

EXAMPLES:
  # List collections in default database
  chroma collections

  # List collections in a specific database and tenant
  chroma collections --database my_db --tenant my_tenant

  # Show all collections in current context
  chroma collections`

	// ------------------------------------------------------------------
	// Collection lifecycle
	// ------------------------------------------------------------------

	// CreateCollectionCmdDescription documents creating a new collection.
	CreateCollectionCmdDescription = `Create a new collection in the current database.

Collections are where your embedded documents are stored. You need to
create a collection before adding documents.

EXAMPLES:
  # Create a simple collection
  chroma create my_collection

  # Create with explicit tenant and database
  chroma create my_collection --tenant my_tenant --database my_db`

	// DeleteCollectionCmdDescription documents deleting a collection.
	DeleteCollectionCmdDescription = `Permanently delete a collection and all its documents.

This operation cannot be undone. The collection and all its data will be removed.

EXAMPLES:
  # Delete a collection
  chroma delete my_collection

  # Delete with explicit tenant and database
  chroma delete my_collection --tenant my_tenant --database my_db`

	// ------------------------------------------------------------------
	// Document operations
	// ------------------------------------------------------------------

	// RecordsCmdDescription documents listing documents.
	RecordsCmdDescription = `List all documents stored in a collection.

Displays document IDs, content previews, and metadata.
This is useful for inspecting collection contents.

EXAMPLES:
  # List records in a collection
  chroma records my_collection

  # List with explicit tenant/database
  chroma records my_collection --tenant my_tenant --database my_db

  # Paginate through large collections
			` // trailing indentation is deliberate to match original source

	// DeleteRecordsCmdDescription documents removing records by ID.
	DeleteRecordsCmdDescription = `Delete documents from a collection by their IDs.

This operation removes one or more documents identified by their IDs.
IDs must match those assigned when the documents were added.

EXAMPLES:
  # Delete a single record
  chroma delete-records my_collection --id doc-1

  # Delete multiple records
			`

	// AddCmdDescription documents adding documents to a collection.
	AddCmdDescription = `Add text documents to a collection with automatic embedding.

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
  cat docs.txt | xargs -I {} chroma add my_collection --doc "{}"`

	// QueryCmdDescription documents semantic search.
	QueryCmdDescription = `Perform semantic search against a collection.

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
  chroma query my_collection -q "$(cat queries.txt)"`

	// ImportJSONLCmdDescription documents importing from JSONL.
	ImportJSONLCmdDescription = `Bulk import documents from a JSONL (JSON Lines) file.

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
    --all-metadata`

	// ChatCmdDescription documents RAG-based chat.
	ChatCmdDescription = `Ask questions about your data using AI-powered retrieval.

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
  chroma chat my_collection "Explain in detail" --n-results 10`

	// DoctorCmdDescription runs diagnostics to verify the installation and configuration.
	DoctorCmdDescription = `Check your cmdChroma setup and configuration.

This command performs a health check of your environment:
  • Verifies ChromaDB server connectivity
  • Checks for required model files and ONNX runtime library
  • Validates configuration (host, port, tenant, database)
  • Inspects relevant environment variables

Use this to troubleshoot setup issues or verify everything is ready.

EXAMPLES:
  # Run diagnostics
  chroma doctor

  # Save full report to a file
  chroma doctor --output doctor-report.txt
`
)
