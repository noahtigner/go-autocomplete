package autocomplete

import (
	"container/heap"
	"slices"
	"sort"
	"strings"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
)

const (
	exactTitleBoost  = 0.2
	prefixTitleBoost = 0.1
)

func rankingScore(item *IndexRecordItem, normalizedQuery string) float64 {
	score := item.bayesianRating
	if normalizedQuery == item.normalizedTitle {
		return score + exactTitleBoost
	}
	if strings.HasPrefix(item.normalizedTitle, normalizedQuery) {
		return score + prefixTitleBoost
	}
	return score
}

type scoredItem struct {
	item  *IndexRecordItem
	score float64
}

func scoredItemLower(a, b scoredItem) bool {
	if a.score != b.score {
		return a.score < b.score
	}
	return a.item.ID < b.item.ID
}

type movieHeap struct {
	items           []scoredItem
	capacity        int
	normalizedQuery string
}

// Required heap methods

func (h movieHeap) Len() int           { return len(h.items) }
func (h movieHeap) Less(i, j int) bool { return scoredItemLower(h.items[i], h.items[j]) }
func (h movieHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *movieHeap) Push(x any) {
	h.items = append(h.items, x.(scoredItem))
}
func (h *movieHeap) Pop() any {
	old := h.items
	n := len(old)
	x := old[n-1]
	h.items = old[0 : n-1]
	return x
}

// Custom methods

func newMovieHeap(query SearchParams) *movieHeap {
	movieHeap := &movieHeap{
		items:           make([]scoredItem, 0, query.limit),
		capacity:        query.limit,
		normalizedQuery: query.normalizedQuery,
	}
	heap.Init(movieHeap)
	return movieHeap
}
func (h *movieHeap) add(x *IndexRecordItem) {
	// Zero-capacity heap; do nothing
	if h.capacity == 0 {
		return
	}

	// Admit items until the heap reaches capacity
	if h.Len() < h.capacity {
		heap.Push(h, scoredItem{item: x, score: rankingScore(x, h.normalizedQuery)})
		return
	}

	// An exact title match receives the largest possible relevance boost, so lower-scoring
	// candidates cannot enter the top K and do not need title-relevance evaluation.
	if x.bayesianRating+exactTitleBoost < h.items[0].score {
		return
	}

	ranked := scoredItem{item: x, score: rankingScore(x, h.normalizedQuery)}

	// Discard items that do not outrank the current lowest result
	if !scoredItemLower(h.items[0], ranked) {
		return
	}

	// Replace the lowest-ranked item and restore heap order; equivalent to Push + Pop
	h.items[0] = ranked
	heap.Fix(h, 0)
}
func (h *movieHeap) topKResults() []movies.Movie {
	resultsCopy := slices.Clone(h.items)
	sort.Slice(resultsCopy, func(i, j int) bool {
		return scoredItemLower(resultsCopy[j], resultsCopy[i])
	})
	topKMovies := make([]movies.Movie, len(resultsCopy))
	for i, resultRecord := range resultsCopy {
		topKMovies[i] = resultRecord.item.Movie
	}
	return topKMovies
}
