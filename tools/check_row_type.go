//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/parquet-go/parquet-go"
)

func main() {
	filePath := "train-00000-of-00001.parquet"
	f, _ := os.Open(filePath)
	stat, _ := f.Stat()

	reader := parquet.NewFileReader(f, stat.Size())
	defer reader.Close()

	// Read a small sample
	rows := make([]any, 1)
	n, _ := reader.Read(rows)
	if n == 0 {
		fmt.Println("No rows read")
		return
	}

	row := rows[0]
	fmt.Printf("Row type: %T\n", row)
	fmt.Printf("Row value: %+v\n", row)
	fmt.Printf("Row reflect kind: %v\n", reflect.TypeOf(row).Kind())

	// Check if it's a struct
	rt := reflect.TypeOf(row)
	if rt.Kind() == reflect.Struct {
		fmt.Println("\nStruct fields:")
		for i := 0; i < rt.NumField(); i++ {
			field := rt.Field(i)
			value := reflect.ValueOf(row).Field(i)
			fmt.Printf("  %s (%s) = %v\n", field.Name, field.Type, value.Interface())
		}
	}
}
