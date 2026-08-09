package autocomplete

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
)

func TestSearch(t *testing.T) {
	index := buildFixtureIndex(t)

	tests := []struct {
		name      string
		query     string
		limit     int
		wantTotal int
		wantIDs   []int
	}{
		{name: "case-insensitive top K", query: "STAR", limit: 3, wantTotal: 12, wantIDs: []int{23, 25, 4}},
		{name: "multiword query", query: "wars star", limit: 10, wantTotal: 1, wantIDs: []int{23}},
		{name: "one-character query", query: "z", limit: 10, wantTotal: 1, wantIDs: []int{22}},
		{name: "two-character query", query: "IT", limit: 10, wantTotal: 3, wantIDs: []int{14, 13, 12}},
		{name: "three-character case-insensitive query", query: "THE", limit: 10, wantTotal: 5, wantIDs: []int{4, 11, 8, 21, 28}},
		{name: "multiword short case-insensitive query", query: "THE A", limit: 10, wantTotal: 3, wantIDs: []int{4, 8, 21}},
		{name: "equal-score tie-breaker", query: "alpha beta", limit: 10, wantTotal: 2, wantIDs: []int{18, 17}},
		{name: "result limit", query: "star", limit: 1, wantTotal: 12, wantIDs: []int{23}},
		{name: "zero limit", query: "star", limit: 0, wantTotal: 12, wantIDs: []int{}},
		{name: "no matches", query: "xylophone", limit: 10, wantTotal: 0, wantIDs: []int{}},
		{name: "both trigrams match but is not a substrings", query: "abcd", limit: 10, wantTotal: 0, wantIDs: []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := index.Search(tt.query, tt.limit)
			if err != nil {
				t.Fatal(err)
			}
			if got.Total != tt.wantTotal {
				t.Errorf("Search(%q, %d) total = %d, want %d", tt.query, tt.limit, got.Total, tt.wantTotal)
			}
			if ids := movieIDs(got.Movies); !slices.Equal(ids, tt.wantIDs) {
				t.Errorf("Search(%q, %d) IDs = %v, want %v", tt.query, tt.limit, ids, tt.wantIDs)
			}
		})
	}
}

func TestQueryWordsRequireVerification(t *testing.T) {
	tests := []struct {
		name       string
		queryWords []string
		want       bool
	}{
		{name: "unigram", queryWords: []string{"a"}, want: false},
		{name: "bigram", queryWords: []string{"it"}, want: false},
		{name: "trigram", queryWords: []string{"the"}, want: false},
		{name: "short multiword", queryWords: []string{"ar", "ch"}, want: false},
		{name: "four bytes", queryWords: []string{"abcd"}, want: true},
		{name: "mixed lengths", queryWords: []string{"the", "star"}, want: true},
		{name: "two-byte rune", queryWords: []string{"é"}, want: false},
		{name: "four-byte rune", queryWords: []string{"😀"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := queryWordsRequireVerification(tt.queryWords); got != tt.want {
				t.Errorf("queryWordsRequireVerification(%q) = %t, want %t", tt.queryWords, got, tt.want)
			}
		})
	}
}

func TestLongQueryCandidateRequiresVerification(t *testing.T) {
	index := buildFixtureIndex(t)
	candidateIds := retrieveSearchCandidateIds(index, []string{"abcd"})

	if !candidateIds.Contains(31) {
		t.Fatal("expected abc bcd to be a candidate for abcd")
	}
}

func TestSearchOneByteQueryRegressions(t *testing.T) {
	index := buildOneByteRegressionIndex(t)

	tests := []struct {
		name      string
		query     string
		limit     int
		wantTotal int
		wantIDs   []int
	}{
		{name: "case insensitive", query: "A", limit: 10, wantTotal: 5, wantIDs: []int{39_063_631, 26_700_024, 103, 101, 2}},
		{name: "whitespace normalized", query: " \tA\n", limit: 10, wantTotal: 5, wantIDs: []int{39_063_631, 26_700_024, 103, 101, 2}},
		{name: "result limit", query: "a", limit: 1, wantTotal: 5, wantIDs: []int{39_063_631}},
		{name: "zero limit", query: "a", limit: 0, wantTotal: 5, wantIDs: []int{}},
		{name: "miss", query: "q", limit: 10, wantTotal: 0, wantIDs: []int{}},
		{name: "repeated character", query: "a a", limit: 10, wantTotal: 5, wantIDs: []int{39_063_631, 26_700_024, 103, 101, 2}},
		{name: "character intersection", query: "a b", limit: 10, wantTotal: 2, wantIDs: []int{39_063_631, 26_700_024}},
		{name: "digit", query: "7", limit: 10, wantTotal: 1, wantIDs: []int{400}},
		{name: "punctuation", query: "!", limit: 10, wantTotal: 1, wantIDs: []int{400}},
		{name: "two byte rune", query: "É", limit: 10, wantTotal: 1, wantIDs: []int{500}},
		{name: "character and bigram", query: "a it", limit: 10, wantTotal: 1, wantIDs: []int{101}},
		{name: "character and trigram", query: "a the", limit: 10, wantTotal: 1, wantIDs: []int{103}},
		{name: "character and long word", query: "z orbit", limit: 10, wantTotal: 0, wantIDs: []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := index.Search(tt.query, tt.limit)
			if err != nil {
				t.Fatal(err)
			}
			if got.Total != tt.wantTotal {
				t.Errorf("Search(%q, %d) total = %d, want %d", tt.query, tt.limit, got.Total, tt.wantTotal)
			}
			if ids := movieIDs(got.Movies); !slices.Equal(ids, tt.wantIDs) {
				t.Errorf("Search(%q, %d) IDs = %v, want %v", tt.query, tt.limit, ids, tt.wantIDs)
			}
		})
	}
}

func TestSearchInvalidInput(t *testing.T) {
	index := buildFixtureIndex(t)

	tests := []struct {
		name  string
		query string
		limit int
	}{
		{name: "empty query", query: "", limit: 10},
		{name: "whitespace-only query", query: " \t\n", limit: 10},
		{name: "negative limit", query: "star", limit: -1},
		{name: "excessively large limit", query: "star", limit: 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := index.Search(tt.query, tt.limit)
			if err == nil {
				t.Fatalf("Search(%q, %d) returned nil error", tt.query, tt.limit)
			}
			if got.Total != 0 || got.Movies != nil {
				t.Errorf("Search(%q, %d) result = %+v, want a zero-value result", tt.query, tt.limit, got)
			}
		})
	}
}

func buildFixtureIndex(t *testing.T) Index {
	t.Helper()

	index, _, err := BuildIndexFromRecordStream("../testdata/etl/movies.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	return index
}

func buildOneByteRegressionIndex(t *testing.T) Index {
	t.Helper()

	movies := []movies.Movie{
		{ID: 39_063_631, PrimaryTitle: "A B"},
		{ID: 2, PrimaryTitle: "A"},
		{ID: 2_670_090, PrimaryTitle: "B"},
		{ID: 26_700_024, PrimaryTitle: "A B"},
		{ID: 100, PrimaryTitle: "It"},
		{ID: 101, PrimaryTitle: "A It"},
		{ID: 102, PrimaryTitle: "The"},
		{ID: 103, PrimaryTitle: "A The"},
		{ID: 300, PrimaryTitle: "Orbit"},
		{ID: 400, PrimaryTitle: "7!"},
		{ID: 500, PrimaryTitle: "É"},
	}

	path := filepath.Join(t.TempDir(), "movies.jsonl")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	encoder := json.NewEncoder(file)
	for _, movie := range movies {
		if err := encoder.Encode(movie); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	index, count, err := BuildIndexFromRecordStream(path)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(movies) {
		t.Fatalf("processed %d records, want %d", count, len(movies))
	}
	return index
}

func movieIDs(movies []movies.Movie) []int {
	ids := make([]int, len(movies))
	for i, movie := range movies {
		ids[i] = movie.ID
	}
	return ids
}
