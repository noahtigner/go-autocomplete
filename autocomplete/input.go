package autocomplete

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func ParseQuery(query string) (string, error) {
	if len(query) > 64 {
		return "", fmt.Errorf("Query must not exceed 64 characters")
	}

	trimmedQuery := strings.TrimSpace(query)

	if len(trimmedQuery) == 0 {
		return "", fmt.Errorf("Query must not be empty or blank")
	}

	if !utf8.ValidString(trimmedQuery) {
		return "", fmt.Errorf("Query string must only contain valid UTF-8 characters")
	}

	return strings.ToLower(trimmedQuery), nil
}
