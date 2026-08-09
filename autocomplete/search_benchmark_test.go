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
		{name: "common-unigram-limit-one", query: "e", limit: 1, wantTotal: 80_000},
		{name: "common-unigram-limit-hundred", query: "e", limit: 100, wantTotal: 80_000},
		{name: "common-unigram-zero-limit", query: "e", limit: 0, wantTotal: 80_000},
		{name: "rare-unigram", query: "f", limit: 10, wantTotal: 20_000},
		{name: "unigram-intersection", query: "e a", limit: 10, wantTotal: 60_000},
		{name: "empty-unigram-intersection", query: "e q", limit: 10, wantTotal: 0},
		{name: "common-bigram", query: "ar", limit: 10, wantTotal: 60_000},
		{name: "common-trigram", query: "the", limit: 10, wantTotal: 20_000},
		{name: "common-short-multiword-case-insensitive", query: "AR CH", limit: 10, wantTotal: 40_000},
		{name: "mixed-short-long-case-insensitive", query: "E EPISODE", limit: 10, wantTotal: 20_000},
		{name: "mixed-short-long-rejection", query: "Z EPISODE", limit: 10, wantTotal: 0},
		{name: "multiword-case-insensitive", query: "STAR WARS", limit: 10, wantTotal: 20_000},
		{name: "unigram-miss", query: "q", limit: 10, wantTotal: 0},
		{name: "long-word-miss", query: "qzxqzxqz", limit: 10, wantTotal: 0},
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

func BenchmarkRetrieveSearchCandidateIds100K(b *testing.B) {
	index := buildBenchmarkIndex(b, 100_000)

	tests := []struct {
		name       string
		queryWords []string
		wantTotal  int
	}{
		{name: "common-unigram", queryWords: []string{"e"}, wantTotal: 80_000},
		{name: "unigram-intersection", queryWords: []string{"e", "a"}, wantTotal: 60_000},
		{name: "empty-unigram-intersection", queryWords: []string{"e", "q"}, wantTotal: 0},
		{name: "mixed-short-long", queryWords: []string{"e", "episode"}, wantTotal: 20_000},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			if got := len(retrieveSearchCandidateIds(index, tt.queryWords)); got != tt.wantTotal {
				b.Fatalf("candidate count for %q = %d, want %d", tt.queryWords, got, tt.wantTotal)
			}

			b.ReportAllocs()

			var candidateCount int
			for b.Loop() {
				candidateCount = len(retrieveSearchCandidateIds(index, tt.queryWords))
			}
			if candidateCount != tt.wantTotal {
				b.Fatalf("candidate count for %q = %d, want %d", tt.queryWords, candidateCount, tt.wantTotal)
			}
		})
	}
}
