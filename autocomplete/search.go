package autocomplete

import (
	"container/heap"
	"fmt"
	"slices"
	"sort"
	"strings"

	models "github.com/noahtigner/go-autocomplete/models"
	sets "github.com/noahtigner/go-autocomplete/sets"
)

type SearchResult struct {
	Total  int
	Movies []models.Movie
}

func ranksLower(a, b *IndexRecordItem) bool {
	leftScore := a.bayesianRating
	rightScore := b.bayesianRating

	if leftScore != rightScore {
		return leftScore < rightScore
	}

	return a.ID < b.ID
}

type movieHeap struct {
	items    []*IndexRecordItem
	capacity int
}

func (h movieHeap) Len() int           { return len(h.items) }
func (h movieHeap) Less(i, j int) bool { return ranksLower(h.items[i], h.items[j]) }
func (h movieHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *movieHeap) Push(x any) {
	h.items = append(h.items, x.(*IndexRecordItem))
}
func (h *movieHeap) Pop() any {
	old := h.items
	n := len(old)
	x := old[n-1]
	h.items = old[0 : n-1]
	return x
}

func newMovieHeap(limit int) *movieHeap {
	movieHeap := &movieHeap{
		items:    make([]*IndexRecordItem, 0, limit),
		capacity: limit,
	}
	heap.Init(movieHeap)
	return movieHeap
}
func (h *movieHeap) add(x *IndexRecordItem) {
	heap.Push(h, x)
	if h.Len() > h.capacity {
		heap.Pop(h)
	}
}
func (h *movieHeap) topKResults() []models.Movie {
	resultsCopy := slices.Clone(h.items)
	sort.Slice(resultsCopy, func(i, j int) bool {
		return ranksLower(resultsCopy[j], resultsCopy[i])
	})
	topKMovies := make([]models.Movie, len(resultsCopy))
	for i, resultRecord := range resultsCopy {
		topKMovies[i] = resultRecord.Movie
	}
	return topKMovies
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
	if limit < 0 {
		return SearchResult{}, fmt.Errorf("A non-negative limit is required")
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
