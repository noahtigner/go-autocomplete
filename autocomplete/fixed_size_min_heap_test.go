package autocomplete

import (
	"testing"

	models "github.com/noahtigner/go-autocomplete/models"
)

func TestRanksLower(t *testing.T) {
	tests := []struct {
		item1    IndexRecordItem
		item2    IndexRecordItem
		want     bool
		testName string
	}{
		{IndexRecordItem{Movie: models.Movie{ID: 1}, bayesianRating: 7.5}, IndexRecordItem{Movie: models.Movie{ID: 1}, bayesianRating: 7.6}, true, "true if a.bayesianRating is lower"},
		{IndexRecordItem{Movie: models.Movie{ID: 1}, bayesianRating: 7.6}, IndexRecordItem{Movie: models.Movie{ID: 1}, bayesianRating: 7.5}, false, "false if a.bayesianRating is higher"},
		{IndexRecordItem{Movie: models.Movie{ID: 1}, bayesianRating: 7.5}, IndexRecordItem{Movie: models.Movie{ID: 2}, bayesianRating: 7.5}, true, "true if ratings are the same and a.ID is lower"},
		{IndexRecordItem{Movie: models.Movie{ID: 2}, bayesianRating: 7.5}, IndexRecordItem{Movie: models.Movie{ID: 1}, bayesianRating: 7.5}, false, "false if ratings are the same and a.ID is higher"},
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

// func TestFixedSizeMinHeap(t *testing.T) {
// 	index := buildFixtureIndex(t)
// }
