package main

import (
	"log"
	"os"

	"github.com/parquet-go/parquet-go"
)

// TestData matches the structure expected by the Parquet import test
type TestData struct {
	Question       string `parquet:"question"`
	ConversationID string `parquet:"conversation_id"`
	Domain         string `parquet:"domain"`
}

func main() {
	// Create test data matching what's expected by the test
	var rows []TestData

	// Create 40 rows to match the test expectation (verify record count matches 40)
	for i := 0; i < 40; i++ {
		rows = append(rows, TestData{
			Question:       "What is the meaning of life?",
			ConversationID: "conv-" + string(rune('a'+i%26)),
			Domain:         "test-domain",
		})
	}

	// Create the Parquet file in the current directory where the test runs
	f, err := os.Create("train-00000-of-00001.parquet")
	if err != nil {
		log.Fatalf("Failed to create Parquet file: %v", err)
	}
	defer f.Close()

	writer := parquet.NewGenericWriter[TestData](f)
	if err != nil {
		log.Fatalf("Failed to create Parquet writer: %v", err)
	}

	_, err = writer.Write(rows)
	if err != nil {
		log.Fatalf("Failed to write Parquet data: %v", err)
	}

	err = writer.Close()
	if err != nil {
		log.Fatalf("Failed to close Parquet writer: %v", err)
	}

	log.Printf("Successfully generated Parquet file with %d rows", len(rows))
}