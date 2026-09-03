package autocomplete

import (
	"slices"
	"strings"
	"testing"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name      string
		raw       RawSearchParams
		wantQuery string
		wantWords []string
		wantLimit int
		wantErr   bool
	}{
		{name: "valid word", raw: RawSearchParams{Term: "Star", Limit: 10}, wantQuery: "star", wantWords: []string{"star"}, wantLimit: 10},
		{name: "valid words", raw: RawSearchParams{Term: "Star Wars", Limit: 10}, wantQuery: "star wars", wantWords: []string{"star", "wars"}, wantLimit: 10},
		{name: "leading and trailing whitespace", raw: RawSearchParams{Term: " Star Wars ", Limit: 10}, wantQuery: "star wars", wantWords: []string{"star", "wars"}, wantLimit: 10},
		{name: "internal whitespace", raw: RawSearchParams{Term: "Star  Wars", Limit: 10}, wantQuery: "star  wars", wantWords: []string{"star", "wars"}, wantLimit: 10},
		{name: "non-ASCII characters", raw: RawSearchParams{Term: "世界", Limit: 10}, wantQuery: "世界", wantWords: []string{"世界"}, wantLimit: 10},
		{name: "punctuation", raw: RawSearchParams{Term: "Star!", Limit: 10}, wantQuery: "star!", wantWords: []string{"star!"}, wantLimit: 10},
		{name: "zero limit", raw: RawSearchParams{Term: "Star", Limit: 0}, wantQuery: "star", wantWords: []string{"star"}, wantLimit: 0},
		{name: "maximum limit", raw: RawSearchParams{Term: "Star", Limit: 100}, wantQuery: "star", wantWords: []string{"star"}, wantLimit: 100},
		{name: "empty query", raw: RawSearchParams{}, wantErr: true},
		{name: "whitespace-only query", raw: RawSearchParams{Term: " \t\n"}, wantErr: true},
		{name: "query at byte limit", raw: RawSearchParams{Term: strings.Repeat("a", 64), Limit: 10}, wantQuery: strings.Repeat("a", 64), wantWords: []string{strings.Repeat("a", 64)}, wantLimit: 10},
		{name: "query over byte limit", raw: RawSearchParams{Term: strings.Repeat("a", 65)}, wantErr: true},
		{name: "invalid byte", raw: RawSearchParams{Term: string([]byte{0xff})}, wantErr: true},
		{name: "invalid bytes", raw: RawSearchParams{Term: string([]byte{0xc3, 0x28})}, wantErr: true},
		{name: "negative limit", raw: RawSearchParams{Term: "star", Limit: -1}, wantErr: true},
		{name: "limit over maximum", raw: RawSearchParams{Term: "star", Limit: 101}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQuery(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseQuery(%+v) error = %v, want error: %t", tt.raw, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.normalizedQuery != tt.wantQuery {
				t.Errorf("normalized query = %q, want %q", got.normalizedQuery, tt.wantQuery)
			}
			if !slices.Equal(got.normalizedQuerySlice, tt.wantWords) {
				t.Errorf("normalized words = %q, want %q", got.normalizedQuerySlice, tt.wantWords)
			}
			if got.limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", got.limit, tt.wantLimit)
			}
		})
	}
}

func TestParseQueryGenres(t *testing.T) {
	tests := []struct {
		name      string
		genres    []string
		want      uint32
		wantError bool
	}{
		{name: "case insensitive", genres: []string{"DrAmA"}, want: 1 << 8},
		{name: "multiple genres", genres: []string{"action", "Sci-Fi"}, want: 1<<0 | 1<<21},
		{name: "duplicates are idempotent", genres: []string{"drama", "drama"}, want: 1 << 8},
		{name: "empty genre", genres: []string{""}, wantError: true},
		{name: "unknown genre", genres: []string{"unknown"}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQuery(RawSearchParams{Term: "star", Limit: 10, Genres: tt.genres})
			if (err != nil) != tt.wantError {
				t.Fatalf("ParseQuery error = %v, want error: %t", err, tt.wantError)
			}
			if tt.wantError {
				return
			}
			if uint32(got.genres) != tt.want {
				t.Errorf("genres = %032b, want %032b", got.genres, tt.want)
			}
		})
	}

}

func TestParseQueryTitleTypes(t *testing.T) {
	tests := []struct {
		name       string
		titleTypes []string
		want       []movies.TitleTypeEnum
		wantError  bool
	}{
		{name: "case insensitive", titleTypes: []string{"TvSeries"}, want: []movies.TitleTypeEnum{7}},
		{name: "multiple title types", titleTypes: []string{"movie", "TVEPISODE"}, want: []movies.TitleTypeEnum{1, 3}},
		{name: "duplicates are preserved", titleTypes: []string{"movie", "movie"}, want: []movies.TitleTypeEnum{1, 1}},
		{name: "empty title type", titleTypes: []string{""}, wantError: true},
		{name: "unknown title type", titleTypes: []string{"unknown"}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQuery(RawSearchParams{Term: "star", Limit: 10, TitleTypes: tt.titleTypes})
			if (err != nil) != tt.wantError {
				t.Fatalf("ParseQuery error = %v, want error: %t", err, tt.wantError)
			}
			if tt.wantError {
				return
			}
			if !slices.Equal(got.titleTypes, tt.want) {
				t.Errorf("title types = %v, want %v", got.titleTypes, tt.want)
			}
		})
	}
}
