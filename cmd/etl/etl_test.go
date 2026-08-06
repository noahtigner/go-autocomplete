package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestMergeTitleData(t *testing.T) {
	var output bytes.Buffer
	titleCount, ratedTitleCount, err := mergeTitleData(
		fixtureScanner(t, "title.basics.tsv"),
		fixtureScanner(t, "title.ratings.tsv"),
		json.NewEncoder(&output),
	)
	if err != nil {
		t.Fatal(err)
	}
	if titleCount != 30 {
		t.Errorf("title count = %d, want 30", titleCount)
	}
	if ratedTitleCount != 18 {
		t.Errorf("rated title count = %d, want 18", ratedTitleCount)
	}

	want, err := os.ReadFile(fixturePath("movies.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != string(want) {
		t.Errorf("merged JSONL did not match the golden fixture\ngot:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestParseIDField(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "IMDb ID", input: "tt0000023", want: 23},
		{name: "invalid ID", input: "ttinvalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIdField(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseIdField(%q) returned nil error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("parseIdField(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseMovie(t *testing.T) {
	t.Run("uses the latest available year", func(t *testing.T) {
		movie, err := parseMovie([]string{
			"tt0000001", "movie", "Title", "Original", "1", "2000", "2001", "90", "Drama",
		})
		if err != nil {
			t.Fatal(err)
		}
		if movie.ID != 1 || !movie.IsAdult {
			t.Errorf("movie identity = %+v, want ID 1 and adult true", movie)
		}
		if movie.Year == nil || *movie.Year != 2001 {
			t.Errorf("year = %v, want 2001", movie.Year)
		}
		if movie.RuntimeMinutes == nil || *movie.RuntimeMinutes != 90 {
			t.Errorf("runtime = %v, want 90", movie.RuntimeMinutes)
		}
	})

	t.Run("preserves missing optional values", func(t *testing.T) {
		movie, err := parseMovie([]string{
			"tt0000002", "movie", "Title", "Original", "0", "\\N", "\\N", "\\N", "Drama",
		})
		if err != nil {
			t.Fatal(err)
		}
		if movie.Year != nil || movie.RuntimeMinutes != nil {
			t.Errorf("movie = %+v, want nil year and runtime", movie)
		}
	})
}

func TestParseMovieErrors(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
	}{
		{name: "wrong field count", fields: []string{"tt0000001"}},
		{name: "invalid ID", fields: []string{"ttinvalid", "movie", "Title", "Original", "0", "2000", "\\N", "90", "Drama"}},
		{name: "invalid start year", fields: []string{"tt0000001", "movie", "Title", "Original", "0", "invalid", "\\N", "90", "Drama"}},
		{name: "invalid end year", fields: []string{"tt0000001", "movie", "Title", "Original", "0", "2000", "invalid", "90", "Drama"}},
		{name: "invalid runtime", fields: []string{"tt0000001", "movie", "Title", "Original", "0", "2000", "\\N", "invalid", "Drama"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseMovie(tt.fields); err == nil {
				t.Fatal("parseMovie returned nil error")
			}
		})
	}
}

func TestNextRating(t *testing.T) {
	t.Run("skips blank lines and trims carriage returns", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader("\r\n\ntt0000001\t7.5\t42\r\n"))
		got, exists, err := nextRating(scanner)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatal("expected a rating record")
		}
		want := []string{"tt0000001", "7.5", "42"}
		if !slices.Equal(got, want) {
			t.Errorf("rating fields = %v, want %v", got, want)
		}
	})

	t.Run("rejects malformed records", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader("tt0000001\t7.5\n"))
		if _, _, err := nextRating(scanner); err == nil {
			t.Fatal("nextRating returned nil error")
		}
	})

	t.Run("returns false at EOF", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader(""))
		_, exists, err := nextRating(scanner)
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Fatal("expected no rating record")
		}
	})
}

func TestOpenGzipScanner(t *testing.T) {
	t.Run("reads gzip contents", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "titles.tsv.gz")
		writeGzipFile(t, path, "first\nsecond\n")

		scanner, closer, err := openGzipScanner(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = closer.Close() })

		if !scanner.Scan() || scanner.Text() != "first" {
			t.Errorf("first scan = %q, want %q", scanner.Text(), "first")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, _, err := openGzipScanner(filepath.Join(t.TempDir(), "missing.tsv.gz"))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("error = %v, want an os.ErrNotExist error", err)
		}
	})

	t.Run("invalid gzip", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "titles.tsv.gz")
		if err := os.WriteFile(path, []byte("not gzip"), 0o644); err != nil {
			t.Fatal(err)
		}

		if _, _, err := openGzipScanner(path); err == nil {
			t.Fatal("expected an error for an invalid gzip file")
		}
	})
}

func fixturePath(fileName string) string {
	return filepath.Join("../..", "testdata", "etl", fileName)
}

func fixtureScanner(t *testing.T, fileName string) *bufio.Scanner {
	t.Helper()

	file, err := os.Open(fixturePath(fileName))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })

	return bufio.NewScanner(file)
}

func writeGzipFile(t *testing.T, path, contents string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := gzip.NewWriter(file)
	if _, err := writer.Write([]byte(contents)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
