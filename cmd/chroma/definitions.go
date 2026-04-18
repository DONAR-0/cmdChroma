package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/donar0/cmdChroma/internal/version"
	"github.com/urfave/cli/v3"
)

func createApp() *cli.Command {
	return &cli.Command{
		Name:    "cmdChroma",
		Version: fmt.Sprintf("%s (git %s, built %s)", AppVersion, version.GitCommit, version.BuildDate),
		Usage:   "Command Line Inteface for Chroma DB Operations",
		Description: "A Comprehensive CLI Tool to interact with Chroma DB," +
			"including connection testing, data operations, and more",
		// Global Flags Available to all command
		Flags: []cli.Flag{hostFlag, portFlag, verboseFlag, tenantFlag, databaseFlag, collectionFlag},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if c.Bool("verbose") {
				slog.Info("Verbose mode enabled")
			}
			return ctx, nil
		},
		Commands: []*cli.Command{
			testCommandDefinition,
			getTenantDefinition,
			listDatabaseDefinition,
			listCollectionsInDatabaseDefinition,
			getOrCreateCollectionInDatabaseDefinition,
			listRecordsInCollection,
			AddBatchDocumentCommandDefinition(),
			QueryDocumentCommandDefinition(),
			ingestRecordsJsonlfile,
		},

		//Default action when no command is provided
		Action: func(ctx context.Context, c *cli.Command) error {
			return cli.ShowAppHelp(c)
		},
	}
}

func QueryDocumentCommandDefinition() *cli.Command {
	name := "query"
	usage := "Search for documents using one or more natural language queries"
	return &cli.Command{
		Name:      name,
		Aliases:   []string{"q", "search"},
		Usage:     usage,
		ArgsUsage: collection_args_usage,
		Description: `Perform a semantic search against a collection. 
You can provide multiple queries to perform a batch search in a single execution.
The tool will vectorize each query locally and find the most similar documents.

EXAMPLES:
   # Single query search
   chroma query my_collection --query "What is Go?"

   # Batch query (multiple searches at once)
   chroma query my_collection -q "How to use ONNX?" -q "Vector database basics"

   # Search with limited results (Top 3)
   chroma query my_collection --query "Wikipedia facts" --n-results 3`,
		Action: handleQueryBatchInCollection,
		Flags: []cli.Flag{
			queryFlag,
			nResultsFlag,
			modelOnnxFileFlag,
			tokenizerJsonFileFlag,
			onnxLibFlag,
		},
	}
}

func AddBatchDocumentCommandDefinition() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Aliases:   []string{"a", "insert"},
		Usage:     "Add one or more documents to a collection",
		ArgsUsage: "<collection_name>",
		Description: `Add text documents to a collection. 
You can provide multiple --doc flags to insert a batch of documents at once. 
If IDs are not provided, unique IDs will be auto-generated.

EXAMPLES:
   # Add a single document
   chroma add my_collection --doc "The capital of France is Paris."

   # Add multiple documents (Batch)
   chroma add my_collection -d "Go is a compiled language." -d "Python is interpreted."

   # Add with specific IDs
   chroma add my_collection -d "Text 1" --id "id-01" -d "Text 2" --id "id-02"`,
		Action: handleBatchAddDocuments,
		Flags: []cli.Flag{
			docSliceFlag,
			idSliceFlag,
			modelOnnxFileFlag,
			tokenizerJsonFileFlag,
			onnxLibFlag,
		},
	}
}

// Commands
var (
	getTenantDefinition = &cli.Command{
		Name:    "tenant", // Better to use lowercase short names for CLI commands
		Aliases: []string{"currentTenant", "cT"},
		Usage:   "Current Tenant in Chroma DB",
		Action:  handleCurrentTenants,
	}

	listDatabaseDefinition = &cli.Command{
		Name:    "databases",
		Aliases: []string{"ls-dbs", "dbs"},
		Usage:   "List Databases in current Tenant",
		Action:  handleListDatabases,
	}

	testCommandDefinition = &cli.Command{
		Name:    "testConnection",
		Aliases: []string{"test", "t"},
		Usage:   "Test the connection to Chroma DB",
		Description: "Verifies connectivity to the DB instance and " +
			"ensures the service is responding correctly",
		Action: handleTestConnection,
		Flags:  []cli.Flag{timeoutFlag},
	}

	listCollectionsInDatabaseDefinition = &cli.Command{
		Name:    "collections",
		Aliases: []string{"ls-colls", "colls"},
		Usage:   "List All the collections in database",
		Action:  handleListCollection,
	}

	getOrCreateCollectionInDatabaseDefinition = &cli.Command{
		Name:    "createCollections",
		Aliases: []string{"mkdir-colls", "mkColl"},
		Usage:   "Create the collections in database",
		Action:  handleCreateCollection,
	}

	listRecordsInCollection = &cli.Command{
		Name:      "records",
		Aliases:   []string{"ls-rs", "rs"},
		Usage:     "List All the records in database",
		ArgsUsage: collection_args_usage,
		Action:    handleListDocuments,
	}

	ingestRecordsJsonlfile = &cli.Command{
		Name:    "import",
		Aliases: []string{"ingest", "jsonl"},
		Usage:   "Ingest a .jsonl file into collection",
		Action:  handleImportJsonlFileInChromaDb,
		Flags: []cli.Flag{nIngestDocumentFlag,
			fieldContentFlag, fieldIdFlag,
			fieldMetadataFlag, allMetadataFlag,
			batchSizeFlag},
	}
)

// Flags
var (
	tenantFlag = &cli.StringFlag{
		Name:    "tenant",
		Value:   "default_tenant",
		Usage:   "Chroma DB Tenant",
		Sources: cli.EnvVars("TENANT"),
	}

	databaseFlag = &cli.StringFlag{
		Name:    "database",
		Value:   "default_database",
		Usage:   "Chroma DB database",
		Sources: cli.EnvVars("DATABASE"),
	}

	collectionFlag = &cli.StringFlag{
		Name:    "collection",
		Usage:   "Chroma DB collection",
		Sources: cli.EnvVars("COLLECTION"),
	}

	hostFlag = &cli.StringFlag{
		Name:    "host",
		Aliases: []string{"H"},
		Value:   "localhost",
		Usage:   "Chroma DB host address",
		Sources: cli.EnvVars("CHROMA_HOST"),
	}

	portFlag = &cli.StringFlag{
		Name:    "port",
		Aliases: []string{"p"},
		Value:   "8000",
		Usage:   "Chroma DB port number",
		Sources: cli.EnvVars("CHROMA_PORT"),
	}

	verboseFlag = &cli.BoolFlag{
		Name:  "verbose",
		Usage: "Enable verbose logging",
	}

	timeoutFlag = &cli.IntFlag{
		Name:  "timeout",
		Value: 30,
		Usage: "Connection timeout in seconds",
	}

	modelOnnxFileFlag = &cli.StringFlag{
		Name:    "model-path",
		Usage:   "Path to the model.onnx file",
		Sources: cli.EnvVars("CHROMA_MODEL_PATH"),
	}

	tokenizerJsonFileFlag = &cli.StringFlag{
		Name:    "tokenizer-path", // Changed from "model" to "tokenizer-path"
		Usage:   "Path to tokenizer.json",
		Sources: cli.EnvVars("CHROMA_TOKENIZER_PATH"),
	}

	onnxLibFlag = &cli.StringFlag{
		Name:    "onnx-lib",
		Usage:   "Path to libonnxruntime.so",
		Sources: cli.EnvVars("CHROMA_ONNX_LIB"),
	}

	queryFlag = &cli.StringSliceFlag{
		Name:     "query",
		Aliases:  []string{"q"},
		Usage:    "Query text (can be repeated for batch query)",
		Required: true,
	}

	nResultsFlag = &cli.IntFlag{
		Name:    "n-results",
		Aliases: []string{"n"},
		Usage:   "Number of results to return per query",
		Value:   5,
	}

	docSliceFlag = &cli.StringSliceFlag{
		Name:     "doc",
		Aliases:  []string{"d"},
		Usage:    "The text document to add (can be repeated for batch add)",
		Required: true,
	}
	idSliceFlag = &cli.StringSliceFlag{

		Name:    "id",
		Aliases: []string{"i"},
		Usage:   "Optional: Custom IDs for the documents (must match the number of documents)",
	}

	nIngestDocumentFlag = &cli.IntFlag{
		Name:    "n-ingest",
		Aliases: []string{"l"},
		Usage:   "Set Limit to number of documents to ingest",
	}

	fieldContentFlag = &cli.StringFlag{
		Name:    "field-content",
		Aliases: []string{"content"},
		Usage:   "Content key in Jsonl Schema",
	}

	fieldIdFlag = &cli.StringFlag{
		Name:    "field-id",
		Aliases: []string{"id-scheama"},
		Usage:   "ID key in Jsonl Schema",
	}

	// CHANGED: StringSlice allows multiple metadata fields
	fieldMetadataFlag = &cli.StringSliceFlag{
		Name:  "field-metadata",
		Usage: "Specific metadata fields to import (can be used multiple times)",
	}

	// NEW: The "Generic" Toggle
	allMetadataFlag = &cli.BoolFlag{
		Name:  "all-metadata",
		Usage: "If true, all fields except content and ID will be stored as metadata",
	}

	// NEW: Batching control
	batchSizeFlag = &cli.IntFlag{
		Name:  "batch-size",
		Value: 100,
		Usage: "Number of documents to send in a single batch",
	}
)

const (
	collection_args_usage = "<collection_name>"
)
