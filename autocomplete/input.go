package autocomplete

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/noahtigner/go-autocomplete/internal/movies"
)

type RawSearchParams struct {
	Term   string
	Limit  int
	Genres []string
}

func ParseQuery(query RawSearchParams) (SearchParams, error) {
	if query.Limit < 0 || query.Limit > 100 {
		return SearchParams{}, fmt.Errorf("Limit must be be in the range [0, 100]")
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

	genres, err := movies.NewGenreBitField()
	if err != nil {
		return SearchParams{}, err
	}

	for _, g := range query.Genres {
		if err := genres.SetGenre(strings.ToLower(g)); err != nil {
			return SearchParams{}, err
		}
	}

	normalizedString := strings.ToLower(trimmedQuery)

	return SearchParams{
		normalizedQuery:      normalizedString,
		normalizedQuerySlice: strings.Fields(normalizedString),
		limit:                query.Limit,
		genres:               genres,
	}, nil
}
