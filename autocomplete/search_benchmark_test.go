package autocomplete

import "testing"

type searchBenchmarkCase struct {
	name       string
	query      string
	genres     []string
	titleTypes []string
	limit      int
	wantTotal  int
}

func buildBenchmarkIndex(b *testing.B, records int) Index {
	b.Helper()

	path, _ := writeSearchBenchmarkJSONL(b, records)
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
	for _, tt := range []searchBenchmarkCase{
		{name: "common-unigram", query: "e", limit: 10, wantTotal: 80_000},
		{name: "common-unigram-limit-one", query: "e", limit: 1, wantTotal: 80_000},
		{name: "common-unigram-limit-hundred", query: "e", limit: 100, wantTotal: 80_000},
		{name: "common-unigram-zero-limit", query: "e", limit: 0, wantTotal: 80_000},
		{name: "common-unigram-genre", query: "e", genres: []string{"drama"}, limit: 10, wantTotal: 40_000},
		{name: "common-unigram-genre-zero-limit", query: "e", genres: []string{"drama"}, limit: 0, wantTotal: 40_000},
		{name: "common-unigram-multi-genre", query: "e", genres: []string{"drama", "action"}, limit: 10, wantTotal: 80_000},
		{name: "common-unigram-title-type", query: "e", titleTypes: []string{"movie"}, limit: 10, wantTotal: 20_000},
		{name: "common-unigram-title-type-zero-limit", query: "e", titleTypes: []string{"movie"}, limit: 0, wantTotal: 20_000},
		{name: "common-unigram-multi-title-type", query: "e", titleTypes: []string{"movie", "short"}, limit: 10, wantTotal: 40_000},
		{name: "common-unigram-genre-and-title-type", query: "e", genres: []string{"drama"}, titleTypes: []string{"movie"}, limit: 10, wantTotal: 10_000},
		{name: "common-unigram-no-title-type-match", query: "e", titleTypes: []string{"video"}, limit: 10, wantTotal: 0},
		{name: "rare-unigram", query: "f", limit: 10, wantTotal: 20_000},
		{name: "unigram-intersection", query: "e a", limit: 10, wantTotal: 60_000},
		{name: "empty-unigram-intersection", query: "e q", limit: 10, wantTotal: 0},
		{name: "common-bigram", query: "ar", limit: 10, wantTotal: 60_000},
		{name: "common-trigram", query: "the", limit: 10, wantTotal: 20_000},
		{name: "common-trigram-genre", query: "the", genres: []string{"drama"}, limit: 10, wantTotal: 10_000},
		{name: "common-trigram-title-type", query: "the", titleTypes: []string{"movie"}, limit: 10, wantTotal: 5_000},
		{name: "common-short-multiword-case-insensitive", query: "AR CH", limit: 10, wantTotal: 40_000},
		{name: "mixed-short-long", query: "E EPISODE", limit: 10, wantTotal: 20_000},
		{name: "mixed-short-long-limit-one", query: "E EPISODE", limit: 1, wantTotal: 20_000},
		{name: "mixed-short-long-zero-limit", query: "E EPISODE", limit: 0, wantTotal: 20_000},
		{name: "mixed-singleton-miss", query: "Z EPISODE", limit: 10, wantTotal: 0},
		{name: "multiword-case-insensitive", query: "STAR WARS", limit: 10, wantTotal: 20_000},
		{name: "unigram-miss", query: "q", limit: 10, wantTotal: 0},
		{name: "long-word-miss", query: "qzxqzxqz", limit: 10, wantTotal: 0},
		{name: "long-query-zero-limit", query: "the", limit: 0, wantTotal: 20_000},
	} {
		b.Run(tt.name, func(b *testing.B) {
			query, err := ParseQuery(RawSearchParams{Term: tt.query, Limit: tt.limit, Genres: tt.genres, TitleTypes: tt.titleTypes})
			if err != nil {
				b.Fatal(err)
			}
			result := index.Search(query)
			assertBenchmarkSearchResult(b, tt, result)

			b.ReportAllocs()
			for b.Loop() {
				_ = index.Search(query)
			}
		})
	}
}

func BenchmarkEndToEndSearch100K(b *testing.B) {
	path, _ := writeBenchmarkJSONL(b, 100_000)
	query := mustParseSearchParams(b, "E EPISODE", 10)

	b.ReportAllocs()
	for b.Loop() {
		index, count, err := BuildIndexFromRecordStream(path)
		if err != nil {
			b.Fatal(err)
		}
		if count != 100_000 {
			b.Fatalf("processed %d records, want 100000", count)
		}
		result := index.Search(query)
		if result.Total != 20_000 || len(result.Movies) != 10 {
			b.Fatalf("Search(\"E EPISODE\") = %+v, want 20000 matches and 10 movies", result)
		}
	}
}

func assertBenchmarkSearchResult(b *testing.B, tt searchBenchmarkCase, result SearchResult) {
	b.Helper()

	if result.Total != tt.wantTotal {
		b.Fatalf("Search(%q) total = %d, want %d", tt.query, result.Total, tt.wantTotal)
	}
	if got, want := len(result.Movies), min(tt.limit, tt.wantTotal); got != want {
		b.Fatalf("Search(%q) returned %d movies, want %d", tt.query, got, want)
	}
}
