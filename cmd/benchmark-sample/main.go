package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"

	models "github.com/noahtigner/go-autocomplete/models"
)

type sampledMovie struct {
	movie    models.Movie
	position int
}

func main() {
	inputPath := flag.String("input", "data/movies.jsonl", "source JSONL file")
	outputPath := flag.String("output", "data/benchmark-movies.jsonl", "sample JSONL file")
	count := flag.Int("count", 100_000, "number of records to sample")
	seed := flag.Int64("seed", 1, "reservoir sampling seed")
	flag.Parse()

	if *count <= 0 {
		fmt.Fprintln(os.Stderr, "count must be positive")
		os.Exit(1)
	}

	if err := writeSample(*inputPath, *outputPath, *count, *seed); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeSample(inputPath, outputPath string, count int, seed int64) (err error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open source data: %w", err)
	}
	defer func() {
		err = errors.Join(err, input.Close())
	}()

	decoder := json.NewDecoder(bufio.NewReader(input))
	random := rand.New(rand.NewSource(seed))
	sample := make([]sampledMovie, 0, count)

	for position := 0; ; position++ {
		var movie models.Movie
		if err := decoder.Decode(&movie); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("decode source data: %w", err)
		}

		entry := sampledMovie{movie: movie, position: position}
		if len(sample) < count {
			sample = append(sample, entry)
			continue
		}

		if replacement := random.Intn(position + 1); replacement < count {
			sample[replacement] = entry
		}
	}

	if len(sample) < count {
		return fmt.Errorf("source data has %d records, fewer than requested %d", len(sample), count)
	}

	sort.Slice(sample, func(i, j int) bool {
		return sample[i].position < sample[j].position
	})

	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create sample data: %w", err)
	}
	defer func() {
		err = errors.Join(err, output.Close())
	}()

	writer := bufio.NewWriter(output)
	defer func() {
		err = errors.Join(err, writer.Flush())
	}()

	encoder := json.NewEncoder(writer)
	for _, entry := range sample {
		if err := encoder.Encode(entry.movie); err != nil {
			return fmt.Errorf("encode sample data: %w", err)
		}
	}

	return nil
}
