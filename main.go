package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	autocomplete "github.com/noahtigner/go-autocomplete/autocomplete"
	models "github.com/noahtigner/go-autocomplete/models"
)

func readFileIntoMemory(filename string) ([]models.Movie, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filename, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(bufio.NewReader(file))
	movies := make([]models.Movie, 0)

	for {
		var movie models.Movie
		err := decoder.Decode(&movie)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", filename, err)
		}

		movies = append(movies, movie)
	}

	return movies, nil
}

func main() {
	ioStart := time.Now()

	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("Error: missing 1 positional argument for the search term")
		os.Exit(1)
	}

	query := args[0]
	if len(query) == 0 {
		fmt.Println("Error: the first positional argument must be a nonempty search term")
		os.Exit(1)
	}

	records, err := readFileIntoMemory("./data/movies.jsonl")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ioDuration := time.Since(ioStart)
	fmt.Printf("Loaded %d records in %.2fs\n", len(records), ioDuration.Seconds())

	indexingStart := time.Now()
	// index := autocomplete.BuildIndex(records)
	index := autocomplete.NewIndex()
	for _, record := range records {
		index.ProcessRecordMetadata(record)
	}
	var wg sync.WaitGroup
	for n := 1; n <= 3; n++ {
		wg.Go(func() {
			for _, record := range records {
				index.ProcessRecord(record, n)
			}
		})
	}
	wg.Wait()
	index.Finalize()

	indexingDuration := time.Since(indexingStart)
	fmt.Printf("Indexed %d records in %.2fs\n", len(records), indexingDuration.Seconds())

	searchStart := time.Now()
	matches := index.Search(query)
	searchDuration := time.Since(searchStart)

	for _, match := range matches[:min(len(matches), 10)] {
		fmt.Printf("\t%s\n", match)
	}

	fmt.Printf("Found %d results in %.2fs\n", len(matches), searchDuration.Seconds())
}
