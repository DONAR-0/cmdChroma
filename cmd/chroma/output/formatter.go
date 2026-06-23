package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/DONAR-0/cmdChroma/internal/client"
	cli "github.com/urfave/cli/v3"
)

// Formatter formats ChromaDB query and document responses for display.
type Formatter struct {
	writer io.Writer
	mode   Mode
}

// NewFormatter creates a Formatter that writes to w in the given mode.
func NewFormatter(w io.Writer, mode Mode) *Formatter {
	return &Formatter{writer: w, mode: mode}
}

// FormatQueryResponse prints query results to the output.
func (f *Formatter) FormatQueryResponse(collection string, queries []string, resp *client.QueryResponse) {
	switch f.mode {
	case ModeJSON:
		f.printJSON(queryResponseAsMap(collection, queries, resp))
	default:
		f.printQueryResponseText(collection, queries, resp)
	}
}

func (f *Formatter) printQueryResponseText(collection string, queries []string, resp *client.QueryResponse) {
	_, _ = fmt.Fprintf(f.writer, "\n🔍 Search results for collection '%s':\n\n", collection)

	for i, q := range queries {
		_, _ = fmt.Fprintf(f.writer, "Query %d: %s\n", i+1, q)
		_, _ = fmt.Fprintln(f.writer, strings.Repeat("-", 60))

		for j := 0; j < len(resp.IDs[i]); j++ {
			_, _ = fmt.Fprintf(f.writer, "  [%d] Distance: %.4f\n", j+1, resp.Distances[i][j])
			_, _ = fmt.Fprintf(f.writer, "      ID: %s\n", resp.IDs[i][j])

			if len(resp.Documents[i]) > j {
				content := resp.Documents[i][j]
				if len(content) > 150 {
					content = content[:150] + "..."
				}

				_, _ = fmt.Fprintf(f.writer, "      Content: %s\n\n", content)
			}
		}

		if i < len(queries)-1 {
			_, _ = fmt.Fprintln(f.writer, strings.Repeat("=", 60)+"\n")
		}
	}
}

// FormatDocumentList prints a list of documents to the output.
func (f *Formatter) FormatDocumentList(collection string, docs *client.GetRecordsResponse) {
	switch f.mode {
	case ModeJSON:
		f.printJSON(documentListAsMap(collection, docs))
	default:
		f.printDocumentListText(collection, docs)
	}
}

func (f *Formatter) printDocumentListText(collection string, docs *client.GetRecordsResponse) {
	_, _ = fmt.Fprintf(f.writer, "\n📄 Documents in collection '%s' (%d total):\n\n", collection, len(docs.IDs))

	for i := 0; i < len(docs.IDs); i++ {
		_, _ = fmt.Fprintf(f.writer, "ID:       %s\n", docs.IDs[i])

		if len(docs.Documents) > i {
			content := docs.Documents[i]
			if len(content) > 100 {
				content = content[:100] + "..."
			}

			_, _ = fmt.Fprintf(f.writer, "Content:  %s\n", content)
		}

		if len(docs.Metadatas) > i && docs.Metadatas[i] != nil {
			_, _ = fmt.Fprintf(f.writer, "Metadata: %v\n", docs.Metadatas[i])
		}

		_, _ = fmt.Fprintln(f.writer, strings.Repeat("-", 40))
	}
}

func (f *Formatter) printJSON(data any) {
	enc := json.NewEncoder(f.writer)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

func queryResponseAsMap(collection string, queries []string, resp *client.QueryResponse) map[string]any {
	return map[string]any{
		"collection": collection,
		"queries":    queries,
		"results":    resp,
	}
}

func documentListAsMap(collection string, docs *client.GetRecordsResponse) map[string]any {
	return map[string]any{
		"collection": collection,
		"count":      len(docs.IDs),
		"documents":  docs,
	}
}

func ModeFromCLI(c any) Mode {
	if cmd, ok := c.(*cli.Command); ok && cmd.Bool("json") {
		return ModeJSON
	}

	return ModeHuman
}
