package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	autocomplete "github.com/noahtigner/go-autocomplete/autocomplete"
)

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "Hello\n")
}

func headers(w http.ResponseWriter, req *http.Request) {
	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func search(w http.ResponseWriter, req *http.Request, idx *autocomplete.Index) {
	ctx := req.Context()

	queryParams := req.URL.Query()
	q := queryParams.Get("q")
	limit := 10
	if queryParams.Has("limit") {
		parsed, err := strconv.Atoi(queryParams.Get("limit"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	query, err := autocomplete.ParseQuery(autocomplete.RawSearchParams{Term: q, Limit: limit})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	select {
	case <-ctx.Done():
		err := ctx.Err()
		fmt.Println("Server:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		searchStart := time.Now()
		results := idx.Search(query)
		searchDuration := time.Since(searchStart)
		for _, match := range results.Movies {
			yearStr := "unknown"
			if match.Year != nil {
				yearStr = strconv.Itoa(*match.Year)
			}
			fmt.Fprintf(w, "\t%s (%s)\t[%d votes -> %f]\n", match.PrimaryTitle, yearStr, match.NumVotes, match.BayesianRating())
		}
		fmt.Fprintf(w, "Found %d results in %.2fs\n", results.Total, searchDuration.Seconds())
	}
}

func main() {
	ioStart := time.Now()

	index, processedCount, err := autocomplete.BuildIndexFromRecordStream("./data/movies.jsonl")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ioDuration := time.Since(ioStart)
	fmt.Printf("Processed %d records in %.2fs\n", processedCount, ioDuration.Seconds())

	http.HandleFunc("GET /search", func(w http.ResponseWriter, req *http.Request) {
		search(w, req, &index)
	})

	http.HandleFunc("/hello", hello)
	http.HandleFunc("/headers", headers)
	http.ListenAndServe(":8090", nil)
}
