package autocomplete

import (
	"errors"
	"fmt"
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

// func TestProcessRecordMetadata(func)
