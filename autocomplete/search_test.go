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
		{name: "invalid UTF-8", query: string([]byte{0xff}), limit: 10, wantTotal: 0, wantIDs: []int{}},
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

func TestOldAndNewSearchParity(t *testing.T) {
	fixtureIndex := buildFixtureIndex(t)
	oneByteIndex := buildOneByteRegressionIndex(t)
	slotBoundaryIndex := buildSlotBoundaryIndex(t)
	unicodeIndex := buildUnicodeRegressionIndex(t)

	tests := []struct {
		name      string
		index     Index
		query     string
		limit     int
		wantTotal int
		wantIDs   []int
		wantError string
	}{
		{name: "case insensitive ranking", index: fixtureIndex, query: "STAR", limit: 3, wantTotal: 12, wantIDs: []int{23, 25, 4}},
		{name: "short multiword", index: fixtureIndex, query: "THE A", limit: 10, wantTotal: 3, wantIDs: []int{4, 8, 21}},
		{name: "long query verification", index: fixtureIndex, query: "abcd", limit: 10, wantIDs: []int{}},
		{name: "zero limit", index: fixtureIndex, query: "star", limit: 0, wantTotal: 12, wantIDs: []int{}},
		{name: "one byte", index: oneByteIndex, query: "a", limit: 10, wantTotal: 5, wantIDs: []int{39_063_631, 26_700_024, 103, 101, 2}},
		{name: "one byte limit one", index: oneByteIndex, query: "a", limit: 1, wantTotal: 5, wantIDs: []int{39_063_631}},
		{name: "one byte limit hundred", index: oneByteIndex, query: "a", limit: 100, wantTotal: 5, wantIDs: []int{39_063_631, 26_700_024, 103, 101, 2}},
		{name: "repeated one byte", index: oneByteIndex, query: "a a", limit: 10, wantTotal: 5, wantIDs: []int{39_063_631, 26_700_024, 103, 101, 2}},
		{name: "one byte intersection", index: oneByteIndex, query: "a b", limit: 10, wantTotal: 2, wantIDs: []int{39_063_631, 26_700_024}},
		{name: "one byte miss", index: oneByteIndex, query: "q", limit: 10, wantIDs: []int{}},
		{name: "one byte intersection miss", index: oneByteIndex, query: "a q", limit: 10, wantIDs: []int{}},
		{name: "punctuation", index: oneByteIndex, query: "!", limit: 10, wantTotal: 1, wantIDs: []int{400}},
		{name: "invalid UTF-8", index: oneByteIndex, query: string([]byte{0xff}), limit: 10, wantIDs: []int{}},
		{name: "unicode", index: oneByteIndex, query: "É", limit: 10, wantTotal: 1, wantIDs: []int{500}},
		{name: "mixed bigram", index: oneByteIndex, query: "a it", limit: 10, wantTotal: 1, wantIDs: []int{101}},
		{name: "mixed trigram", index: oneByteIndex, query: "a the", limit: 10, wantTotal: 1, wantIDs: []int{103}},
		{name: "mixed long rejection", index: oneByteIndex, query: "z orbit", limit: 10, wantIDs: []int{}},
		{name: "slot boundary", index: slotBoundaryIndex, query: "z", limit: 10, wantTotal: 2, wantIDs: []int{1_000_064, 1_000_063}},
		{name: "mixed unicode ASCII", index: unicodeIndex, query: "é a", limit: 10, wantTotal: 1, wantIDs: []int{901}},
		{name: "verified unicode ASCII", index: unicodeIndex, query: "écho a", limit: 10, wantTotal: 1, wantIDs: []int{901}},
		{name: "empty and invalid limit", index: fixtureIndex, query: "", limit: -1, wantError: "A limit between 0 and 100 is required"},
		{name: "large invalid limit", index: fixtureIndex, query: "star", limit: 101, wantError: "A limit between 0 and 100 is required"},
		{name: "whitespace query", index: fixtureIndex, query: " \t\n", limit: 10, wantError: "At least one query word is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldResult, oldErr := tt.index.oldSearch(tt.query, tt.limit)
			newResult, newErr := tt.index.newSearch(tt.query, tt.limit)

			if errorMessage(oldErr) != errorMessage(newErr) {
				t.Fatalf("old error = %q, new error = %q", errorMessage(oldErr), errorMessage(newErr))
			}
			if errorMessage(newErr) != tt.wantError {
				t.Fatalf("error = %q, want %q", errorMessage(newErr), tt.wantError)
			}
			if newErr != nil {
				return
			}

			assertSearchResult(t, "oldSearch", oldResult, tt.wantTotal, tt.wantIDs)
			assertSearchResult(t, "newSearch", newResult, tt.wantTotal, tt.wantIDs)
		})
	}
}

func TestSearchStrategiesMatchBruteForce(t *testing.T) {
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
				oldResult, oldErr := index.oldSearch(query, limit)
				newResult, newErr := index.newSearch(query, limit)
				if oldErr != nil || newErr != nil {
					t.Fatalf("old error = %v, new error = %v", oldErr, newErr)
				}
				assertEquivalentSearchResult(t, "oldSearch", oldResult, want)
				assertEquivalentSearchResult(t, "newSearch", newResult, want)
			})
		}
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

func assertSearchResult(t *testing.T, implementation string, result SearchResult, wantTotal int, wantIDs []int) {
	t.Helper()

	if result.Total != wantTotal {
		t.Errorf("%s total = %d, want %d", implementation, result.Total, wantTotal)
	}
	if got := movieIDs(result.Movies); !slices.Equal(got, wantIDs) {
		t.Errorf("%s IDs = %v, want %v", implementation, got, wantIDs)
	}
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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

	return buildIndexFromMovies(t, movies)
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

	return buildIndexFromMovies(t, []movies.Movie{
		{ID: 901, PrimaryTitle: "A Écho"},
		{ID: 902, PrimaryTitle: "Écho"},
		{ID: 903, PrimaryTitle: "A Other"},
	})
}

func buildIndexFromMovies(t *testing.T, records []movies.Movie) Index {
	t.Helper()

	path := writeMoviesJSONL(t, records)
	index, count, err := BuildIndexFromRecordStream(path)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(records) {
		t.Fatalf("processed %d records, want %d", count, len(records))
	}
	return index
}

func bruteForceSearch(records []movies.Movie, query string, limit int) SearchResult {
	queryWords := strings.Fields(strings.ToLower(query))
	candidates := make([]*IndexRecordItem, 0, len(records))
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
			candidates = append(candidates, &IndexRecordItem{
				Movie:          records[i],
				bayesianRating: records[i].BayesianRating(),
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].bayesianRating != candidates[j].bayesianRating {
			return candidates[i].bayesianRating > candidates[j].bayesianRating
		}
		return candidates[i].ID > candidates[j].ID
	})

	resultCount := min(limit, len(candidates))
	result := SearchResult{
		Total:  len(candidates),
		Movies: make([]movies.Movie, resultCount),
	}
	for i := range result.Movies {
		result.Movies[i] = candidates[i].Movie
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
