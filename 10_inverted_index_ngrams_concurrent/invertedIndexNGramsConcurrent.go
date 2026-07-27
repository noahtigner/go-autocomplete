package invertedIndexNGramsConcurrent

import (
	"slices"
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

	var wg sync.WaitGroup

	for n := 1; n <= 3; n += 1 {
		wg.Go(func() {
			index := reverseIndex.nIndex(n)
			lastSeen := make(map[string]int)

			for _, product := range products {
				normalizedName := strings.ToLower(product.Name)

				for word := range strings.FieldsSeq(normalizedName) {
					for _, gram := range getNGrams(word, n) {
						// prevent duplicate products caused by repeated grams
						if previous, exists := lastSeen[gram]; exists && previous == product.ID {
							continue
						}

						lastSeen[gram] = product.ID
						index[gram] = append(index[gram], normalizedName)
					}
				}
			}
		})
	}

	wg.Wait()

	return reverseIndex
}

func retrieveSearchCandidates(reverseIndex Index, queryWords []string) dataStructures.Set[string] {
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
	return intersection
}

func Search(reverseIndex Index, query string) []string {
	normalizedQuery := strings.ToLower(query)
	queryWords := strings.Fields(normalizedQuery)

	if len(queryWords) == 0 {
		return []string{}
	}

	candidates := retrieveSearchCandidates(reverseIndex, queryWords)

	for candidate := range candidates {
		for _, word := range queryWords {
			if !strings.Contains(candidate, word) {
				candidates.Remove(candidate)
			}
		}
	}

	results := candidates.ToSlice()
	slices.Sort(results)
	return results
}
