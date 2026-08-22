package autocomplete

import (
	"slices"
	"sort"
	"strconv"
	"strings"
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
		{name: "exact title boost", query: "alpha beta", limit: 10, wantTotal: 2, wantIDs: []int{17, 18}},
		{name: "result limit", query: "star", limit: 1, wantTotal: 12, wantIDs: []int{23}},
		{name: "zero limit", query: "star", limit: 0, wantTotal: 12, wantIDs: []int{}},
		{name: "no matches", query: "xylophone", limit: 10, wantTotal: 0, wantIDs: []int{}},
		{name: "both trigrams match but is not a substrings", query: "abcd", limit: 10, wantTotal: 0, wantIDs: []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := index.Search(mustParseSearchParams(t, tt.query, tt.limit))
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
		{name: "case insensitive", query: "A", limit: 10, wantTotal: 5, wantIDs: []int{2, 39_063_631, 26_700_024, 103, 101}},
		{name: "whitespace normalized", query: " \tA\n", limit: 10, wantTotal: 5, wantIDs: []int{2, 39_063_631, 26_700_024, 103, 101}},
		{name: "result limit", query: "a", limit: 1, wantTotal: 5, wantIDs: []int{2}},
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
			got := index.Search(mustParseSearchParams(t, tt.query, tt.limit))
			if got.Total != tt.wantTotal {
				t.Errorf("Search(%q, %d) total = %d, want %d", tt.query, tt.limit, got.Total, tt.wantTotal)
			}
			if ids := movieIDs(got.Movies); !slices.Equal(ids, tt.wantIDs) {
				t.Errorf("Search(%q, %d) IDs = %v, want %v", tt.query, tt.limit, ids, tt.wantIDs)
			}
		})
	}
}

func TestSearchBitmapSlotBoundary(t *testing.T) {
	index := buildSlotBoundaryIndex(t)
	result := index.Search(mustParseSearchParams(t, "z", 10))
	assertSearchResult(t, result, 2, []int{1_000_064, 1_000_063})
}

func TestSearchMixedUnicodeAndASCII(t *testing.T) {
	index := buildUnicodeRegressionIndex(t)

	for _, query := range []string{"é a", "écho a"} {
		t.Run(query, func(t *testing.T) {
			result := index.Search(mustParseSearchParams(t, query, 10))
			assertSearchResult(t, result, 1, []int{901})
		})
	}
}

func TestSearchMatchesBruteForce(t *testing.T) {
	records := make([]movies.Movie, 130)
	titles := []string{
		"Alpha Beta",
		"Gamma Delta",
		"Orbit Éclair",
		"7! Signal",
		"ABC X BCD",
		"Zulu Archive",
	}
	for i := range records {
		rating := 4.0 + float64(i%50)/10
		title := titles[i%len(titles)]
		if i%2 == 0 {
			title = strings.ToUpper(title)
		}
		records[i] = movies.Movie{
			ID:            20_000_000 + i*10_003,
			PrimaryTitle:  title,
			AverageRating: &rating,
			NumVotes:      100 + i,
		}
	}

	index := buildIndexFromMovies(t, records)
	queries := []string{
		"a", "a a", "a b", "q", "a q", "alpha", "abcd", "é", "é a", "éclair a", "7", "!", "orbit é", "z archive", "missing", "\tALPHA  BETA\n",
	}
	limits := []int{0, 1, 5, 100}

	for _, query := range queries {
		for _, limit := range limits {
			t.Run(query+"/"+strconv.Itoa(limit), func(t *testing.T) {
				want := bruteForceSearch(records, query, limit)
				result := index.Search(mustParseSearchParams(t, query, limit))
				assertEquivalentSearchResult(t, "Search", result, want)
			})
		}
	}
}

func assertSearchResult(t *testing.T, result SearchResult, wantTotal int, wantIDs []int) {
	t.Helper()

	if result.Total != wantTotal {
		t.Errorf("total = %d, want %d", result.Total, wantTotal)
	}
	if got := movieIDs(result.Movies); !slices.Equal(got, wantIDs) {
		t.Errorf("IDs = %v, want %v", got, wantIDs)
	}
}

func assertEquivalentSearchResult(t *testing.T, implementation string, got, want SearchResult) {
	t.Helper()

	if got.Total != want.Total {
		t.Errorf("%s total = %d, want %d", implementation, got.Total, want.Total)
	}
	if gotIDs, wantIDs := movieIDs(got.Movies), movieIDs(want.Movies); !slices.Equal(gotIDs, wantIDs) {
		t.Errorf("%s IDs = %v, want %v", implementation, gotIDs, wantIDs)
	}
}

func mustParseSearchParams(t testing.TB, term string, limit int) SearchParams {
	t.Helper()

	params, err := ParseQuery(RawSearchParams{Term: term, Limit: limit})
	if err != nil {
		t.Fatalf("ParseQuery(%q, %d): %v", term, limit, err)
	}
	return params
}

func buildFixtureIndex(t *testing.T) Index {
	t.Helper()

	return buildIndexFromPath(t, "../testdata/etl/movies.jsonl")
}

func buildOneByteRegressionIndex(t *testing.T) Index {
	t.Helper()

	return buildIndexFromMovies(t, oneByteRegressionMovies())
}

func oneByteRegressionMovies() []movies.Movie {
	return []movies.Movie{
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
}

func buildSlotBoundaryIndex(t *testing.T) Index {
	t.Helper()

	records := make([]movies.Movie, 66)
	for i := range records {
		records[i] = movies.Movie{
			ID:           1_000_000 + i,
			PrimaryTitle: "Plain",
		}
	}
	records[63].PrimaryTitle = "Zulu"
	records[64].PrimaryTitle = "Zulu"

	return buildIndexFromMovies(t, records)
}

func buildUnicodeRegressionIndex(t *testing.T) Index {
	t.Helper()

	return buildIndexFromMovies(t, unicodeRegressionMovies())
}

func unicodeRegressionMovies() []movies.Movie {
	return []movies.Movie{
		{ID: 901, PrimaryTitle: "A Écho"},
		{ID: 902, PrimaryTitle: "Écho"},
		{ID: 903, PrimaryTitle: "A Other"},
	}
}

func buildIndexFromMovies(t *testing.T, records []movies.Movie) Index {
	t.Helper()

	return buildIndexFromPath(t, writeMoviesJSONL(t, records))
}

func buildIndexFromPath(t *testing.T, path string) Index {
	t.Helper()

	index, _, err := BuildIndexFromRecordStream(path)
	if err != nil {
		t.Fatal(err)
	}
	return index
}

func bruteForceSearch(records []movies.Movie, query string, limit int) SearchResult {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	queryWords := strings.Fields(normalizedQuery)
	candidates := make([]scoredItem, 0, len(records))
	for i := range records {
		normalizedTitle := strings.ToLower(records[i].PrimaryTitle)
		matches := true
		for _, word := range queryWords {
			if !strings.Contains(normalizedTitle, word) {
				matches = false
				break
			}
		}
		if matches {
			item := &IndexRecordItem{
				Movie:           records[i],
				bayesianRating:  records[i].BayesianRating(),
				normalizedTitle: normalizedTitle,
			}
			candidates = append(candidates, scoredItem{item: item, score: rankingScore(item, normalizedQuery)})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return scoredItemLower(candidates[j], candidates[i])
	})

	resultCount := min(limit, len(candidates))
	result := SearchResult{
		Total:  len(candidates),
		Movies: make([]movies.Movie, resultCount),
	}
	for i := range result.Movies {
		result.Movies[i] = candidates[i].item.Movie
	}
	return result
}

func movieIDs(movies []movies.Movie) []int {
	ids := make([]int, len(movies))
	for i, movie := range movies {
		ids[i] = movie.ID
	}
	return ids
}
