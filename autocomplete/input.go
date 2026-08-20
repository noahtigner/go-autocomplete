package autocomplete

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type RawSearchParams struct {
	Term  string
	Limit *int
}

func ParseQuery(query RawSearchParams) (SearchParams, error) {
	limit := 10
	if query.Limit != nil {
		if *query.Limit < 0 || *query.Limit > 100 {
			return SearchParams{}, fmt.Errorf("Limit must be be in the range [0, 100]")
		}
		limit = *query.Limit
	}

	if len(query.Term) > 64 {
		return SearchParams{}, fmt.Errorf("Query must not exceed 64 bytes")
	}

	trimmedQuery := strings.TrimSpace(query.Term)

	if len(trimmedQuery) == 0 {
		return SearchParams{}, fmt.Errorf("Query must not be empty or blank")
	}

	if !utf8.ValidString(trimmedQuery) {
		return SearchParams{}, fmt.Errorf("Query string must only contain valid UTF-8 characters")
	}

	normalizedString := strings.ToLower(trimmedQuery)

	return SearchParams{
		normalizedQuery:      normalizedString,
		normalizedQuerySlice: strings.Fields(normalizedString),
		limit:                limit,
	}, nil
}
