package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	autocomplete "github.com/noahtigner/go-autocomplete/autocomplete"
)

func TestSearchHandler(t *testing.T) {
	index, _, err := autocomplete.BuildIndexFromRecordStream("testdata/etl/movies.jsonl")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		url           string
		wantStatus    int
		wantResultLen int
		wantFound     bool
	}{
		{name: "default limit", url: "/search?q=star", wantStatus: http.StatusOK, wantResultLen: 10, wantFound: true},
		{name: "explicit limit", url: "/search?q=star&limit=1", wantStatus: http.StatusOK, wantResultLen: 1, wantFound: true},
		{name: "zero limit", url: "/search?q=star&limit=0", wantStatus: http.StatusOK, wantResultLen: 0, wantFound: true},
		{name: "missing query", url: "/search", wantStatus: http.StatusBadRequest},
		{name: "blank query", url: "/search?q=+", wantStatus: http.StatusBadRequest},
		{name: "invalid limit", url: "/search?q=star&limit=one", wantStatus: http.StatusBadRequest},
		{name: "limit above maximum", url: "/search?q=star&limit=101", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.url, nil)
			response := httptest.NewRecorder()

			search(response, request, &index)

			if got := response.Code; got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if got := resultLineCount(response.Body.String()); got != tt.wantResultLen {
				t.Errorf("result count = %d, want %d; response = %q", got, tt.wantResultLen, response.Body.String())
			}
			if got := strings.Contains(response.Body.String(), "Found "); got != tt.wantFound {
				t.Errorf("contains search summary = %t, want %t; response = %q", got, tt.wantFound, response.Body.String())
			}
		})
	}
}

func resultLineCount(response string) int {
	count := 0
	for line := range strings.SplitSeq(response, "\n") {
		if strings.HasPrefix(line, "\t") {
			count++
		}
	}
	return count
}

func TestSearchRouteRejectsNonGETRequests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search", func(w http.ResponseWriter, req *http.Request) {})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/search?q=star", nil))

	if got := response.Code; got != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", got, http.StatusMethodNotAllowed)
	}
}
