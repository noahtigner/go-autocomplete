package autocomplete

import (
	"slices"
	"strings"
	"sync"

	models "github.com/noahtigner/go-autocomplete/models"
	sets "github.com/noahtigner/go-autocomplete/sets"
)

// This approach supports substring matching across each word in the query
// It builds the different n-gram indexes concurrently and uses an optimized set intersection algorithm

type Index struct {
	unigrams        map[string][]int
	bigrams         map[string][]int
	trigrams        map[string][]int
	productNames    map[int]string
	normalizedNames map[int]string

	lastSeenUnigrams map[string]int
	lastSeenBigrams  map[string]int
	lastSeenTrigrams map[string]int
}

func (idx Index) nIndex(n int) map[string][]int {
	switch n {
	case 1:
		return idx.unigrams
	case 2:
		return idx.bigrams
	default:
		return idx.trigrams
	}
}

func (idx Index) lastSeenNIndex(n int) map[string]int {
	switch n {
	case 1:
		return idx.lastSeenUnigrams
	case 2:
		return idx.lastSeenBigrams
	default:
		return idx.lastSeenTrigrams
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

func BuildIndex(movies []models.Movie) Index {
	reverseIndex := Index{
		unigrams:        make(map[string][]int),
		bigrams:         make(map[string][]int),
		trigrams:        make(map[string][]int),
		productNames:    make(map[int]string),
		normalizedNames: make(map[int]string),
	}

	for _, movie := range movies {
		reverseIndex.productNames[movie.ID] = movie.PrimaryTitle
		reverseIndex.normalizedNames[movie.ID] = strings.ToLower(movie.PrimaryTitle)
	}

	var wg sync.WaitGroup

	for n := 1; n <= 3; n += 1 {
		wg.Go(func() {
			index := reverseIndex.nIndex(n)
			lastSeen := make(map[string]int)

			for _, movie := range movies {
				normalizedName := reverseIndex.normalizedNames[movie.ID]

				for word := range strings.FieldsSeq(normalizedName) {
					for _, gram := range getNGrams(word, n) {
						// prevent duplicate products caused by repeated grams
						if previous, exists := lastSeen[gram]; exists && previous == movie.ID {
							continue
						}

						lastSeen[gram] = movie.ID
						index[gram] = append(index[gram], movie.ID)
					}
				}
			}
		})
	}

	wg.Wait()

	return reverseIndex
}

func NewIndex() Index {
	reverseIndex := Index{
		unigrams:         make(map[string][]int),
		bigrams:          make(map[string][]int),
		trigrams:         make(map[string][]int),
		productNames:     make(map[int]string),
		normalizedNames:  make(map[int]string),
		lastSeenUnigrams: make(map[string]int),
		lastSeenBigrams:  make(map[string]int),
		lastSeenTrigrams: make(map[string]int),
	}
	return reverseIndex
}

func (idx *Index) Finalize() {
	idx.lastSeenUnigrams = nil
	idx.lastSeenBigrams = nil
	idx.lastSeenTrigrams = nil
}

func (idx Index) ProcessRecordMetadata(record models.Movie) {
	idx.productNames[record.ID] = record.PrimaryTitle
	idx.normalizedNames[record.ID] = strings.ToLower(record.PrimaryTitle)
}

func (idx Index) ProcessRecord(record models.Movie, n int) {
	index := idx.nIndex(n)
	lastSeen := idx.lastSeenNIndex(n)

	normalizedName := idx.normalizedNames[record.ID]

	for word := range strings.FieldsSeq(normalizedName) {
		for _, gram := range getNGrams(word, n) {
			// prevent duplicate products caused by repeated grams
			if previous, exists := lastSeen[gram]; exists && previous == record.ID {
				continue
			}

			lastSeen[gram] = record.ID
			index[gram] = append(index[gram], record.ID)
		}
	}
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
