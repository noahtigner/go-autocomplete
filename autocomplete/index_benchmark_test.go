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

var diverseBenchmarkTitleTemplates = []string{
	"The Alpha Archive",
	"Beta Orbit Chronicle",
	"Gamma Delta Theater",
	"Éclair Cinema",
	"7! Signal",
	"Zebra Horizon",
	"ABC X BCD",
	"Quasar Atlas",
	"Silent Meadow",
	"Night Harbor",
}

func generateBenchmarkMovie(i int) movies.Movie {
	title := fmt.Sprintf(
		"%s %06d",
		benchmarkTitleTemplates[i%len(benchmarkTitleTemplates)],
		i,
	)
	rating := max(min(4.0+float64(i%60)/10, 9.9), 0.1)

	return movies.Movie{
		ID:            i + 1,
		TitleType:     "movie",
		PrimaryTitle:  title,
		AverageRating: &rating,
		NumVotes:      100 + i%100_000,
	}
}

func generateDiverseBenchmarkMovie(i int) movies.Movie {
	title := fmt.Sprintf(
		"%s %06d",
		diverseBenchmarkTitleTemplates[i%len(diverseBenchmarkTitleTemplates)],
		i,
	)
	rating := max(min(4.0+float64(i%60)/10, 9.9), 0.1)

	return movies.Movie{
		ID:            10_000_000 + i*10_003,
		TitleType:     "movie",
		PrimaryTitle:  title,
		AverageRating: &rating,
		NumVotes:      100 + i%100_000,
	}
}

func generateSlotBoundaryBenchmarkMovie(i int) movies.Movie {
	title := "Plain"
	if i == 63 || i == 64 {
		title = "Zulu"
	}
	rating := max(min(4.0+float64(i%60)/10, 9.9), 0.1)

	return movies.Movie{
		ID:            50_000_000 + i*10_003,
		TitleType:     "movie",
		PrimaryTitle:  title,
		AverageRating: &rating,
		NumVotes:      100 + i,
	}
}

func writeBenchmarkJSONL(b *testing.B, records int) (string, int64) {
	return writeBenchmarkJSONLWithGenerator(b, records, generateBenchmarkMovie)
}

func writeDiverseBenchmarkJSONL(b *testing.B, records int) (string, int64) {
	return writeBenchmarkJSONLWithGenerator(b, records, generateDiverseBenchmarkMovie)
}

func writeSlotBoundaryBenchmarkJSONL(b *testing.B) (string, int64) {
	return writeBenchmarkJSONLWithGenerator(b, 130, generateSlotBoundaryBenchmarkMovie)
}

func writeBenchmarkJSONLWithGenerator(b *testing.B, records int, generate func(int) movies.Movie) (string, int64) {
	b.Helper()

	path := filepath.Join(b.TempDir(), "movies.jsonl")

	file, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)

	for i := range records {
		if err := encoder.Encode(generate(i)); err != nil {
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
