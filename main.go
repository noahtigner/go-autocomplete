package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	autocomplete "github.com/noahtigner/go-autocomplete/autocomplete"
)

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

	index, processedCount, err := autocomplete.BuildIndexFromRecordStream("./data/movies.jsonl")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ioDuration := time.Since(ioStart)
	fmt.Printf("Processed %d records in %.2fs\n", processedCount, ioDuration.Seconds())

	searchStart := time.Now()

	results, err := index.Search(query, 10)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	searchDuration := time.Since(searchStart)

	for _, match := range results.Movies {
		yearStr := "unknown"
		if match.Year != nil {
			yearStr = strconv.Itoa(*match.Year)
		}
		fmt.Printf("\t%s (%s)\t[%d votes -> %f]\n", match.PrimaryTitle, yearStr, match.NumVotes, match.BayesianRating())
	}

	fmt.Printf("Found %d results in %.2fs\n", results.Total, searchDuration.Seconds())
}
