package autocomplete

import (
	"slices"
	"strings"
	"sync"

	"github.com/noahtigner/go-autocomplete/products"
	"github.com/noahtigner/go-autocomplete/sets"
)

// This approach supports substring matching across each word in the query
// It builds the different n-gram indexes concurrently and uses an optimized set intersection algorithm

type Index struct {
	unigrams        map[string][]int
	bigrams         map[string][]int
	trigrams        map[string][]int
	productNames    map[int]string
	normalizedNames map[int]string
}

func (i Index) nIndex(n int) map[string][]int {
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

func BuildIndex(products []products.Product) Index {
	reverseIndex := Index{
		unigrams:        make(map[string][]int),
		bigrams:         make(map[string][]int),
		trigrams:        make(map[string][]int),
		productNames:    make(map[int]string),
		normalizedNames: make(map[int]string),
	}

	for _, product := range products {
		reverseIndex.productNames[product.ID] = product.Name
		reverseIndex.normalizedNames[product.ID] = strings.ToLower(product.Name)
	}

	var wg sync.WaitGroup

	for n := 1; n <= 3; n += 1 {
		wg.Go(func() {
			index := reverseIndex.nIndex(n)
			lastSeen := make(map[string]int)

			for _, product := range products {
				normalizedName := reverseIndex.normalizedNames[product.ID]

				for word := range strings.FieldsSeq(normalizedName) {
					for _, gram := range getNGrams(word, n) {
						// prevent duplicate products caused by repeated grams
						if previous, exists := lastSeen[gram]; exists && previous == product.ID {
							continue
						}

						lastSeen[gram] = product.ID
						index[gram] = append(index[gram], product.ID)
					}
				}
			}
		})
	}

	wg.Wait()

	return reverseIndex
}

func retrieveSearchCandidates(reverseIndex Index, queryWords []string) sets.Set[int] {
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

func filterSearchCandidates(reverseIndex Index, queryWords []string, candidates sets.Set[int]) []string {
	results := make([]string, 0, len(candidates))

	for candidate := range candidates {
		matchesAll := true
		for _, word := range queryWords {
			if !strings.Contains(reverseIndex.normalizedNames[candidate], word) {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			results = append(results, reverseIndex.productNames[candidate])
		}
	}

	return results
}

func (reverseIndex Index) Search(query string) []string {
	normalizedQuery := strings.ToLower(query)
	queryWords := strings.Fields(normalizedQuery)

	if len(queryWords) == 0 {
		return []string{}
	}

	candidates := retrieveSearchCandidates(reverseIndex, queryWords)
	results := filterSearchCandidates(reverseIndex, queryWords, candidates)
	slices.Sort(results)

	return results
}
