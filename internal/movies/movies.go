package movies

import (
	"fmt"
	"slices"
	"strings"

	"github.com/noahtigner/go-autocomplete/internal/sets"
)

type Movie struct {
	ID             int           `json:"id"`
	TitleType      TitleTypeEnum `json:"titleType"`
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

type TitleTypeEnum uint8

const unknownTitleType TitleTypeEnum = iota

var TitleTypesByIndex = [...]string{
	"movie",
	"short",
	"tvepisode",
	"tvminiseries",
	"tvmovie",
	"tvpilot",
	"tvseries",
	"tvshort",
	"tvspecial",
	"video",
	"videogame",
}

// Keep this ordering stable: JSONL persists title types as one-based indexes.
var TitleTypes = func() map[string]int {
	titleTypesByName := make(map[string]int)

	for i, t := range TitleTypesByIndex {
		titleTypesByName[t] = i + 1
	}

	return titleTypesByName
}()

func NewTitleTypeEnum(titleType string) (TitleTypeEnum, error) {
	var tt TitleTypeEnum

	if err := tt.SetTitleType(titleType); err != nil {
		return 0, err
	}
	return tt, nil
}

func (t *TitleTypeEnum) SetTitleType(titleType string) error {
	titleTypeIdx, has := TitleTypes[strings.ToLower(titleType)]
	if !has {
		return fmt.Errorf("Unexpected title type %v", titleType)
	}
	*t = TitleTypeEnum(titleTypeIdx)
	return nil
}

func (t TitleTypeEnum) IsValid() bool {
	return t != unknownTitleType && int(t) <= len(TitleTypesByIndex)
}

func (t TitleTypeEnum) HasIntersection(titleTypes ...TitleTypeEnum) bool {
	return slices.Contains(titleTypes, t)
}
