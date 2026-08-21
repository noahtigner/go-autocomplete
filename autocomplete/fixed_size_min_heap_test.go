package autocomplete

import (
	"slices"
	"testing"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
)

func TestRanksLower(t *testing.T) {
	tests := []struct {
		item1    IndexRecordItem
		item2    IndexRecordItem
		want     bool
		testName string
	}{
		{IndexRecordItem{Movie: movies.Movie{ID: 1}, bayesianRating: 7.5}, IndexRecordItem{Movie: movies.Movie{ID: 1}, bayesianRating: 7.6}, true, "true if a.bayesianRating is lower"},
		{IndexRecordItem{Movie: movies.Movie{ID: 1}, bayesianRating: 7.6}, IndexRecordItem{Movie: movies.Movie{ID: 1}, bayesianRating: 7.5}, false, "false if a.bayesianRating is higher"},
		{IndexRecordItem{Movie: movies.Movie{ID: 1}, bayesianRating: 7.5}, IndexRecordItem{Movie: movies.Movie{ID: 2}, bayesianRating: 7.5}, true, "true if ratings are the same and a.ID is lower"},
		{IndexRecordItem{Movie: movies.Movie{ID: 2}, bayesianRating: 7.5}, IndexRecordItem{Movie: movies.Movie{ID: 1}, bayesianRating: 7.5}, false, "false if ratings are the same and a.ID is higher"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got := ranksLower(&tt.item1, &tt.item2)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
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
