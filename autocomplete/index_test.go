package autocomplete

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
)

func TestGetNGrams(t *testing.T) {
	tests := []struct {
		word string
		n    int
		want []string
		name string
	}{
		{"apple", 1, []string{"a", "p", "p", "l", "e"}, "unigrams"},
		{"apple", 2, []string{"ap", "pp", "pl", "le"}, "bigrams"},
		{"apple", 3, []string{"app", "ppl", "ple"}, "trigrams"},
		{"apple", 0, []string{"a", "p", "p", "l", "e"}, "zero uses unigrams"},
		{"apple", 4, []string{"app", "ppl", "ple"}, "large n uses trigrams"},
		{"a", 2, []string{"a"}, "word shorter than n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNGrams(tt.word, tt.n); !slices.Equal(got, tt.want) {
				t.Errorf("getNGrams(%q, %d) = %v, want %v", tt.word, tt.n, got, tt.want)
			}
		})
	}
}

func TestGramsForQueryWord(t *testing.T) {
	tests := []struct {
		word string
		want []string
	}{
		{"a", []string{"a"}},
		{"ap", []string{"ap"}},
		{"app", []string{"app"}},
		{"appl", []string{"app", "ppl"}},
		{"", []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			if got := gramsForQueryWord(tt.word); !slices.Equal(got, tt.want) {
				t.Errorf("gramsForQueryWord(%q) = %v, want %v", tt.word, got, tt.want)
			}
		})
	}
}

func TestBuildIndexFromRecordStream(t *testing.T) {
	t.Run("builds searchable records", func(t *testing.T) {
		index, count, err := BuildIndexFromRecordStream("../testdata/etl/movies.jsonl")
		if err != nil {
			t.Fatal(err)
		}
		if count != 31 {
			t.Fatalf("processed count = %d, want 31", count)
		}

		result, err := index.newSearch("moo", 10)
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 1 || !slices.Equal(movieIDs(result.Movies), []int{26}) {
			t.Errorf("Search(\"moo\") = %+v, want movie 26", result)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, count, err := BuildIndexFromRecordStream(filepath.Join(t.TempDir(), "missing.jsonl"))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("error = %v, want an os.ErrNotExist error", err)
		}
		if count != 0 {
			t.Errorf("processed count = %d, want 0", count)
		}
	})

	t.Run("invalid JSONL", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "movies.jsonl")
		if err := os.WriteFile(path, []byte("invalid JSON\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		_, count, err := BuildIndexFromRecordStream(path)
		if err == nil {
			t.Fatal("expected an error for invalid JSONL")
		}
		if count != 0 {
			t.Errorf("processed count = %d, want 0", count)
		}
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

		result, err := index.newSearch("a", 10)
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 0 || len(result.Movies) != 0 {
			t.Errorf("Search(\"a\") = %+v, want no matches", result)
		}
	})

	t.Run("duplicate ID", func(t *testing.T) {
		path := writeMoviesJSONL(t, []movies.Movie{
			{ID: 1, PrimaryTitle: "First"},
			{ID: 1, PrimaryTitle: "Second"},
		})

		_, count, err := BuildIndexFromRecordStream(path)
		if err == nil || !strings.Contains(err.Error(), "Duplicate record with id 1") {
			t.Fatalf("error = %v, want duplicate ID error", err)
		}
		if count != 0 {
			t.Errorf("processed count = %d, want 0", count)
		}
	})
}

func TestBuildIndexRecordLimit(t *testing.T) {
	oldMaxRecords := maxRecords
	maxRecords = 2
	t.Cleanup(func() { maxRecords = oldMaxRecords })

	t.Run("exact limit", func(t *testing.T) {
		path := writeMoviesJSONL(t, []movies.Movie{
			{ID: 1, PrimaryTitle: "Alpha"},
			{ID: 2, PrimaryTitle: "Zulu"},
		})

		index, count, err := BuildIndexFromRecordStream(path)
		if err != nil {
			t.Fatal(err)
		}
		if count != maxRecords {
			t.Fatalf("processed count = %d, want %d", count, maxRecords)
		}

		result, err := index.newSearch("z", 10)
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 1 || !slices.Equal(movieIDs(result.Movies), []int{2}) {
			t.Errorf("Search(\"z\") = %+v, want movie 2", result)
		}
	})

	t.Run("one record over limit", func(t *testing.T) {
		path := writeMoviesJSONL(t, []movies.Movie{
			{ID: 1, PrimaryTitle: "Alpha"},
			{ID: 2, PrimaryTitle: "Bravo"},
			{ID: 3, PrimaryTitle: "Charlie"},
		})

		_, count, err := BuildIndexFromRecordStream(path)
		if err == nil || !strings.Contains(err.Error(), "Maximum of 2 records exceeded") {
			t.Fatalf("error = %v, want maximum-records error", err)
		}
		if count != 0 {
			t.Errorf("processed count = %d, want 0", count)
		}
	})
}

func writeMoviesJSONL(t *testing.T, records []movies.Movie) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "movies.jsonl")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	encoder := json.NewEncoder(file)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	return path
}
