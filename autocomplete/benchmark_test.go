package autocomplete

import (
	"errors"
	"os"
	"testing"
)

func benchmarkDataPath(b *testing.B) string {
	b.Helper()

	path := os.Getenv("GO_AUTOCOMPLETE_BENCH_DATA")
	if path == "" {
		path = "../data/benchmark-movies.jsonl"
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		b.Skipf("benchmark data is missing; generate it with: go run ./cmd/benchmark-sample -count 100000")
	} else if err != nil {
		b.Fatal(err)
	}

	return path
}

func buildBenchmarkIndex(b *testing.B) Index {
	b.Helper()

	path := benchmarkDataPath(b)
	index, _, err := BuildIndexFromRecordStream(path)
	if err != nil {
		b.Fatal(err)
	}
	return index
}

func BenchmarkBuildIndexFromRecordStream(b *testing.B) {
	path := benchmarkDataPath(b)

	b.ReportAllocs()
	for b.Loop() {
		if _, _, err := BuildIndexFromRecordStream(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	b.StopTimer()
	index := buildBenchmarkIndex(b)

	tests := []struct {
		name  string
		query string
		limit int
	}{
		{name: "narrow", query: "star wars", limit: 10},
		{name: "broad", query: "star", limit: 10},
		{name: "one_character", query: "a", limit: 10},
		{name: "two_character", query: "it", limit: 10},
		{name: "no_match", query: "xylophone", limit: 10},
		{name: "zero_limit", query: "star", limit: 0},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				if _, err := index.Search(tt.query, tt.limit); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
