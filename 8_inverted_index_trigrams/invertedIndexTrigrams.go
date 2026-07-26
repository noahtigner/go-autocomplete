package invertedIndexTrigrams

import (
	"strings"

	dataStructures "github.com/noahtigner/go-autocomplete/data_structures"
	models "github.com/noahtigner/go-autocomplete/models"
)

// This approach works when all words in the query are at least 3 characters long

func getTrigrams(word string) []string {
	if len(word) < 4 {
		return []string{word}
	}
	trigrams := make([]string, len(word)-2)
	for i := range len(word) - 2 {
		trigrams[i] = word[i : i+3]
	}
	return trigrams
}

func Search(products []models.Product, query string) []string {
	reverseIndex := make(map[string][]string)

	for _, product := range products {
		normalizedName := strings.ToLower(product.Name)
		for word := range strings.FieldsSeq(normalizedName) {
			trigrams := dataStructures.Unique(getTrigrams(word))

			for _, trigram := range trigrams {
				reverseIndex[trigram] = append(reverseIndex[trigram], product.Name)
			}
		}
	}

	normalizedQuery := strings.ToLower(query)
	queryWords := strings.Fields(normalizedQuery)

	if len(queryWords) == 0 {
		return []string{}
	}

	wordResults := make([]dataStructures.Set[string], len(queryWords))

	for i, word := range queryWords {
		var trigramSets []dataStructures.Set[string]
		trigrams := dataStructures.Unique(getTrigrams(word))

		for _, trigram := range trigrams {
			trigramSet := dataStructures.NewSet[string]()
			for _, match := range reverseIndex[trigram] {
				trigramSet.Add(match)
			}
			trigramSets = append(trigramSets, trigramSet)
		}

		wordResults[i] = trigramSets[0].Intersection(trigramSets[1:])
	}

	intersection := wordResults[0].Intersection(wordResults[1:])
	intersectedResults := make([]string, len(intersection))
	i := 0
	for intersectedWord := range intersection {
		intersectedResults[i] = intersectedWord
		i += 1
	}

	return intersectedResults
}
