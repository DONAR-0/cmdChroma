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
			TestCommandDefinition(),
			GetTenantDefinition(),
			ListDatabaseDefinition(),
			ListCollectionsInDatabaseDefinition(),
			GetOrCreateCollectionInDatabaseDefinition(),
			ListRecordsInCollection(),
			AddBatchDocumentCommandDefinition(),
			QueryDocumentCommandDefinition(),
		},

		//Default action when no command is provided
		Action: func(ctx context.Context, c *cli.Command) error {
			return cli.ShowAppHelp(c)
		},
	}
}

func GetTenantDefinition() *cli.Command {
	return &cli.Command{
		Name:    "tenant", // Better to use lowercase short names for CLI commands
		Aliases: []string{"currentTenant", "cT"},
		Usage:   "Current Tenant in Chroma DB",
		Action:  handleCurrentTenants,
	}
}

func ListDatabaseDefinition() *cli.Command {
	return &cli.Command{
		Name:    "databases",
		Aliases: []string{"ls-dbs", "dbs"},
		Usage:   "List Databases in current Tenant",
		Action:  handleListDatabases,
	}
}

func ListCollectionsInDatabaseDefinition() *cli.Command {
	return &cli.Command{
		Name:    "collections",
		Aliases: []string{"ls-colls", "colls"},
		Usage:   "List All the collections in database",
		Action:  handleListCollection,
	}
}

func GetOrCreateCollectionInDatabaseDefinition() *cli.Command {
	return &cli.Command{
		Name:    "createCollections",
		Aliases: []string{"mkdir-colls", "mkColl"},
		Usage:   "Create the collections in database",
		Action:  handleCreateCollection,
	}
}

func ListRecordsInCollection() *cli.Command {
	return &cli.Command{
		Name:      "records",
		Aliases:   []string{"ls-rs", "rs"},
		Usage:     "List All the records in database",
		ArgsUsage: collection_args_usage,
		Action:    handleListDocuments,
	}
}

func TestCommandDefinition() *cli.Command {
	return &cli.Command{
		Name:    "testConnection",
		Aliases: []string{"test", "t"},
		Usage:   "Test the connection to Chroma DB",
		Description: "Verifies connectivity to the DB instance and " +
			"ensures the service is responding correctly",
		Action: handleTestConnection,
		Flags:  []cli.Flag{timeoutFlag},
	}
}

func QueryDocumentCommandDefinition() *cli.Command {
	return &cli.Command{
		Name:      "query",
		Aliases:   []string{"q", "search"},
		Usage:     "Search for documents using one or more natural language queries",
		ArgsUsage: "<collection_name>",
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
			&cli.IntFlag{
				Name:    "n-results",
				Aliases: []string{"n"},
				Usage:   "Number of results to return per query",
				Value:   5,
			},
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
			&cli.StringSliceFlag{
				Name:     "doc",
				Aliases:  []string{"d"},
				Usage:    "The text document to add (can be repeated for batch add)",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:    "id",
				Aliases: []string{"i"},
				Usage:   "Optional: Custom IDs for the documents (must match the number of documents)",
			},
			modelOnnxFileFlag,
			tokenizerJsonFileFlag,
			onnxLibFlag,
		},
	}
}

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
)

const (
	collection_args_usage = "<collection_name>"
)
