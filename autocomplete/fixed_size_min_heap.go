package autocomplete

import (
	"container/heap"
	"slices"
	"sort"

	models "github.com/noahtigner/go-autocomplete/models"
)

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

// Required heap methods

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

// Custom methods

func newMovieHeap(limit int) *movieHeap {
	movieHeap := &movieHeap{
		items:    make([]*IndexRecordItem, 0, limit),
		capacity: limit,
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
		heap.Push(h, x)
		return
	}

	// Discard items that do not outrank the current lowest result
	if !ranksLower(h.items[0], x) {
		return
	}

	// Replace the lowest-ranked item and restore heap order; equivalent to Push + Pop
	h.items[0] = x
	heap.Fix(h, 0)
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
