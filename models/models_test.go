package models

import "testing"

func TestBayesianRating(t *testing.T) {
	float5dot0 := 5.0
	floatNeg1 := -1.0
	float10dot1 := 10.1

	tests := []struct {
		numVotes      int
		averageRating *float64
		want          float64
		testName      string
	}{
		// expected - insufficient info
		{0, nil, 0, "No NumVotes or AverageRating"},
		{99, nil, 0, "No AverageRating"},
		{0, &float5dot0, 0, "No NumVotes"},
		// unexpected - invalid info
		{-1, &float5dot0, 0, "Negative NumVotes"},
		{99, &floatNeg1, 0, "Negative AverageRating"},
		{99, &float10dot1, 0, "Excessive AverageRating"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			movie := Movie{
				AverageRating: tt.averageRating,
				NumVotes:      tt.numVotes,
			}

			got := movie.BayesianRating()

			if tt.want != got {
				t.Errorf("Want %f, got %f", tt.want, got)
			}
		})
	}

	t.Run("Happy path", func(t *testing.T) {
		movie := Movie{
			AverageRating: &float5dot0,
			NumVotes:      100,
		}

		got := movie.BayesianRating()

		if got < 0 || got > 10 {
			t.Errorf("Want float in range [0, 10], got %f", got)
		}
	})
}
