package autocomplete

import (
	"fmt"
	"strings"

	models "github.com/noahtigner/go-autocomplete/models"
	sets "github.com/noahtigner/go-autocomplete/sets"
)

type SearchResult struct {
	Total  int
	Movies []models.Movie
}

func retrieveSearchCandidateIds(reverseIndex Index, queryWords []string) sets.Set[int] {
	wordResults := make([]sets.Set[int], len(queryWords))

	for i, word := range queryWords {
		var gramSets []sets.Set[int]
		grams := sets.Unique(gramsForQueryWord(word))
		index := reverseIndex.nIndex(len(word))

		for _, gram := range grams {
			gramSet := sets.NewSet[int]()
			for _, match := range index[gram] {
				gramSet.Add(match)
			}
			gramSets = append(gramSets, gramSet)
		}

		wordResults[i] = sets.Intersection(gramSets)
	}

	intersection := sets.Intersection(wordResults)
	return intersection
}

func matchesAllQueryWords(movieTitle string, queryWords []string) bool {
	normalizedTitle := normalizeName(movieTitle)
	for _, word := range queryWords {
		if !strings.Contains(normalizedTitle, word) {
			return false
		}
	}
	return true
}

func queryWordsRequireVerification(queryWords []string) bool {
	// The byte-based index stores complete query words up to trigrams.
	// If all query words are short, we can skip matchesAllQueryWords which massively reduces allocations
	for _, word := range queryWords {
		if len(word) > 3 {
			return true
		}
	}
	return false
}

func (reverseIndex Index) Search(query string, limit int) (SearchResult, error) {
	normalizedQuery := strings.ToLower(query)
	queryWords := strings.Fields(normalizedQuery)

	if len(queryWords) == 0 {
		return SearchResult{}, fmt.Errorf("At least one query word is required")
	}
	if limit < 0 || limit > 100 {
		return SearchResult{}, fmt.Errorf("A limit between 0 and 100 is required")
	}

	candidateIds := retrieveSearchCandidateIds(reverseIndex, queryWords)
	requiresVerification := queryWordsRequireVerification(queryWords)

	totalMatches := 0
	topResults := newMovieHeap(limit)

	for candidateId := range candidateIds {
		record := reverseIndex.records[candidateId]

		if requiresVerification && !matchesAllQueryWords(record.PrimaryTitle, queryWords) {
			continue
		}

		totalMatches += 1
		topResults.add(record)
	}

	heapResults := topResults.topKResults()

	return SearchResult{
		Total:  totalMatches,
		Movies: heapResults,
	}, nil
}
