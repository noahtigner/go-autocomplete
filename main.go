package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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

func writeJSON(w http.ResponseWriter, status int, content any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(content)
}

func writeError(w http.ResponseWriter, status int, message string) {
	errJson := struct {
		Detail string `json:"detail"`
	}{Detail: message}
	err := writeJSON(w, status, &errJson)
	if err != nil {
		fmt.Println("Server:", err)
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
			fmt.Println("Server:", err)
			writeError(w, http.StatusBadRequest, "Error parsing limit")
			return
		}
		limit = parsed
	}

	genres := queryParams["genre"]
	titleTypes := queryParams["type"]

	query, err := autocomplete.ParseQuery(autocomplete.RawSearchParams{Term: q, Limit: limit, Genres: genres, TitleTypes: titleTypes})
	if err != nil {
		fmt.Println("Server:", err)
		writeError(w, http.StatusBadRequest, "Error parsing query")
		return
	}

	select {
	case <-ctx.Done():
		err := ctx.Err()
		fmt.Println("Server:", err)
		writeError(w, http.StatusGatewayTimeout, "Connection closed")
	default:
		searchStart := time.Now()
		results := idx.Search(query)
		searchDuration := time.Since(searchStart)
		fmt.Printf("Found %d results in %.2fs\n", results.Total, searchDuration.Seconds())
		err := writeJSON(w, http.StatusOK, results)
		if err != nil {
			fmt.Println("Server:", err)
			writeError(w, http.StatusInternalServerError, "Error writing response")
		}

	}
}

func cors(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Vary", "Origin")

		if req.Header.Get("Origin") == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if req.Method == http.MethodOptions {
			if req.Header.Get("Origin") != allowedOrigin {
				http.Error(w, "CORS origin denied", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func main() {
	allowedOrigin, present := os.LookupEnv("CORS_ALLOWED_ORIGIN")
	allowedOrigin = strings.TrimSpace(allowedOrigin)

	if !present || allowedOrigin == "" {
		fmt.Fprintln(os.Stderr, "CORS_ALLOWED_ORIGIN must be set")
		os.Exit(1)
	}

	ioStart := time.Now()

	index, processedCount, err := autocomplete.BuildIndexFromRecordStream("./data/movies.jsonl")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ioDuration := time.Since(ioStart)
	fmt.Printf("Processed %d records in %.2fs\n", processedCount, ioDuration.Seconds())

	mux := http.NewServeMux()

	mux.HandleFunc("GET /search", func(w http.ResponseWriter, req *http.Request) {
		search(w, req, &index)
	})

	mux.HandleFunc("/hello", hello)
	mux.HandleFunc("/headers", headers)

	if err := http.ListenAndServe(":8090", cors(allowedOrigin, mux)); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
