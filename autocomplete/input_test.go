package autocomplete

import (
	"slices"
	"strings"
	"testing"
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
		{name: "valid word uses default limit", raw: RawSearchParams{Term: "Star"}, wantQuery: "star", wantWords: []string{"star"}, wantLimit: 10},
		{name: "valid words", raw: RawSearchParams{Term: "Star Wars"}, wantQuery: "star wars", wantWords: []string{"star", "wars"}, wantLimit: 10},
		{name: "leading and trailing whitespace", raw: RawSearchParams{Term: " Star Wars "}, wantQuery: "star wars", wantWords: []string{"star", "wars"}, wantLimit: 10},
		{name: "internal whitespace", raw: RawSearchParams{Term: "Star  Wars"}, wantQuery: "star  wars", wantWords: []string{"star", "wars"}, wantLimit: 10},
		{name: "non-ASCII characters", raw: RawSearchParams{Term: "世界"}, wantQuery: "世界", wantWords: []string{"世界"}, wantLimit: 10},
		{name: "punctuation", raw: RawSearchParams{Term: "Star!"}, wantQuery: "star!", wantWords: []string{"star!"}, wantLimit: 10},
		{name: "zero limit", raw: RawSearchParams{Term: "Star", Limit: intPointer(0)}, wantQuery: "star", wantWords: []string{"star"}, wantLimit: 0},
		{name: "maximum limit", raw: RawSearchParams{Term: "Star", Limit: intPointer(100)}, wantQuery: "star", wantWords: []string{"star"}, wantLimit: 100},
		{name: "empty query", raw: RawSearchParams{}, wantErr: true},
		{name: "whitespace-only query", raw: RawSearchParams{Term: " \t\n"}, wantErr: true},
		{name: "query at byte limit", raw: RawSearchParams{Term: strings.Repeat("a", 64)}, wantQuery: strings.Repeat("a", 64), wantWords: []string{strings.Repeat("a", 64)}, wantLimit: 10},
		{name: "query over byte limit", raw: RawSearchParams{Term: strings.Repeat("a", 65)}, wantErr: true},
		{name: "invalid byte", raw: RawSearchParams{Term: string([]byte{0xff})}, wantErr: true},
		{name: "invalid bytes", raw: RawSearchParams{Term: string([]byte{0xc3, 0x28})}, wantErr: true},
		{name: "negative limit", raw: RawSearchParams{Term: "star", Limit: intPointer(-1)}, wantErr: true},
		{name: "limit over maximum", raw: RawSearchParams{Term: "star", Limit: intPointer(101)}, wantErr: true},
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

func intPointer(value int) *int {
	return &value
}
