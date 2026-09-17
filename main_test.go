package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		wantTotal     int
		wantResultLen int
	}{
		{name: "default limit", url: "/search?q=star", wantStatus: http.StatusOK, wantTotal: 12, wantResultLen: 10},
		{name: "explicit limit", url: "/search?q=star&limit=1", wantStatus: http.StatusOK, wantTotal: 12, wantResultLen: 1},
		{name: "zero limit", url: "/search?q=star&limit=0", wantStatus: http.StatusOK, wantTotal: 12, wantResultLen: 0},
		{name: "genre filter", url: "/search?q=zed&genre=drama", wantStatus: http.StatusOK, wantTotal: 1, wantResultLen: 1},
		{name: "case insensitive genre filter", url: "/search?q=zed&genre=DrAmA", wantStatus: http.StatusOK, wantTotal: 1, wantResultLen: 1},
		{name: "multiple genre filters match either", url: "/search?q=zed&genre=action&genre=drama", wantStatus: http.StatusOK, wantTotal: 1, wantResultLen: 1},
		{name: "nonmatching genre filter", url: "/search?q=zed&genre=action", wantStatus: http.StatusOK, wantTotal: 0, wantResultLen: 0},
		{name: "invalid genre", url: "/search?q=zed&genre=unknown", wantStatus: http.StatusBadRequest},
		{name: "title type filter", url: "/search?q=zed&type=movie", wantStatus: http.StatusOK, wantTotal: 1, wantResultLen: 1},
		{name: "case insensitive title type filter", url: "/search?q=zed&type=Movie", wantStatus: http.StatusOK, wantTotal: 1, wantResultLen: 1},
		{name: "multiple title type filters match either", url: "/search?q=zed&type=short&type=movie", wantStatus: http.StatusOK, wantTotal: 1, wantResultLen: 1},
		{name: "nonmatching title type filter", url: "/search?q=zed&type=short", wantStatus: http.StatusOK, wantTotal: 0, wantResultLen: 0},
		{name: "genre and title type filters both match", url: "/search?q=zed&genre=drama&type=movie", wantStatus: http.StatusOK, wantTotal: 1, wantResultLen: 1},
		{name: "invalid title type", url: "/search?q=zed&type=unknown", wantStatus: http.StatusBadRequest},
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
			if tt.wantStatus != http.StatusOK {
				return
			}

			var result autocomplete.SearchResult
			if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
				t.Fatalf("decode search response: %v", err)
			}
			if result.Total != tt.wantTotal {
				t.Errorf("total = %d, want %d", result.Total, tt.wantTotal)
			}
			if got := len(result.Movies); got != tt.wantResultLen {
				t.Errorf("result count = %d, want %d", got, tt.wantResultLen)
			}
		})
	}
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

func TestCORS(t *testing.T) {
	const allowedOrigin = "https://noahtigner.com"

	handler := cors(allowedOrigin, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name                    string
		method                  string
		origin                  string
		requestedMethod         string
		wantStatus              int
		wantAllowedOrigin       string
		wantAllowedMethods      string
		wantAllowedRequestHeads string
	}{
		{
			name:                    "allows configured origin",
			method:                  http.MethodGet,
			origin:                  allowedOrigin,
			wantStatus:              http.StatusOK,
			wantAllowedOrigin:       allowedOrigin,
			wantAllowedMethods:      "GET, OPTIONS",
			wantAllowedRequestHeads: "Content-Type",
		},
		{
			name:       "does not allow another origin",
			method:     http.MethodGet,
			origin:     "https://example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:                    "allows configured preflight",
			method:                  http.MethodOptions,
			origin:                  allowedOrigin,
			requestedMethod:         http.MethodGet,
			wantStatus:              http.StatusNoContent,
			wantAllowedOrigin:       allowedOrigin,
			wantAllowedMethods:      "GET, OPTIONS",
			wantAllowedRequestHeads: "Content-Type",
		},
		{
			name:            "rejects another origin preflight",
			method:          http.MethodOptions,
			origin:          "https://example.com",
			requestedMethod: http.MethodGet,
			wantStatus:      http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, "/search", nil)
			request.Header.Set("Origin", tt.origin)
			if tt.requestedMethod != "" {
				request.Header.Set("Access-Control-Request-Method", tt.requestedMethod)
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if got := response.Code; got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllowedOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantAllowedOrigin)
			}
			if got := response.Header().Get("Access-Control-Allow-Methods"); got != tt.wantAllowedMethods {
				t.Errorf("Access-Control-Allow-Methods = %q, want %q", got, tt.wantAllowedMethods)
			}
			if got := response.Header().Get("Access-Control-Allow-Headers"); got != tt.wantAllowedRequestHeads {
				t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, tt.wantAllowedRequestHeads)
			}
			if got := response.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary = %q, want %q", got, "Origin")
			}
		})
	}
}
