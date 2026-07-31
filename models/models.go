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
	NumVotes       *int     `json:"numVotes"`
}
