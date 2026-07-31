package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	models "github.com/noahtigner/go-autocomplete/models"
)

func downloadData(fileUrl string) error {
	fileUrlParts := strings.Split(fileUrl, "/")
	fileName := fileUrlParts[len(fileUrlParts)-1]

	resp, err := http.Get(fileUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Received a non-200 response for %s: %s", fileUrl, resp.Status)
	}

	out, err := os.Create("./data/" + fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	return nil
}

func decompressData(gzipStream io.Reader) (io.ReadCloser, error) {
	reader, err := gzip.NewReader(gzipStream)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func openGzipScanner(path string) (*bufio.Scanner, io.Closer, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	readCloser, err := decompressData(file)
	if err != nil {
		return nil, nil, err
	}

	scanner := bufio.NewScanner(readCloser)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	return scanner, readCloser, nil
}

func nextRating(scanner *bufio.Scanner) ([]string, bool, error) {
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			return nil, false, fmt.Errorf("rating record has %d fields, expected 3: %q", len(fields), line)
		}

		return fields, true, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, false, err
	}

	return nil, false, nil
}

func parseIdField(s string) (int, error) {
	return strconv.Atoi(strings.ReplaceAll(s, "t", ""))
}

func parseMovie(fields []string) (models.Movie, error) {
	if len(fields) != 9 {
		return models.Movie{}, fmt.Errorf("Encountered title record with the wrong number of fields")
	}

	id, err := parseIdField(fields[0])
	if err != nil {
		return models.Movie{}, fmt.Errorf("Error parsing id field, %s\n", fields)
	}

	var year *int
	if fields[5] != "\\N" {
		value, err := strconv.Atoi(fields[5])
		if err != nil {
			return models.Movie{}, fmt.Errorf("Error parsing startYear field, %s\n", fields)
		}
		year = &value
	}

	if fields[6] != "\\N" {
		endYear, err := strconv.Atoi(fields[6])
		if err != nil {
			return models.Movie{}, fmt.Errorf("Error parsing endYear field")
		}
		if year == nil || endYear > *year {
			year = &endYear
		}
	}

	var runtimeMinutes *int
	if fields[7] != "\\N" {
		value, err := strconv.Atoi(fields[7])
		if err != nil {
			return models.Movie{}, fmt.Errorf("Error parsing runtimeMinutes field. %s\n", fields)
		}
		runtimeMinutes = &value
	}

	movie := models.Movie{
		ID:             id,
		TitleType:      fields[1],
		PrimaryTitle:   fields[2],
		OriginalTitle:  fields[3],
		IsAdult:        fields[4] == "1",
		Year:           year,
		RuntimeMinutes: runtimeMinutes,
		Genres:         fields[8],
	}

	return movie, nil
}

func etl() error {
	dataSets := []string{
		"https://datasets.imdbws.com/title.basics.tsv.gz",
		"https://datasets.imdbws.com/title.ratings.tsv.gz",
	}
	var dlWaitGroup sync.WaitGroup
	dlErrChan := make(chan error, len(dataSets))

	for _, fileUrl := range dataSets {
		dlWaitGroup.Go(func() {
			if err := downloadData(fileUrl); err != nil {
				dlErrChan <- err
			}
		})
	}

	dlWaitGroup.Wait()
	close(dlErrChan)
	for err := range dlErrChan {
		return err
	}

	output, err := os.Create("./data/movies.jsonl")
	if err != nil {
		return err
	}
	defer output.Close()

	writer := bufio.NewWriter(output)
	defer writer.Flush()

	encoder := json.NewEncoder(writer)

	titleScanner, titleCloser, err := openGzipScanner("./data/title.basics.tsv.gz")
	defer titleCloser.Close()

	ratingScanner, ratingCloser, err := openGzipScanner("./data/title.ratings.tsv.gz")
	defer ratingCloser.Close()

	// Read past the headers. The ratings file has fewer rows than the titles
	// file, so the two scanners cannot be advanced in lockstep.
	if !titleScanner.Scan() {
		if err := titleScanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("Title file is empty")
	}
	if !ratingScanner.Scan() {
		if err := ratingScanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("Ratings file is empty")
	}

	ratingFields, ratingExists, err := nextRating(ratingScanner)
	if err != nil {
		return err
	}

	titleCount := 0
	ratedTitleCount := 0

	for titleScanner.Scan() {
		titleFields := strings.Split(strings.TrimSuffix(titleScanner.Text(), "\r"), "\t")

		movie, err := parseMovie(titleFields)
		if err != nil {
			return err
		}

		titleCount++

		var ratingId int
		if ratingExists {
			ratingId, err = parseIdField(ratingFields[0])
			if err != nil {
				return err
			}
		}

		for ratingExists && ratingId < movie.ID {
			ratingFields, ratingExists, err = nextRating(ratingScanner)
			if err != nil {
				return err
			}
			if ratingExists {
				ratingId, err = parseIdField(ratingFields[0])
				if err != nil {
					return err
				}
			}
		}

		if ratingExists && ratingId == movie.ID {
			averageRating, err := strconv.ParseFloat(ratingFields[1], 64)
			if err != nil {
				return fmt.Errorf("Error parsing averageRating field")
			}

			numVotes, err := strconv.Atoi(ratingFields[2])
			if err != nil {
				return fmt.Errorf("Error parsing numVotes field")
			}

			movie.AverageRating = &averageRating
			movie.NumVotes = &numVotes

			ratedTitleCount++
			ratingFields, ratingExists, err = nextRating(ratingScanner)
			if err != nil {
				return err
			}
		}

		if err := encoder.Encode(movie); err != nil {
			return err
		}

	}

	if err := titleScanner.Err(); err != nil {
		return err
	}

	fmt.Printf("Processed %d titles, %d with ratings\n", titleCount, ratedTitleCount)
	return nil
}

func main() {
	err := etl()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
