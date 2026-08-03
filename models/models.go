package models

type Movie struct {
	ID             int      `json:"id"`
	TitleType      string   `json:"titleType"`
	PrimaryTitle   string   `json:"primaryTitle"`
	OriginalTitle  string   `json:"originalTitle"`
	IsAdult        bool     `json:"isAdult"`
	Year           *int     `json:"year"`
	RuntimeMinutes *int     `json:"runtimeMinutes"`
	Genres         string   `json:"genres"`
	AverageRating  *float64 `json:"averageRating"`
	NumVotes       int      `json:"numVotes"`
}

func (m Movie) BayesianRating() float64 {
	const globalAverage = 6.5 // TODO: tune this
	const minimumVotes = 1_000

	if m.AverageRating == nil {
		return 0
	}

	votes := float64(m.NumVotes)
	rating := *m.AverageRating

	if rating < 0 || rating > 10 || votes <= 0 {
		return 0
	}

	return (votes*rating + minimumVotes*globalAverage) / (votes + minimumVotes)
}
