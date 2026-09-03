package autocomplete

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
)

var benchmarkTitleTemplates = []string{
	"The Silent River",
	"Star Wars Episode",
	"Moonlight Archive",
	"Garden Chronicle",
	"Night Shift",
}

var benchmarkTitleTypes = []string{"movie", "short", "tvSeries", "tvMovie"}

func generateBenchmarkMovie(i int, titleTypeName string) movies.Movie {
	title := fmt.Sprintf(
		"%s %06d",
		benchmarkTitleTemplates[i%len(benchmarkTitleTemplates)],
		i,
	)
	rating := max(min(4.0+float64(i%60)/10, 9.9), 0.1)
	genre := "drama"
	if i%2 != 0 {
		genre = "action"
	}
	genres, err := movies.NewGenreBitField(genre)
	if err != nil {
		panic(err)
	}
	titleType, err := movies.NewTitleTypeEnum(titleTypeName)
	if err != nil {
		panic(err)
	}

	return movies.Movie{
		ID:            i + 1,
		TitleType:     titleType,
		PrimaryTitle:  title,
		Genres:        genres,
		AverageRating: &rating,
		NumVotes:      100 + i%100_000,
	}
}

func writeBenchmarkJSONL(b *testing.B, records int) (string, int64) {
	return writeBenchmarkJSONLWithTitleTypes(b, records, false)
}

func writeSearchBenchmarkJSONL(b *testing.B, records int) (string, int64) {
	return writeBenchmarkJSONLWithTitleTypes(b, records, true)
}

func writeBenchmarkJSONLWithTitleTypes(b *testing.B, records int, mixedTitleTypes bool) (string, int64) {
	b.Helper()

	path := filepath.Join(b.TempDir(), "movies.jsonl")

	file, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)

	for i := range records {
		titleTypeName := "movie"
		if mixedTitleTypes {
			titleTypeName = benchmarkTitleTypes[(i/len(benchmarkTitleTemplates))%len(benchmarkTitleTypes)]
		}
		if err := encoder.Encode(generateBenchmarkMovie(i, titleTypeName)); err != nil {
			b.Fatal(err)
		}
	}

	if err := writer.Flush(); err != nil {
		b.Fatal(err)
	}
	if err := file.Close(); err != nil {
		b.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		b.Fatal(err)
	}

	return path, info.Size()
}

func BenchmarkBuildIndex100K(b *testing.B) {
	path, bytes := writeBenchmarkJSONL(b, 100_000)

	b.SetBytes(bytes)
	b.ReportAllocs()
	for b.Loop() {
		_, count, err := BuildIndexFromRecordStream(path)
		if err != nil {
			b.Fatal(err)
		}
		if count != 100_000 {
			b.Fatalf("processed %d records, want 100000", count)
		}
	}
}
