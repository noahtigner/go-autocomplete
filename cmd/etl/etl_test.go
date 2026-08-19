package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
	if titleCount != 31 {
		t.Errorf("title count = %d, want 31", titleCount)
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

func TestDownloadDataPublishesCompletedResponse(t *testing.T) {
	dataDir := t.TempDir()
	const fileName = "title.basics.tsv.gz"
	finalPath := filepath.Join(dataDir, fileName)
	writeFile(t, finalPath, "old")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "new")
	}))
	defer server.Close()

	if err := downloadData(server.Client(), server.URL+"/"+fileName, dataDir); err != nil {
		t.Fatal(err)
	}
	assertFileContents(t, finalPath, "new")
	assertNoTempFiles(t, dataDir, "."+fileName+"-")
}

func TestDownloadDataPreservesExistingFileOnCopyFailure(t *testing.T) {
	dataDir := t.TempDir()
	const fileName = "title.ratings.tsv.gz"
	finalPath := filepath.Join(dataDir, fileName)
	writeFile(t, finalPath, "old")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		_, _ = io.WriteString(w, "short")
	}))
	defer server.Close()

	err := downloadData(server.Client(), server.URL+"/"+fileName, dataDir)
	if err == nil {
		t.Fatal("downloadData returned nil error")
	}
	assertFileContents(t, finalPath, "old")
	assertNoTempFiles(t, dataDir, "."+fileName+"-")
}

func TestDownloadDataDoesNotCreateOutputForNonOKResponse(t *testing.T) {
	dataDir := t.TempDir()
	const fileName = "title.basics.tsv.gz"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := downloadData(server.Client(), server.URL+"/"+fileName, dataDir)
	if err == nil {
		t.Fatal("downloadData returned nil error")
	}
	if _, err := os.Stat(filepath.Join(dataDir, fileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("final file stat error = %v, want os.ErrNotExist", err)
	}
	assertNoTempFiles(t, dataDir, "."+fileName+"-")
}

func TestWriteMoviesJSONLPublishesCompletedMerge(t *testing.T) {
	dataDir := t.TempDir()
	titlePath := filepath.Join(dataDir, "title.basics.tsv.gz")
	ratingPath := filepath.Join(dataDir, "title.ratings.tsv.gz")
	outputPath := filepath.Join(dataDir, "movies.jsonl")
	writeGzipFile(t, titlePath, fixtureContents(t, "title.basics.tsv"))
	writeGzipFile(t, ratingPath, fixtureContents(t, "title.ratings.tsv"))
	writeFile(t, outputPath, "old output")

	titleCount, ratedTitleCount, err := writeMoviesJSONL(outputPath, titlePath, ratingPath)
	if err != nil {
		t.Fatal(err)
	}
	if titleCount != 31 || ratedTitleCount != 18 {
		t.Fatalf("counts = (%d, %d), want (31, 18)", titleCount, ratedTitleCount)
	}
	assertFileContents(t, outputPath, fixtureContents(t, "movies.jsonl"))
	assertNoTempFiles(t, dataDir, ".movies.jsonl-")
}

func TestWriteMoviesJSONLPreservesExistingOutputOnMergeFailure(t *testing.T) {
	dataDir := t.TempDir()
	titlePath := filepath.Join(dataDir, "title.basics.tsv.gz")
	ratingPath := filepath.Join(dataDir, "title.ratings.tsv.gz")
	outputPath := filepath.Join(dataDir, "movies.jsonl")
	writeGzipFile(t, titlePath, strings.Join([]string{
		"tconst\ttitleType\tprimaryTitle\toriginalTitle\tisAdult\tstartYear\tendYear\truntimeMinutes\tgenres",
		"tt0000001\tmovie\tFirst\tFirst\t0\t2000\t\\N\t90\tDrama",
		"malformed record",
	}, "\n")+"\n")
	writeGzipFile(t, ratingPath, "tconst\taverageRating\tnumVotes\ntt0000001\t8.0\t10\n")
	writeFile(t, outputPath, "old output")

	if _, _, err := writeMoviesJSONL(outputPath, titlePath, ratingPath); err == nil {
		t.Fatal("writeMoviesJSONL returned nil error")
	}
	assertFileContents(t, outputPath, "old output")
	assertNoTempFiles(t, dataDir, ".movies.jsonl-")
}

func TestWriteMoviesJSONLPreservesExistingOutputOnSourceCloseFailure(t *testing.T) {
	dataDir := t.TempDir()
	titlePath := filepath.Join(dataDir, "title.basics.tsv.gz")
	ratingPath := filepath.Join(dataDir, "title.ratings.tsv.gz")
	outputPath := filepath.Join(dataDir, "movies.jsonl")
	writeFile(t, outputPath, "old output")

	errClose := errors.New("close failed")
	openScanner := func(path string) (*bufio.Scanner, io.Closer, error) {
		switch path {
		case titlePath:
			return bufio.NewScanner(strings.NewReader("tconst\ttitleType\tprimaryTitle\toriginalTitle\tisAdult\tstartYear\tendYear\truntimeMinutes\tgenres\ntt0000001\tmovie\tFirst\tFirst\t0\t2000\t\\N\t90\tDrama\n")), errorCloser{errClose}, nil
		case ratingPath:
			return bufio.NewScanner(strings.NewReader("tconst\taverageRating\tnumVotes\ntt0000001\t8.0\t10\n")), io.NopCloser(strings.NewReader("")), nil
		default:
			return nil, nil, errors.New("unexpected source path")
		}
	}

	_, _, err := writeMoviesJSONLWithOpener(outputPath, titlePath, ratingPath, openScanner)
	if !errors.Is(err, errClose) {
		t.Fatalf("error = %v, want close error", err)
	}
	assertFileContents(t, outputPath, "old output")
	assertNoTempFiles(t, dataDir, ".movies.jsonl-")
}

type errorCloser struct {
	err error
}

func (c errorCloser) Close() error {
	return c.err
}

func fixturePath(fileName string) string {
	return filepath.Join("../..", "testdata", "etl", fileName)
}

func fixtureContents(t *testing.T, fileName string) string {
	t.Helper()

	contents, err := os.ReadFile(fixturePath(fileName))
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
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

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContents(t *testing.T, path string, want string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(contents); got != want {
		t.Errorf("contents of %s = %q, want %q", path, got, want)
	}
}

func assertNoTempFiles(t *testing.T, dir string, prefix string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			t.Errorf("temporary file %s remains", filepath.Join(dir, entry.Name()))
		}
	}
}
