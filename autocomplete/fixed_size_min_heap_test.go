package autocomplete

import (
	"slices"
	"testing"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
)

func TestRankingScore(t *testing.T) {
	tests := []struct {
		name  string
		item  IndexRecordItem
		query string
		want  float64
	}{
		{name: "bayesian rating", item: IndexRecordItem{bayesianRating: 7.5, normalizedTitle: "galaxy"}, query: "star wars", want: 7.5},
		{name: "exact title boost", item: IndexRecordItem{bayesianRating: 7.5, normalizedTitle: "star wars"}, query: "star wars", want: 7.7},
		{name: "prefix title boost", item: IndexRecordItem{bayesianRating: 7.5, normalizedTitle: "star wars episode"}, query: "star wars", want: 7.6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rankingScore(&tt.item, tt.query); got != tt.want {
				t.Errorf("rankingScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoredItemLowerIsAntisymmetric(t *testing.T) {
	items := []scoredItem{
		{item: &IndexRecordItem{Movie: movies.Movie{ID: 1}}, score: 7.0},
		{item: &IndexRecordItem{Movie: movies.Movie{ID: 2}}, score: 7.1},
		{item: &IndexRecordItem{Movie: movies.Movie{ID: 3}}, score: 7.1},
	}

	for i := range items {
		for j := range items {
			if i == j {
				continue
			}
			if scoredItemLower(items[i], items[j]) && scoredItemLower(items[j], items[i]) {
				t.Errorf("items %d and %d both rank lower than each other", items[i].item.ID, items[j].item.ID)
			}
		}
	}
}

func TestScoredItemLower(t *testing.T) {
	lowScore := scoredItem{item: &IndexRecordItem{Movie: movies.Movie{ID: 2}}, score: 7.0}
	highScore := scoredItem{item: &IndexRecordItem{Movie: movies.Movie{ID: 1}}, score: 7.1}
	higherID := scoredItem{item: &IndexRecordItem{Movie: movies.Movie{ID: 3}}, score: 7.1}

	if !scoredItemLower(lowScore, highScore) {
		t.Error("lower score should rank lower")
	}
	if scoredItemLower(highScore, lowScore) {
		t.Error("higher score should not rank lower")
	}
	if !scoredItemLower(highScore, higherID) {
		t.Error("lower ID should rank lower when scores tie")
	}
}

func TestMovieHeapUsesTitleRelevanceDuringAdmission(t *testing.T) {
	heap := newMovieHeap(SearchParams{limit: 1, normalizedQuery: "star wars"})
	nonExact := IndexRecordItem{Movie: movies.Movie{ID: 1}, normalizedTitle: "star wars episode", bayesianRating: 7.05}
	exact := IndexRecordItem{Movie: movies.Movie{ID: 2}, normalizedTitle: "star wars", bayesianRating: 7.0}

	heap.add(&nonExact)
	heap.add(&exact)

	if got := movieIDs(heap.topKResults()); !slices.Equal(got, []int{2}) {
		t.Errorf("topKResults IDs = %v, want [2]", got)
	}
}

func TestFixedSizeMinHeap(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		items  []IndexRecordItem
		wantID []int
	}{
		{
			name:  "zero capacity",
			limit: 0,
			items: []IndexRecordItem{
				{Movie: movies.Movie{ID: 1}, bayesianRating: 8},
				{Movie: movies.Movie{ID: 2}, bayesianRating: 9},
			},
			wantID: []int{},
		},
		{
			name:  "fills heap",
			limit: 3,
			items: []IndexRecordItem{
				{Movie: movies.Movie{ID: 1}, bayesianRating: 5},
				{Movie: movies.Movie{ID: 2}, bayesianRating: 7},
				{Movie: movies.Movie{ID: 3}, bayesianRating: 6},
			},
			wantID: []int{2, 3, 1},
		},
		{
			name:  "rejects lower-ranked item",
			limit: 3,
			items: []IndexRecordItem{
				{Movie: movies.Movie{ID: 1}, bayesianRating: 5},
				{Movie: movies.Movie{ID: 2}, bayesianRating: 7},
				{Movie: movies.Movie{ID: 3}, bayesianRating: 6},
				{Movie: movies.Movie{ID: 4}, bayesianRating: 4},
			},
			wantID: []int{2, 3, 1},
		},
		{
			name:  "replaces lowest-ranked item",
			limit: 3,
			items: []IndexRecordItem{
				{Movie: movies.Movie{ID: 1}, bayesianRating: 5},
				{Movie: movies.Movie{ID: 2}, bayesianRating: 7},
				{Movie: movies.Movie{ID: 3}, bayesianRating: 6},
				{Movie: movies.Movie{ID: 4}, bayesianRating: 8},
			},
			wantID: []int{4, 2, 3},
		},
		{
			name:  "higher ID wins rating tie",
			limit: 2,
			items: []IndexRecordItem{
				{Movie: movies.Movie{ID: 10}, bayesianRating: 7},
				{Movie: movies.Movie{ID: 20}, bayesianRating: 7},
				{Movie: movies.Movie{ID: 30}, bayesianRating: 7},
			},
			wantID: []int{30, 20},
		},
		{
			name:  "lower ID loses rating tie",
			limit: 2,
			items: []IndexRecordItem{
				{Movie: movies.Movie{ID: 20}, bayesianRating: 7},
				{Movie: movies.Movie{ID: 30}, bayesianRating: 7},
				{Movie: movies.Movie{ID: 10}, bayesianRating: 7},
			},
			wantID: []int{30, 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			heap := newMovieHeap(SearchParams{limit: tt.limit, normalizedQuery: "test"})
			for i := range tt.items {
				heap.add(&tt.items[i])
			}

			if got := movieIDs(heap.topKResults()); !slices.Equal(got, tt.wantID) {
				t.Errorf("topKResults IDs = %v, want %v", got, tt.wantID)
			}
			if got := cap(heap.items); got != tt.limit {
				t.Errorf("heap capacity = %d, want %d", got, tt.limit)
			}
		})
	}
}
