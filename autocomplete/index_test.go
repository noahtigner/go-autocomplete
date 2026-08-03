package autocomplete

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	models "github.com/noahtigner/go-autocomplete/models"
)

func TestClean(t *testing.T) {
	index := Index{
		lastSeenUnigrams: make(map[string]int),
		lastSeenBigrams:  make(map[string]int),
		lastSeenTrigrams: make(map[string]int),
	}

	index.clean()

	if index.lastSeenUnigrams != nil || index.lastSeenBigrams != nil || index.lastSeenTrigrams != nil {
		t.Errorf("clean did not properly nil the lastSeen maps")
	}
}

func TestNIndex(t *testing.T) {
	index := Index{
		unigrams: make(map[string][]int),
		bigrams:  make(map[string][]int),
		trigrams: make(map[string][]int),
	}

	tests := []struct {
		n    int
		want map[string][]int
		name string
	}{
		// expected
		{1, index.unigrams, "unigrams"},
		{2, index.bigrams, "bigrams"},
		{3, index.trigrams, "trigrams"},
		// n out of bounds
		{-1, index.trigrams, "trigrams fallback (-1)"},
		{0, index.trigrams, "trigrams fallback (0)"},
		{999, index.trigrams, "trigrams fallback (999)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := index.nIndex(tt.n)
			mutationMarker := fmt.Sprintf("test-marker-%d", tt.n)
			got[mutationMarker] = []int{tt.n}

			if !slices.Equal(tt.want[mutationMarker], []int{tt.n}) {
				t.Errorf("nIndex(%d) did not return the expected index map", tt.n)
			}
		})
	}
}

func TestLastSeenNIndex(t *testing.T) {
	index := Index{
		lastSeenUnigrams: make(map[string]int),
		lastSeenBigrams:  make(map[string]int),
		lastSeenTrigrams: make(map[string]int),
	}

	tests := []struct {
		n    int
		want map[string]int
		name string
	}{
		// expected
		{1, index.lastSeenUnigrams, "lastSeenUnigrams"},
		{2, index.lastSeenBigrams, "lastSeenBigrams"},
		{3, index.lastSeenTrigrams, "lastSeenTrigrams"},
		// n out of bounds
		{-1, index.lastSeenTrigrams, "lastSeenTrigrams fallback (-1)"},
		{0, index.lastSeenTrigrams, "lastSeenTrigrams fallback (0)"},
		{999, index.lastSeenTrigrams, "lastSeenTrigrams fallback (999)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := index.lastSeenNIndex(tt.n)
			mutationMarker := fmt.Sprintf("test-marker-%d", tt.n)
			got[mutationMarker] = tt.n

			if tt.want[mutationMarker] != tt.n {
				t.Errorf("lastSeenNIndex(%d) did not return the expected last seen map", tt.n)
			}
		})
	}
}

func TestGetNGrams(t *testing.T) {
	tests := []struct {
		word string
		n    int
		want []string
		name string
	}{
		// expected
		{"apple", 1, []string{"a", "p", "p", "l", "e"}, "(apple, 1)"},
		{"apple", 2, []string{"ap", "pp", "pl", "le"}, "(apple, 2)"},
		{"apple", 3, []string{"app", "ppl", "ple"}, "(apple, 3)"},
		// n out of bounds
		{"apple", -1, []string{"a", "p", "p", "l", "e"}, "(apple, -1->1)"},
		{"apple", 0, []string{"a", "p", "p", "l", "e"}, "(apple, 0->1)"},
		{"apple", 4, []string{"app", "ppl", "ple"}, "(apple, 4->3)"},
		{"apple", 999, []string{"app", "ppl", "ple"}, "(apple, 999->3)"},
		{"", 1, []string{""}, "('', 1)"},
		{"a", 2, []string{"a"}, "(a, 2->1)"},
		{"ap", 3, []string{"ap"}, "(ap, 3->2)"},
		{"app", 3, []string{"app"}, "(app, 3)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNGrams(tt.word, tt.n)

			if !slices.Equal(got, tt.want) {
				t.Errorf("getNGrams(%s, %d) did not return the expected strings", tt.word, tt.n)
			}
		})
	}
}

func TestGramsForQueryWord(t *testing.T) {
	tests := []struct {
		word string
		want []string
		name string
	}{
		// expected
		{"a", []string{"a"}, "a"},
		{"ap", []string{"ap"}, "ap"},
		{"app", []string{"app"}, "app"},
		{"appl", []string{"app", "ppl"}, "appl"},
		// empty word
		{"", []string{""}, "empty word"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gramsForQueryWord(tt.word)

			if !slices.Equal(got, tt.want) {
				t.Errorf("gramsForQueryWord(%s) did not return the expected strings; expected %v, received %v", tt.word, tt.want, got)
			}
		})
	}
}

func TestNormalizeName(t *testing.T) {
	t.Run("normalize", func(t *testing.T) {
		testName := "testName"
		got := normalizeName(testName)
		expected := strings.ToLower(testName)

		if got != expected {
			t.Errorf("normalize(%s) did not return the expected string; expected %v, received %v", testName, expected, got)
		}
	})
}

func TestOpenJSONLFile(t *testing.T) {
	t.Run("Temporary file is opened and a decoder is returned", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "movies.jsonl")
		contents := `{"id":1,"titleType":"movie","primaryTitle":"Test","originalTitle":"Test","isAdult":false,"year":null,"runtimeMinutes":null,"genres":"Drama","averageRating":null,"numVotes":0}` + "\n"

		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}

		file, decoder, err := openJsonlFile(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = file.Close() })

		var movie models.Movie
		if err := decoder.Decode(&movie); err != nil {
			t.Fatal(err)
		}
		if movie.ID != 1 {
			t.Errorf("movie.ID = %d, expected 1", movie.ID)
		}
	})

	t.Run("Missing file", func(t *testing.T) {
		file, decoder, err := openJsonlFile(
			filepath.Join(t.TempDir(), "missing.jsonl"),
		)

		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("error = %v, want an os.ErrNotExist error", err)
		}
		if file != nil || decoder != nil {
			t.Fatal("expected nil file and decoder")
		}
	})
}

func TestProcessRecordMetadata(t *testing.T) {
	index := Index{
		records: make(map[int]*IndexRecordItem),
	}

	averageRating := 0.93
	movie := models.Movie{
		ID:             123,
		TitleType:      "movie",
		PrimaryTitle:   "Star Wars: Episode IV - A New Hope",
		OriginalTitle:  "Star Wars",
		IsAdult:        false,
		Year:           nil,
		RuntimeMinutes: nil,
		Genres:         "Action, Science Fiction",
		AverageRating:  &averageRating,
		NumVotes:       123456,
	}

	expectedBayesianRating := movie.BayesianRating()

	index.processRecordMetadata(&movie)

	indexRecord, exists := index.records[movie.ID]
	if !exists {
		t.Fatalf("processRecordMetadata did not properly store movie with ID %d", movie.ID)
	}

	if indexRecord.Movie != movie {
		t.Errorf("processRecordMetadata stored movie = %+v, want %+v", indexRecord.Movie, movie)
	}

	if indexRecord.bayesianRating != expectedBayesianRating {
		t.Errorf("processRecordMetadata did not properly calculate the rating for movie with ID %d; expected %f, got %f", movie.ID, expectedBayesianRating, indexRecord.bayesianRating)
	}
}

func TestProcessRecord(t *testing.T) {
	expectedUnigramIndex := map[string][]int{
		"s": {1, 99},
		"t": {1, 99},
		"a": {1, 99},
		"r": {1, 99},
		"w": {1},
		"i": {99},
		"b": {99},
		"o": {99},
		"n": {99},
	}

	expectedBigramIndex := map[string][]int{
		"st": {1, 99},
		"ta": {1, 99},
		"ar": {1, 99},
		"wa": {1},
		"rs": {1},
		"a":  {99},
		"is": {99},
		"bo": {99},
		"or": {99},
		"rn": {99},
	}

	expectedTrigramIndex := map[string][]int{
		"sta": {1, 99},
		"tar": {1, 99},
		"war": {1},
		"ars": {1},
		"a":   {99},
		"is":  {99},
		"bor": {99},
		"orn": {99},
	}

	tests := []struct {
		n                int
		expectedIndexMap map[string][]int
		testName         string
	}{
		{1, expectedUnigramIndex, "unigrams"},
		{2, expectedBigramIndex, "bigrams"},
		{3, expectedTrigramIndex, "trigrams"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			index := Index{
				unigrams:         make(map[string][]int),
				bigrams:          make(map[string][]int),
				trigrams:         make(map[string][]int),
				lastSeenUnigrams: make(map[string]int),
				lastSeenBigrams:  make(map[string]int),
				lastSeenTrigrams: make(map[string]int),
			}

			index.processRecord(1, "star wars", tt.n)
			index.processRecord(99, "a star is born", tt.n)

			got := index.nIndex(tt.n)
			if !maps.EqualFunc(got, tt.expectedIndexMap, func(got, want []int) bool {
				return slices.Equal(got, want)
			}) {
				t.Errorf("processRecord %s map = %v, want %v", tt.testName, got, tt.expectedIndexMap)
			}
		})
	}
}

func TestBuildIndexFromRecordStream(t *testing.T) {
	t.Run("builds from fixture", func(t *testing.T) {
		fixturePath := filepath.Join("..", "testdata", "etl", "movies.jsonl")

		index, count, err := BuildIndexFromRecordStream(fixturePath)
		if err != nil {
			t.Fatal(err)
		}
		if count != 30 {
			t.Errorf("processed count = %d, want 30", count)
		}
		if len(index.records) != 30 {
			t.Errorf("record count = %d, want 30", len(index.records))
		}

		for _, id := range []int{1, 9, 23} {
			if index.records[id] == nil {
				t.Errorf("record %d was not indexed", id)
			}
		}

		starWars := index.records[23]
		if starWars == nil {
			t.Fatal("record 23 was not indexed")
		}
		if starWars.PrimaryTitle != "Star Wars: A New Fixture" {
			t.Errorf("record 23 title = %q, want %q", starWars.PrimaryTitle, "Star Wars: A New Fixture")
		}
		if starWars.AverageRating == nil || *starWars.AverageRating != 8.6 {
			t.Errorf("record 23 average rating = %v, want 8.6", starWars.AverageRating)
		}
		if starWars.NumVotes != 1_000_000 {
			t.Errorf("record 23 votes = %d, want 1000000", starWars.NumVotes)
		}
		if starWars.bayesianRating != starWars.Movie.BayesianRating() {
			t.Errorf("record 23 cached rating = %v, want %v", starWars.bayesianRating, starWars.Movie.BayesianRating())
		}

		unrated := index.records[9]
		if unrated == nil {
			t.Fatal("record 9 was not indexed")
		}
		if unrated.AverageRating != nil || unrated.NumVotes != 0 || unrated.bayesianRating != 0 {
			t.Errorf("record 9 = %+v, want an unrated movie", unrated)
		}

		if !slices.Equal(index.trigrams["moo"], []int{26}) {
			t.Errorf("trigrams[\"moo\"] = %v, want [26]", index.trigrams["moo"])
		}
		if index.lastSeenUnigrams != nil || index.lastSeenBigrams != nil || index.lastSeenTrigrams != nil {
			t.Error("last-seen maps were not cleared")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		index, count, err := BuildIndexFromRecordStream(
			filepath.Join(t.TempDir(), "missing.jsonl"),
		)
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("error = %v, want an os.ErrNotExist error", err)
		}
		if count != 0 {
			t.Errorf("processed count = %d, want 0", count)
		}
		assertZeroIndex(t, index)
	})

	t.Run("invalid JSONL", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "movies.jsonl")
		contents := `{"id":1,"titleType":"movie","primaryTitle":"Test","originalTitle":"Test","isAdult":false,"year":null,"runtimeMinutes":null,"genres":"Drama","averageRating":null,"numVotes":0}` + "\ninvalid JSON\n"
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}

		index, count, err := BuildIndexFromRecordStream(path)
		if err == nil {
			t.Fatal("expected an error for invalid JSONL")
		}
		if count != 0 {
			t.Errorf("processed count = %d, want 0", count)
		}
		assertZeroIndex(t, index)
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "movies.jsonl")
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}

		index, count, err := BuildIndexFromRecordStream(path)
		if err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("processed count = %d, want 0", count)
		}
		if index.unigrams == nil || index.bigrams == nil || index.trigrams == nil || index.records == nil {
			t.Fatal("successful empty build returned an uninitialized index")
		}
		if len(index.unigrams) != 0 || len(index.bigrams) != 0 || len(index.trigrams) != 0 || len(index.records) != 0 {
			t.Error("empty input produced index entries")
		}
		if index.lastSeenUnigrams != nil || index.lastSeenBigrams != nil || index.lastSeenTrigrams != nil {
			t.Error("last-seen maps were not cleared")
		}
	})
}

func assertZeroIndex(t *testing.T, index Index) {
	t.Helper()

	if index.unigrams != nil || index.bigrams != nil || index.trigrams != nil || index.records != nil ||
		index.lastSeenUnigrams != nil || index.lastSeenBigrams != nil || index.lastSeenTrigrams != nil {
		t.Errorf("index = %+v, want a zero-value index", index)
	}
}
