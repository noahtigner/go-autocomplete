package autocomplete

import (
	"sort"
	"strings"

	models "github.com/noahtigner/go-autocomplete/models"
	sets "github.com/noahtigner/go-autocomplete/sets"
)

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

func filterSearchCandidates(reverseIndex Index, queryWords []string, candidateIds sets.Set[int]) []int {
	results := make([]int, 0, len(candidateIds))

	for candidate := range candidateIds {
		matchesAll := true
		recordName := reverseIndex.records[candidate].PrimaryTitle
		for _, word := range queryWords {
			if !strings.Contains(normalizeName(recordName), word) {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			results = append(results, candidate)
		}
	}

	return results
}

func (reverseIndex Index) Search(query string) []models.Movie {
	normalizedQuery := strings.ToLower(query)
	queryWords := strings.Fields(normalizedQuery)

	if len(queryWords) == 0 {
		return []models.Movie{}
	}

	candidateIds := retrieveSearchCandidateIds(reverseIndex, queryWords)
	resultIds := filterSearchCandidates(reverseIndex, queryWords, candidateIds)

	results := make([]models.Movie, len(resultIds))
	for i, id := range resultIds {
		results[i] = *reverseIndex.records[id]
	}

	// Sort by weighted rating (desc), falling back to ID (asc) if both are unrated
	sort.Slice(results, func(i int, j int) bool {
		leftScore := results[i].BayesianRating()
		rightScore := results[j].BayesianRating()

		if leftScore != rightScore {
			return leftScore > rightScore
		}

		return results[i].ID > results[j].ID
	})

	return results
}
