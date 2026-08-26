package movies

import (
	"fmt"

	"github.com/noahtigner/go-autocomplete/internal/sets"
)

type Movie struct {
	ID             int           `json:"id"`
	TitleType      string        `json:"titleType"`
	PrimaryTitle   string        `json:"primaryTitle"`
	OriginalTitle  string        `json:"originalTitle"`
	IsAdult        bool          `json:"isAdult"`
	Year           *int          `json:"year"`
	RuntimeMinutes *int          `json:"runtimeMinutes"`
	Genres         GenreBitField `json:"genres"`
	AverageRating  *float64      `json:"averageRating"`
	NumVotes       int           `json:"numVotes"`
}

func (m Movie) BayesianRating() float64 {
	const globalAverage = 6.5  // TODO tune this
	const minimumVotes = 1_000 // TODO tune this

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

var GenresByIndex = [...]string{
	"action",
	"adult",
	"adventure",
	"animation",
	"biography",
	"comedy",
	"crime",
	"documentary",
	"drama",
	"family",
	"fantasy",
	"film-noir",
	"game-show",
	"history",
	"horror",
	"music",
	"musical",
	"mystery",
	"news",
	"reality-tv",
	"romance",
	"sci-fi",
	"short",
	"sport",
	"talk-show",
	"thriller",
	"war",
	"western",
}

const genreBitFieldWidth = 32

// Keep this ordering stable: JSONL persists genre masks using these indexes.
var _ [genreBitFieldWidth - len(GenresByIndex)]struct{}

var Genres = func() map[string]int {
	moviesByGenre := make(map[string]int)

	for i, genre := range GenresByIndex {
		moviesByGenre[genre] = i
	}

	return moviesByGenre
}()

type GenreBitField sets.BitField

func NewGenreBitField(genres ...string) (GenreBitField, error) {
	var gbf GenreBitField

	for _, g := range genres {
		if g == "" {
			continue
		}

		err := gbf.SetGenre(g)
		if err != nil {
			return 0, err
		}
	}

	return gbf, nil
}

func (g *GenreBitField) SetGenre(genre string) error {
	genreIdx, has := Genres[genre]
	if !has {
		return fmt.Errorf("Unexpected genre %v", genre)
	}
	(*sets.BitField)(g).Set(genreIdx)
	return nil
}

func (g1 GenreBitField) HasIntersection(g2 GenreBitField) bool {
	return (sets.BitField)(g1).HasIntersection((sets.BitField)(g2))
}
