package output

import (
	"strings"
	"testing"

	"github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/urfave/cli/v3"
)

func TestFormatter_FormatQueryResponse_HumanMode(t *testing.T) {
	var buf strings.Builder

	formatter := NewFormatter(&buf, ModeHuman)

	resp := &client.QueryResponse{
		IDs:       [][]string{{"id1", "id2"}},
		Documents: [][]string{{"doc one", "doc two"}},
		Distances: [][]float32{{0.1, 0.2}},
	}

	formatter.FormatQueryResponse("my_collection", []string{"query1"}, resp)

	output := buf.String()
	if !strings.Contains(output, "Search results") {
		t.Errorf("expected 'Search results' in output, got: %s", output)
	}

	if !strings.Contains(output, "my_collection") {
		t.Errorf("expected collection name in output, got: %s", output)
	}

	if !strings.Contains(output, "Distance: 0.1000") {
		t.Errorf("expected formatted distance in output, got: %s", output)
	}

	if !strings.Contains(output, "doc one") {
		t.Errorf("expected document content in output, got: %s", output)
	}
}

func TestFormatter_FormatQueryResponse_JSONMode(t *testing.T) {
	var buf strings.Builder

	formatter := NewFormatter(&buf, ModeJSON)

	resp := &client.QueryResponse{
		IDs:       [][]string{{"id1"}},
		Documents: [][]string{{"doc one"}},
		Distances: [][]float32{{0.1}},
	}

	formatter.FormatQueryResponse("my_collection", []string{"q"}, resp)

	output := buf.String()
	if !strings.Contains(output, `"collection":`) || !strings.Contains(output, `"my_collection"`) {
		t.Errorf("expected JSON with collection in output, got: %s", output)
	}

	if !strings.Contains(output, `"ids"`) {
		t.Errorf("expected JSON with ids in output, got: %s", output)
	}
}

func TestFormatter_FormatQueryResponse_MultipleQueries(t *testing.T) {
	var buf strings.Builder

	formatter := NewFormatter(&buf, ModeHuman)

	resp := &client.QueryResponse{
		IDs:       [][]string{{"q1-doc1"}, {"q2-doc1"}},
		Documents: [][]string{{"answer1"}, {"answer2"}},
		Distances: [][]float32{{0.1}, {0.2}},
	}

	formatter.FormatQueryResponse("coll", []string{"query1", "query2"}, resp)

	output := buf.String()
	if !strings.Contains(output, "Query 1: query1") {
		t.Errorf("expected 'Query 1: query1' in output, got: %s", output)
	}

	if !strings.Contains(output, "Query 2: query2") {
		t.Errorf("expected 'Query 2: query2' in output, got: %s", output)
	}

	if !strings.Contains(output, "answer1") {
		t.Errorf("expected 'answer1' in output, got: %s", output)
	}

	if !strings.Contains(output, "answer2") {
		t.Errorf("expected 'answer2' in output, got: %s", output)
	}
}

func TestFormatter_FormatDocumentList_HumanMode(t *testing.T) {
	var buf strings.Builder

	formatter := NewFormatter(&buf, ModeHuman)

	docs := &client.GetRecordsResponse{
		IDs:       []string{"id1", "id2"},
		Documents: []string{"content one", "content two"},
		Metadatas: []map[string]any{{"key": "value1"}, {"key": "value2"}},
	}

	formatter.FormatDocumentList("test_collection", docs)

	output := buf.String()
	if !strings.Contains(output, "Documents in collection 'test_collection'") {
		t.Errorf("expected header in output, got: %s", output)
	}

	if !strings.Contains(output, "id1") {
		t.Errorf("expected 'id1' in output, got: %s", output)
	}

	if !strings.Contains(output, "content one") {
		t.Errorf("expected 'content one' in output, got: %s", output)
	}

	if !strings.Contains(output, "Metadata: map") {
		t.Errorf("expected metadata in output, got: %s", output)
	}
}

func TestFormatter_FormatDocumentList_JSONMode(t *testing.T) {
	var buf strings.Builder

	formatter := NewFormatter(&buf, ModeJSON)

	docs := &client.GetRecordsResponse{
		IDs:       []string{"id1"},
		Documents: []string{"doc1"},
		Metadatas: []map[string]any{{"key": "value"}},
	}

	formatter.FormatDocumentList("coll", docs)

	output := buf.String()
	if !strings.Contains(output, `"collection":`) || !strings.Contains(output, `"coll"`) {
		t.Errorf("expected collection in JSON output, got: %s", output)
	}

	if !strings.Contains(output, `"count":`) {
		t.Errorf("expected count in JSON output, got: %s", output)
	}
}

func TestFormatter_FormatQueryResponse_TruncatesLongContent(t *testing.T) {
	var buf strings.Builder

	formatter := NewFormatter(&buf, ModeHuman)

	longContent := strings.Repeat("a", 200)
	resp := &client.QueryResponse{
		IDs:       [][]string{{"id1"}},
		Documents: [][]string{{longContent}},
		Distances: [][]float32{{0.1}},
	}

	formatter.FormatQueryResponse("coll", []string{"q"}, resp)

	output := buf.String()
	if strings.Contains(output, longContent) {
		t.Errorf("expected long content to be truncated, but it wasn't")
	}

	if !strings.Contains(output, "...") {
		t.Errorf("expected truncation marker '...' in output, got: %s", output)
	}
}

func TestModeFromCLI(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{&cli.BoolFlag{Name: "json"}},
	}
	if ModeFromCLI(cmd) != ModeHuman {
		t.Errorf("ModeFromCLI() without json flag = %v, want ModeHuman", ModeFromCLI(cmd))
	}

	if err := cmd.Set("json", "true"); err != nil {
		t.Fatal(err)
	}

	if ModeFromCLI(cmd) != ModeJSON {
		t.Errorf("ModeFromCLI() with json flag = %v, want ModeJSON", ModeFromCLI(cmd))
	}

	// Test with non-command input
	if ModeFromCLI(nil) != ModeHuman {
		t.Errorf("ModeFromCLI(nil) = %v, want ModeHuman", ModeFromCLI(nil))
	}
}
