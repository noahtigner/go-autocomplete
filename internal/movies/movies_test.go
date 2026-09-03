package movies

import (
	"encoding/json"
	"strings"
	"testing"
)

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

func TestGenreBitField(t *testing.T) {
	wantGenreOrder := [...]string{
		"action", "adult", "adventure", "animation", "biography", "comedy", "crime",
		"documentary", "drama", "family", "fantasy", "film-noir", "game-show", "history",
		"horror", "music", "musical", "mystery", "news", "reality-tv", "romance", "sci-fi",
		"short", "sport", "talk-show", "thriller", "war", "western",
	}
	if GenresByIndex != wantGenreOrder {
		t.Errorf("GenresByIndex = %v, want %v", GenresByIndex, wantGenreOrder)
	}
	if len(GenresByIndex) > genreBitFieldWidth {
		t.Fatalf("genre count = %d, exceeds bitfield width %d", len(GenresByIndex), genreBitFieldWidth)
	}

	for index, genre := range GenresByIndex {
		t.Run(genre, func(t *testing.T) {
			got, err := NewGenreBitField(genre)
			if err != nil {
				t.Fatal(err)
			}

			want := uint32(1) << index
			if uint32(got) != want {
				t.Errorf("%q mask = %032b, want %032b", genre, got, want)
			}
		})
	}

	t.Run("combines genres and ignores duplicates", func(t *testing.T) {
		got, err := NewGenreBitField("drama", "sci-fi", "drama")
		if err != nil {
			t.Fatal(err)
		}

		want := uint32(1<<8 | 1<<21)
		if uint32(got) != want {
			t.Errorf("mask = %032b, want %032b", got, want)
		}
	})

	t.Run("rejects unknown genre", func(t *testing.T) {
		if _, err := NewGenreBitField("unknown"); err == nil {
			t.Fatal("NewGenreBitField returned nil error")
		}
	})
}

func TestGenreBitFieldJSON(t *testing.T) {
	genres, err := NewGenreBitField("drama", "musical")
	if err != nil {
		t.Fatal(err)
	}
	titleType, err := NewTitleTypeEnum("movie")
	if err != nil {
		t.Fatal(err)
	}

	movie := Movie{ID: 1, TitleType: titleType, Genres: genres}
	encoded, err := json.Marshal(movie)
	if err != nil {
		t.Fatal(err)
	}

	const want = `{"id":1,"titleType":1,"primaryTitle":"","originalTitle":"","isAdult":false,"year":null,"runtimeMinutes":null,"genres":65792,"averageRating":null,"numVotes":0}`
	if string(encoded) != want {
		t.Errorf("JSON = %s, want %s", encoded, want)
	}

	var decoded Movie
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Genres != genres {
		t.Errorf("decoded genres = %032b, want %032b", decoded.Genres, genres)
	}
	if decoded.TitleType != titleType {
		t.Errorf("decoded title type = %d, want %d", decoded.TitleType, titleType)
	}

	var withoutGenres Movie
	if err := json.Unmarshal([]byte(`{"id":2}`), &withoutGenres); err != nil {
		t.Fatal(err)
	}
	if withoutGenres.Genres != 0 {
		t.Errorf("missing genres = %032b, want empty mask", withoutGenres.Genres)
	}

	var nullGenres Movie
	if err := json.Unmarshal([]byte(`{"id":3,"genres":null}`), &nullGenres); err != nil {
		t.Fatal(err)
	}
	if nullGenres.Genres != 0 {
		t.Errorf("null genres = %032b, want empty mask", nullGenres.Genres)
	}
}

func TestTitleTypeEnum(t *testing.T) {
	wantTitleTypeOrder := [...]string{
		"movie", "short", "tvepisode", "tvminiseries", "tvmovie", "tvpilot",
		"tvseries", "tvshort", "tvspecial", "video", "videogame",
	}
	if TitleTypesByIndex != wantTitleTypeOrder {
		t.Errorf("TitleTypesByIndex = %v, want %v", TitleTypesByIndex, wantTitleTypeOrder)
	}

	for index, titleType := range TitleTypesByIndex {
		t.Run(titleType, func(t *testing.T) {
			got, err := NewTitleTypeEnum(strings.ToUpper(titleType))
			if err != nil {
				t.Fatal(err)
			}
			want := TitleTypeEnum(index + 1)
			if got != want {
				t.Errorf("%q enum = %d, want %d", titleType, got, want)
			}
			if !got.IsValid() {
				t.Errorf("%q enum is invalid", titleType)
			}
		})
	}

	if unknownTitleType.IsValid() {
		t.Error("unknown title type is valid")
	}
	if TitleTypeEnum(len(TitleTypesByIndex) + 1).IsValid() {
		t.Error("out-of-range title type is valid")
	}
	if _, err := NewTitleTypeEnum("unknown"); err == nil {
		t.Fatal("NewTitleTypeEnum returned nil error for an unknown title type")
	}

	movie, err := NewTitleTypeEnum("movie")
	if err != nil {
		t.Fatal(err)
	}
	short, err := NewTitleTypeEnum("short")
	if err != nil {
		t.Fatal(err)
	}
	if !movie.HasIntersection(short, movie) {
		t.Error("movie does not match a filter containing movie")
	}
	if movie.HasIntersection(short) {
		t.Error("movie matches a filter without movie")
	}
}
