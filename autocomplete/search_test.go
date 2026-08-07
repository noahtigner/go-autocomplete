package autocomplete

import (
	"slices"
	"testing"

	models "github.com/noahtigner/go-autocomplete/models"
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

func movieIDs(movies []models.Movie) []int {
	ids := make([]int, len(movies))
	for i, movie := range movies {
		ids[i] = movie.ID
	}
	return ids
}
