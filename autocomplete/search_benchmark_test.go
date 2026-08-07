package autocomplete

import "testing"

func buildBenchmarkIndex(b *testing.B, records int) Index {
	b.Helper()

	path, _ := writeBenchmarkJSONL(b, records)

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

	tests := []struct {
		name      string
		query     string
		limit     int
		wantTotal int
	}{
		{name: "common-unigram", query: "e", limit: 10, wantTotal: 80_000},
		{name: "common-bigram", query: "ar", limit: 10, wantTotal: 60_000},
		{name: "common-trigram", query: "the", limit: 10, wantTotal: 20_000},
		{name: "multiword-case-insensitive", query: "STAR WARS", limit: 10, wantTotal: 20_000},
		{name: "miss", query: "qzxqzxqz", limit: 10, wantTotal: 0},
		{name: "zero-limit", query: "the", limit: 0, wantTotal: 20_000},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Validate the generated workload before timing it.
			result, err := index.Search(tt.query, tt.limit)
			if err != nil {
				b.Fatal(err)
			}
			if result.Total != tt.wantTotal {
				b.Fatalf("Search(%q) total = %d, want %d",
					tt.query, result.Total, tt.wantTotal)
			}
			if got, want := len(result.Movies), min(tt.limit, tt.wantTotal); got != want {
				b.Fatalf("Search(%q) returned %d movies, want %d",
					tt.query, got, want)
			}

			b.ReportAllocs()

			// b.Loop starts timing here and stops it after the loop.
			for b.Loop() {
				_, _ = index.Search(tt.query, tt.limit)
			}
		})
	}
}
