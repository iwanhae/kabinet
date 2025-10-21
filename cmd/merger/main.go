package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/marcboeker/go-duckdb/v2"
)

func main() {
	// Define flags
	outputFile := flag.String("o", "", "Output parquet file path (required)")
	flag.Parse()

	// Validate output flag
	if *outputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: output file (-o) is required")
		flag.Usage()
		os.Exit(1)
	}

	// Get input files from remaining arguments
	inputFiles := flag.Args()
	if len(inputFiles) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one input parquet file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Validate input files exist
	for _, file := range inputFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			log.Fatalf("Error: input file does not exist: %s", file)
		}
	}

	// Merge the parquet files
	if err := mergeParquetFiles(inputFiles, *outputFile); err != nil {
		log.Fatalf("Error merging parquet files: %v", err)
	}

	fmt.Printf("Successfully merged %d files into %s\n", len(inputFiles), *outputFile)
}

func mergeParquetFiles(inputFiles []string, outputFile string) error {
	// Open an in-memory DuckDB database
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Build the array of input files for the query
	// Format: ['file1.parquet', 'file2.parquet', ...]
	quotedFiles := make([]string, len(inputFiles))
	for i, file := range inputFiles {
		// Escape single quotes in file paths and wrap in quotes
		escaped := strings.ReplaceAll(file, "'", "''")
		quotedFiles[i] = fmt.Sprintf("'%s'", escaped)
	}
	filesArray := fmt.Sprintf("[%s]", strings.Join(quotedFiles, ", "))

	// Build the COPY query
	query := fmt.Sprintf(
		"COPY (FROM read_parquet(%s)) TO '%s' (FORMAT parquet, COMPRESSION zstd);",
		filesArray,
		strings.ReplaceAll(outputFile, "'", "''"),
	)

	// Execute the merge query
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to execute merge query: %w", err)
	}

	return nil
}
