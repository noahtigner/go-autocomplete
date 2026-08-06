package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	models "github.com/noahtigner/go-autocomplete/models"
)

func TestWriteSample(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "movies.jsonl")
	writeMovies(t, inputPath, []models.Movie{
		{ID: 1, PrimaryTitle: "One"},
		{ID: 2, PrimaryTitle: "Two"},
		{ID: 3, PrimaryTitle: "Three"},
		{ID: 4, PrimaryTitle: "Four"},
		{ID: 5, PrimaryTitle: "Five"},
	})

	firstOutput := filepath.Join(t.TempDir(), "first.jsonl")
	secondOutput := filepath.Join(t.TempDir(), "second.jsonl")
	if err := writeSample(inputPath, firstOutput, 3, 1); err != nil {
		t.Fatal(err)
	}
	if err := writeSample(inputPath, secondOutput, 3, 1); err != nil {
		t.Fatal(err)
	}

	first, err := os.ReadFile(firstOutput)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondOutput)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Error("samples with the same seed differ")
	}

	if ids := movieIDs(t, firstOutput); len(ids) != 3 || !slices.IsSorted(ids) {
		t.Errorf("sample IDs = %v, want three IDs in source order", ids)
	}
}

func TestWriteSampleRejectsOversizedSample(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "movies.jsonl")
	writeMovies(t, inputPath, []models.Movie{{ID: 1}})

	err := writeSample(inputPath, filepath.Join(t.TempDir(), "sample.jsonl"), 2, 1)
	if err == nil {
		t.Fatal("expected an error when the requested sample exceeds the source size")
	}
}

func writeMovies(t *testing.T, path string, movies []models.Movie) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	encoder := json.NewEncoder(file)
	for _, movie := range movies {
		if err := encoder.Encode(movie); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func movieIDs(t *testing.T, path string) []int {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })

	decoder := json.NewDecoder(bufio.NewReader(file))
	var ids []int
	for {
		var movie models.Movie
		if err := decoder.Decode(&movie); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}
		ids = append(ids, movie.ID)
	}
	return ids
}
