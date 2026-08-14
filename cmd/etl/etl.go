package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
)

type gzipFileCloser struct {
	file   *os.File
	reader *gzip.Reader
}

func (c gzipFileCloser) Close() (err error) {
	return errors.Join(c.reader.Close(), c.file.Close())
}

func writeAtomically(path string, write func(*os.File) error) (err error) {
	tempFile, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	committed := false
	closeAttempted := false
	defer func() {
		if !closeAttempted {
			closeAttempted = true
			err = errors.Join(err, tempFile.Close())
		}
		if !committed {
			err = errors.Join(err, os.Remove(tempPath))
		}
	}()

	if err := write(tempFile); err != nil {
		return err
	}

	closeAttempted = true
	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		return err
	}

	committed = true
	return nil
}

func downloadData(client *http.Client, fileUrl string, outputDir string) error {
	fileUrlParts := strings.Split(fileUrl, "/")
	fileName := fileUrlParts[len(fileUrlParts)-1]

	resp, err := client.Get(fileUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Received a non-200 response for %s: %s", fileUrl, resp.Status)
	}

	return writeAtomically(filepath.Join(outputDir, fileName), func(tempFile *os.File) error {
		_, err := io.Copy(tempFile, resp.Body)
		return err
	})
}

func openGzipScanner(path string) (*bufio.Scanner, io.Closer, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, nil, errors.Join(
			err,
			file.Close(),
		)
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	closer := gzipFileCloser{
		file:   file,
		reader: reader,
	}
	return scanner, closer, nil
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

func parseMovie(fields []string) (movies.Movie, error) {
	if len(fields) != 9 {
		return movies.Movie{}, fmt.Errorf("Encountered title record with the wrong number of fields")
	}

	id, err := parseIdField(fields[0])
	if err != nil {
		return movies.Movie{}, fmt.Errorf("Error parsing id field, %s\n", fields)
	}

	var year *int
	if fields[5] != "\\N" {
		value, err := strconv.Atoi(fields[5])
		if err != nil {
			return movies.Movie{}, fmt.Errorf("Error parsing startYear field, %s\n", fields)
		}
		year = &value
	}

	if fields[6] != "\\N" {
		endYear, err := strconv.Atoi(fields[6])
		if err != nil {
			return movies.Movie{}, fmt.Errorf("Error parsing endYear field")
		}
		if year == nil || endYear > *year {
			year = &endYear
		}
	}

	var runtimeMinutes *int
	if fields[7] != "\\N" {
		value, err := strconv.Atoi(fields[7])
		if err != nil {
			return movies.Movie{}, fmt.Errorf("Error parsing runtimeMinutes field. %s\n", fields)
		}
		runtimeMinutes = &value
	}

	movie := movies.Movie{
		ID:             id,
		TitleType:      fields[1],
		PrimaryTitle:   fields[2],
		OriginalTitle:  fields[3],
		IsAdult:        fields[4] == "1",
		Year:           year,
		RuntimeMinutes: runtimeMinutes,
		Genres:         fields[8],
		NumVotes:       0, // default
	}

	return movie, nil
}

func mergeTitleData(titleScanner, ratingScanner *bufio.Scanner, encoder *json.Encoder) (titleCount, ratedTitleCount int, err error) {
	// Read past the headers. The ratings file has fewer rows than the titles
	// file, so the two scanners cannot be advanced in lockstep.
	if !titleScanner.Scan() {
		if err := titleScanner.Err(); err != nil {
			return 0, 0, err
		}
		return 0, 0, fmt.Errorf("Title file is empty")
	}
	if !ratingScanner.Scan() {
		if err := ratingScanner.Err(); err != nil {
			return 0, 0, err
		}
		return 0, 0, fmt.Errorf("Ratings file is empty")
	}

	ratingFields, ratingExists, err := nextRating(ratingScanner)
	if err != nil {
		return 0, 0, err
	}

	for titleScanner.Scan() {
		titleFields := strings.Split(strings.TrimSuffix(titleScanner.Text(), "\r"), "\t")

		movie, err := parseMovie(titleFields)
		if err != nil {
			return 0, 0, err
		}

		titleCount++

		var ratingID int
		if ratingExists {
			ratingID, err = parseIdField(ratingFields[0])
			if err != nil {
				return 0, 0, err
			}
		}

		for ratingExists && ratingID < movie.ID {
			ratingFields, ratingExists, err = nextRating(ratingScanner)
			if err != nil {
				return 0, 0, err
			}
			if ratingExists {
				ratingID, err = parseIdField(ratingFields[0])
				if err != nil {
					return 0, 0, err
				}
			}
		}

		if ratingExists && ratingID == movie.ID {
			averageRating, err := strconv.ParseFloat(ratingFields[1], 64)
			if err != nil {
				return 0, 0, fmt.Errorf("Error parsing averageRating field")
			}

			numVotes, err := strconv.Atoi(ratingFields[2])
			if err != nil {
				return 0, 0, fmt.Errorf("Error parsing numVotes field")
			}

			movie.AverageRating = &averageRating
			movie.NumVotes = numVotes

			ratedTitleCount++
			ratingFields, ratingExists, err = nextRating(ratingScanner)
			if err != nil {
				return 0, 0, err
			}
		}

		if err := encoder.Encode(movie); err != nil {
			return 0, 0, err
		}
	}

	if err := titleScanner.Err(); err != nil {
		return 0, 0, err
	}

	return titleCount, ratedTitleCount, nil
}

type gzipScannerOpener func(string) (*bufio.Scanner, io.Closer, error)

func mergeDownloadedDataWithOpener(output io.Writer, titlePath string, ratingPath string, openScanner gzipScannerOpener) (titleCount int, ratedTitleCount int, err error) {
	titleScanner, titleCloser, err := openScanner(titlePath)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		err = errors.Join(err, titleCloser.Close())
	}()

	ratingScanner, ratingCloser, err := openScanner(ratingPath)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		err = errors.Join(err, ratingCloser.Close())
	}()

	return mergeTitleData(titleScanner, ratingScanner, json.NewEncoder(output))
}

func writeMoviesJSONL(outputPath string, titlePath string, ratingPath string) (titleCount int, ratedTitleCount int, err error) {
	return writeMoviesJSONLWithOpener(outputPath, titlePath, ratingPath, openGzipScanner)
}

func writeMoviesJSONLWithOpener(outputPath string, titlePath string, ratingPath string, openScanner gzipScannerOpener) (titleCount int, ratedTitleCount int, err error) {
	err = writeAtomically(outputPath, func(tempFile *os.File) error {
		writer := bufio.NewWriter(tempFile)
		titleCount, ratedTitleCount, err = mergeDownloadedDataWithOpener(writer, titlePath, ratingPath, openScanner)
		if err != nil {
			return err
		}
		return writer.Flush()
	})
	return titleCount, ratedTitleCount, err
}

func etl() (err error) {
	dataDir := "./data"
	dataSets := []string{
		"https://datasets.imdbws.com/title.basics.tsv.gz",
		"https://datasets.imdbws.com/title.ratings.tsv.gz",
	}
	var dlWaitGroup sync.WaitGroup
	dlErrChan := make(chan error, len(dataSets))

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for _, fileUrl := range dataSets {
		dlWaitGroup.Go(func() {
			if err := downloadData(client, fileUrl, dataDir); err != nil {
				dlErrChan <- err
			}
		})
	}

	dlWaitGroup.Wait()
	close(dlErrChan)
	for err := range dlErrChan {
		return err
	}

	titleCount, ratedTitleCount, err := writeMoviesJSONL(
		filepath.Join(dataDir, "movies.jsonl"),
		filepath.Join(dataDir, "title.basics.tsv.gz"),
		filepath.Join(dataDir, "title.ratings.tsv.gz"),
	)
	if err != nil {
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
