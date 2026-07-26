package invertedIndexNGrams

import (
	"strings"

	dataStructures "github.com/noahtigner/go-autocomplete/data_structures"
	models "github.com/noahtigner/go-autocomplete/models"
)

// This approach works supports substring matching across each word in the query

type Index struct {
	unigrams map[string][]string
	bigrams  map[string][]string
	trigrams map[string][]string
}

func getNGrams(word string, n int) []string {
	n = min(3, max(n, 1))
	if len(word) < n+1 {
		return []string{word}
	}
	grams := make([]string, len(word)-(n-1))
	for i := range len(word) - (n - 1) {
		grams[i] = word[i : i+n]
	}
	return grams
}

func (i Index) nIndex(n int) map[string][]string {
	n = min(3, max(n, 1))
	switch n {
	case 1:
		return i.unigrams
	case 2:
		return i.bigrams
	default:
		return i.trigrams
	}
}

func gramsForQueryWord(word string) []string {
	return getNGrams(word, len(word))
}

func Search(products []models.Product, query string) []string {
	reverseIndex := Index{
		unigrams: make(map[string][]string),
		bigrams:  make(map[string][]string),
		trigrams: make(map[string][]string),
	}

	for _, product := range products {
		normalizedName := strings.ToLower(product.Name)
		for word := range strings.FieldsSeq(normalizedName) {
			for n := 1; n <= 3; n += 1 {
				grams := dataStructures.Unique(getNGrams(word, n))
				for _, gram := range grams {
					reverseIndex.nIndex(n)[gram] = append(reverseIndex.nIndex(n)[gram], product.Name)
				}
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
		var gramSets []dataStructures.Set[string]
		grams := dataStructures.Unique(gramsForQueryWord(word))

		for _, gram := range grams {
			gramSet := dataStructures.NewSet[string]()
			for _, match := range reverseIndex.nIndex(len(word))[gram] {
				gramSet.Add(match)
			}
			gramSets = append(gramSets, gramSet)
		}

		wordResults[i] = gramSets[0].Intersection(gramSets[1:])
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
