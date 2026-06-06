package main

import (
	"log"
	"os"

	"github.com/parquet-go/parquet-go"
)

type ParquetRow struct {
	Question       string `parquet:"question"`
	ConversationID string `parquet:"conversation_id"`
	Domain         string `parquet:"domain"`
}

func main() {
	// Create 40 rows of sample data
	var rows []ParquetRow
	for i := 0; i < 40; i++ {
		rows = append(rows, ParquetRow{
			Question:       "Sample question " + string(rune('A'+i%26)),
			ConversationID: "conv-" + string(rune('a'+i%26)),
			Domain:         "test-domain",
		})
	}

	// Create the Parquet file
	f, err := os.Create("train-00000-of-00001.parquet")
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	writer := parquet.NewGenericWriter[f](f)
	if err != nil {
		log.Fatalf("Failed to create writer: %v", err)
	}

	_, err = writer.Write(rows)
	if err != nil {
		log.Fatalf("Failed to write rows: %v", err)
	}

	err = writer.Close()
	if err != nil {
		log.Fatalf("Failed to close writer: %v", err)
	}

	log.Println("Successfully generated Parquet file with 40 rows")
}