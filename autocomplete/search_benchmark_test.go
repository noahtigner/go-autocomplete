package autocomplete

import (
	"slices"
	"testing"
)

type searchBenchmarkCase struct {
	name      string
	query     string
	limit     int
	wantTotal int
}

func buildBenchmarkIndex(b *testing.B, records int) Index {
	b.Helper()

	path, _ := writeBenchmarkJSONL(b, records)
	return buildBenchmarkIndexFromJSONL(b, path, records)
}

func buildDiverseBenchmarkIndex(b *testing.B, records int) Index {
	b.Helper()

	path, _ := writeDiverseBenchmarkJSONL(b, records)
	return buildBenchmarkIndexFromJSONL(b, path, records)
}

func buildSlotBoundaryBenchmarkIndex(b *testing.B) Index {
	b.Helper()

	path, _ := writeSlotBoundaryBenchmarkJSONL(b)
	return buildBenchmarkIndexFromJSONL(b, path, 130)
}

func buildBenchmarkIndexFromJSONL(b *testing.B, path string, records int) Index {
	b.Helper()

	index, count, err := BuildIndexFromRecordStream(path)
	if err != nil {
		b.Fatal(err)
	}
	if count != records {
		b.Fatalf("processed %d records, want %d", count, records)
	}

	return index
}

func BenchmarkSearchIndex100K(b *testing.B) {
	index := buildBenchmarkIndex(b, 100_000)
	runSearchBenchmarks(b, index, []searchBenchmarkCase{
		{name: "common-unigram", query: "e", limit: 10, wantTotal: 80_000},
		{name: "common-unigram-limit-one", query: "e", limit: 1, wantTotal: 80_000},
		{name: "common-unigram-limit-hundred", query: "e", limit: 100, wantTotal: 80_000},
		{name: "common-unigram-zero-limit", query: "e", limit: 0, wantTotal: 80_000},
		{name: "rare-unigram", query: "f", limit: 10, wantTotal: 20_000},
		{name: "unigram-intersection", query: "e a", limit: 10, wantTotal: 60_000},
		{name: "empty-unigram-intersection", query: "e q", limit: 10, wantTotal: 0},
		{name: "common-bigram", query: "ar", limit: 10, wantTotal: 60_000},
		{name: "common-trigram", query: "the", limit: 10, wantTotal: 20_000},
		{name: "common-short-multiword-case-insensitive", query: "AR CH", limit: 10, wantTotal: 40_000},
		{name: "mixed-short-long", query: "E EPISODE", limit: 10, wantTotal: 20_000},
		{name: "mixed-short-long-limit-one", query: "E EPISODE", limit: 1, wantTotal: 20_000},
		{name: "mixed-short-long-zero-limit", query: "E EPISODE", limit: 0, wantTotal: 20_000},
		{name: "mixed-singleton-miss", query: "Z EPISODE", limit: 10, wantTotal: 0},
		{name: "multiword-case-insensitive", query: "STAR WARS", limit: 10, wantTotal: 20_000},
		{name: "unigram-miss", query: "q", limit: 10, wantTotal: 0},
		{name: "long-word-miss", query: "qzxqzxqz", limit: 10, wantTotal: 0},
		{name: "long-query-zero-limit", query: "the", limit: 0, wantTotal: 20_000},
	})
}

func BenchmarkSearchDiverseIndex100K(b *testing.B) {
	index := buildDiverseBenchmarkIndex(b, 100_000)
	runSearchBenchmarks(b, index, []searchBenchmarkCase{
		{name: "unicode", query: "éclair", limit: 10, wantTotal: 10_000},
		{name: "mixed-unicode-ASCII", query: "éclair a", limit: 10, wantTotal: 10_000},
		{name: "rare-long-success", query: "quasar", limit: 10, wantTotal: 10_000},
		{name: "present-singleton-rejection", query: "z quasar", limit: 10, wantTotal: 0},
		{name: "long-query-false-positive", query: "abcd", limit: 10, wantTotal: 0},
		{name: "punctuation-bigram", query: "7!", limit: 10, wantTotal: 10_000},
	})
}

func BenchmarkSearchSlotBoundary(b *testing.B) {
	index := buildSlotBoundaryBenchmarkIndex(b)
	runSearchBenchmarks(b, index, []searchBenchmarkCase{
		{name: "slots-63-and-64", query: "z", limit: 10, wantTotal: 2},
		{name: "slots-63-and-64-limit-one", query: "z", limit: 1, wantTotal: 2},
	})
}

func runSearchBenchmarks(b *testing.B, index Index, tests []searchBenchmarkCase) {
	b.Helper()

	implementations := []struct {
		name   string
		search func(string, int) (SearchResult, error)
	}{
		{name: "old", search: index.oldSearch},
		{name: "new", search: index.newSearch},
	}

	for _, tt := range tests {
		oldResult, oldErr := index.oldSearch(tt.query, tt.limit)
		newResult, newErr := index.newSearch(tt.query, tt.limit)
		assertBenchmarkSearchParity(b, tt, oldResult, oldErr, newResult, newErr)

		for _, implementation := range implementations {
			b.Run(tt.name+"/"+implementation.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, _ = implementation.search(tt.query, tt.limit)
				}
			})
		}
	}
}

func assertBenchmarkSearchParity(b *testing.B, tt searchBenchmarkCase, oldResult SearchResult, oldErr error, newResult SearchResult, newErr error) {
	b.Helper()

	if errorMessage(oldErr) != errorMessage(newErr) {
		b.Fatalf("Search(%q) old error = %q, new error = %q", tt.query, errorMessage(oldErr), errorMessage(newErr))
	}
	if newErr != nil {
		b.Fatal(newErr)
	}
	if oldResult.Total != tt.wantTotal || newResult.Total != tt.wantTotal {
		b.Fatalf("Search(%q) totals = old %d, new %d, want %d", tt.query, oldResult.Total, newResult.Total, tt.wantTotal)
	}
	if !slices.Equal(movieIDs(oldResult.Movies), movieIDs(newResult.Movies)) {
		b.Fatalf("Search(%q) old IDs = %v, new IDs = %v", tt.query, movieIDs(oldResult.Movies), movieIDs(newResult.Movies))
	}
	if got, want := len(newResult.Movies), min(tt.limit, tt.wantTotal); got != want {
		b.Fatalf("Search(%q) returned %d movies, want %d", tt.query, got, want)
	}
}
