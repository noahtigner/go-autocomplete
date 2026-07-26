package invertedIndexPrefixAnyPosition

import (
	"strings"

	dataStructures "github.com/noahtigner/go-autocomplete/data_structures"
	models "github.com/noahtigner/go-autocomplete/models"
)

func Search(products []models.Product, query string) []string {
	reverseIndex := make(map[string][]string)

	for _, product := range products {
		for word := range strings.FieldsSeq(product.Name) {
			for i := range len(word) {
				for j := i + 1; j <= len(word); j += 1 {
					reverseIndex[word[i:j]] = append(reverseIndex[word[i:j]], product.Name)
				}
			}
		}
	}

	queryWords := strings.Fields(query)

	if len(queryWords) == 0 {
		return []string{}
	}

	wordResults := make([]dataStructures.Set[string], len(queryWords))

	for i, word := range queryWords {
		wordSet := dataStructures.NewSet[string]()
		for j := range len(word) {
			for k := j + 1; k < len(word); k += 1 {
				for _, match := range reverseIndex[word[j:k]] {
					wordSet.Add(match)
				}
			}
		}
		wordResults[i] = wordSet
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
