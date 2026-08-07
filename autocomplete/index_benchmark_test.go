package autocomplete

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/noahtigner/go-autocomplete/models"
)

var benchmarkTitleTemplates = []string{
	"The Silent River",
	"Star Wars Episode",
	"Moonlight Archive",
	"Garden Chronicle",
	"Night Shift",
}

func generateBenchmarkMovie(i int) models.Movie {
	title := fmt.Sprintf(
		"%s %06d",
		benchmarkTitleTemplates[i%len(benchmarkTitleTemplates)],
		i,
	)
	rating := max(min(4.0+float64(i%60)/10, 9.9), 0.1)

	return models.Movie{
		ID:            i + 1,
		TitleType:     "movie",
		PrimaryTitle:  title,
		AverageRating: &rating,
		NumVotes:      100 + i%100_000,
	}
}

func writeBenchmarkJSONL(b *testing.B, records int) (string, int64) {
	b.Helper()

	path := filepath.Join(b.TempDir(), "movies.jsonl")

	file, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)

	for i := range records {
		if err := encoder.Encode(generateBenchmarkMovie(i)); err != nil {
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

func BenchmarkBuildIndex100k(b *testing.B) {
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
