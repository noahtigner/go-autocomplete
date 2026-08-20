package autocomplete

import (
	"strings"
	"testing"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		testName string
		query    string
		wantStr  string
		wantErr  bool
	}{
		{"Valid word", "Star", "star", false},
		{"Valid words", "Star Wars", "star wars", false},
		{"Valid words with leading whitespace", " Star Wars", "star wars", false},
		{"Valid words with trailing whitespace", "Star Wars ", "star wars", false},
		{"Valid words with non-ASCII characters", "世界", "世界", false},
		{"Valid words with punctuation", "Star!", "star!", false},
		{"Empty query", "", "", true},
		{"Empty query with whitespace", " ", "", true},
		{"Excessively long query", strings.Repeat("A", 999), "", true},
		{"Invalid byte", string([]byte{0xff}), "", true},
		{"Invalid bytes", string([]byte{0xc3, 0x28}), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			gotStr, err := ParseQuery(tt.query)
			if gotStr != tt.wantStr {
				t.Errorf("ParseQuery(%s) got %s, want %s", tt.query, gotStr, tt.wantStr)
			}
			gotErr := err != nil
			if gotErr != tt.wantErr {
				t.Errorf("ParseQuery(%s) got error: %t, want error: %t", tt.query, gotErr, tt.wantErr)
			}
		})
	}
}
