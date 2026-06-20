package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestDumpSchema_CollectionCreateInput prints the JSON schema the Inspector
// actually receives for collection_create. Investigation only.
func TestDumpSchema_CollectionCreateInput(t *testing.T) {
	t1 := mcp.NewTool("collection_create",
		mcp.WithDescription("Create a new empty collection"),
		mcp.WithInputSchema[CollectionCreateInput](),
	)
	out, _ := json.MarshalIndent(t1, "", "  ")

	fmt.Println("=== INPUT SCHEMA ===")
	fmt.Println(string(out))
}
