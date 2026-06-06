package main

import (
	"fmt"
	"log"
	"os"

	"github.com/parquet-go/parquet-go"
)

// TestData matches the structure of the original Parquet file
type TestData struct {
	ConversationID string `parquet:"conversation_id"`
	Domain         string `parquet:"domain"`
	SubDomain      string `parquet:"subDomain"`
	AuthorID       string `parquet:"author_id"`
	Question       string `parquet:"question"`
	Answer         string `parquet:"answer"`
	Format         string `parquet:"format"`
	Images         string `parquet:"images"`
}

func main() {
	// Create test data matching the original Parquet file structure (40 rows)
	var rows []TestData

	for i := 0; i < 40; i++ {
		rows = append(rows, TestData{
			ConversationID: fmt.Sprintf("conv-%03d", i),
			Domain:         "test-domain",
			SubDomain:      "test-subdomain",
			AuthorID:       "user_001",
			Question:       "What is the meaning of life?",
			Answer:         "The answer to everything is 42",
			Format:         "QA",
			Images:         `[{"bytes":"","path":""}]`,
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
