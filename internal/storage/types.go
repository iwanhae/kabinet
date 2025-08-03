package storage

import "time"

// ParquetFileInfo represents information about a parquet file
type ParquetFileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// RangeQueryResult contains metadata about a range query execution
type RangeQueryResult struct {
	Duration time.Duration
	Files    []ParquetFileInfo
}
