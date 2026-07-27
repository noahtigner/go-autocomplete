package invertedIndexNGramsConcurrent

import (
	"strings"
	"sync"

	dataStructures "github.com/noahtigner/go-autocomplete/data_structures"
	models "github.com/noahtigner/go-autocomplete/models"
)

// This approach works supports substring matching across each word in the query
// It builds the different n-gram indexes concurrently and uses an optimized set intersection algorithm

type Index struct {
	unigrams map[string][]string
	bigrams  map[string][]string
	trigrams map[string][]string
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

func gramsForQueryWord(word string) []string {
	return getNGrams(word, len(word))
}

func BuildReverseIndex(products []models.Product) Index {
	reverseIndex := Index{
		unigrams: make(map[string][]string),
		bigrams:  make(map[string][]string),
		trigrams: make(map[string][]string),
	}

	normalizedNames := make([]string, len(products))
	for i, product := range products {
		normalizedNames[i] = strings.ToLower(product.Name)
	}

	var wg sync.WaitGroup

	for n := 1; n <= 3; n += 1 {
		wg.Go(func() {
			for _, productName := range normalizedNames {
				for word := range strings.FieldsSeq(productName) {
					grams := dataStructures.Unique(getNGrams(word, n))
					for _, gram := range grams {
						reverseIndex.nIndex(n)[gram] = append(reverseIndex.nIndex(n)[gram], productName)
					}
				}
			}
		})
	}

	wg.Wait()

	return reverseIndex
}

func Search(reverseIndex Index, query string) []string {
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

		wordResults[i] = dataStructures.Intersection(gramSets)
	}

	intersection := dataStructures.Intersection(wordResults)

	return intersection.ToSlice()
}
