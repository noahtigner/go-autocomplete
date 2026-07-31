package autocomplete

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	models "github.com/noahtigner/go-autocomplete/models"
)

// This approach supports substring matching across each word in the query
// It builds the different n-gram indexes concurrently and uses an optimized set intersection algorithm

type Index struct {
	unigrams map[string][]int
	bigrams  map[string][]int
	trigrams map[string][]int

	records map[int]*models.Movie

	lastSeenUnigrams map[string]int
	lastSeenBigrams  map[string]int
	lastSeenTrigrams map[string]int
}

type indexJob struct {
	id             int
	normalizedName string
}

func (idx Index) nIndex(n int) map[string][]int {
	switch n {
	case 1:
		return idx.unigrams
	case 2:
		return idx.bigrams
	default:
		return idx.trigrams
	}
}

func (idx Index) lastSeenNIndex(n int) map[string]int {
	switch n {
	case 1:
		return idx.lastSeenUnigrams
	case 2:
		return idx.lastSeenBigrams
	default:
		return idx.lastSeenTrigrams
	}
}

func getNGrams(word string, n int) []string {
	n = min(3, max(n, 1))
	if len(word) < n+1 {
		return []string{word}
	}
	grams := make([]string, len(word)-(n-1))
	for i := range len(word) - (n - 1) {
		grams[i] = word[i : i+n]
	}
	return grams
}

func gramsForQueryWord(word string) []string {
	return getNGrams(word, len(word))
}

func normalizeName(title string) string {
	return strings.ToLower(title)
}

func openJsonlFile(fileName string) (*os.File, *json.Decoder, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", fileName, err)
	}

	decoder := json.NewDecoder(bufio.NewReader(file))
	return file, decoder, nil
}

func (idx *Index) finalize() {
	idx.lastSeenUnigrams = nil
	idx.lastSeenBigrams = nil
	idx.lastSeenTrigrams = nil
}

func (idx Index) processRecordMetadata(movie *models.Movie) {
	idx.records[movie.ID] = movie
}

func (idx Index) processRecord(id int, normalizedName string, n int) {
	index := idx.nIndex(n)
	lastSeen := idx.lastSeenNIndex(n)

	for word := range strings.FieldsSeq(normalizedName) {
		for _, gram := range getNGrams(word, n) {
			// prevent duplicate products caused by repeated grams
			if previous, exists := lastSeen[gram]; exists && previous == id {
				continue
			}

			lastSeen[gram] = id
			index[gram] = append(index[gram], id)
		}
	}
}

func BuildIndexFromRecordStream(fileName string) (Index, int, error) {
	index := Index{
		unigrams:         make(map[string][]int),
		bigrams:          make(map[string][]int),
		trigrams:         make(map[string][]int),
		records:          make(map[int]*models.Movie),
		lastSeenUnigrams: make(map[string]int),
		lastSeenBigrams:  make(map[string]int),
		lastSeenTrigrams: make(map[string]int),
	}

	jobs := [3]chan indexJob{
		make(chan indexJob, 1024),
		make(chan indexJob, 1024),
		make(chan indexJob, 1024),
	}

	var wg sync.WaitGroup
	for workerNum := 1; workerNum <= 3; workerNum++ {
		wg.Go(func() {
			for job := range jobs[workerNum-1] {
				index.processRecord(job.id, job.normalizedName, workerNum)
			}
		})
	}

	file, decoder, err := openJsonlFile(fileName)
	if err != nil {
		return Index{}, 0, err
	}
	defer file.Close()

	processedCount := 0
	for {
		var movie models.Movie
		err = decoder.Decode(&movie)

		if err == io.EOF {
			for _, jobChannel := range jobs {
				close(jobChannel)
			}
			break
		}
		if err != nil {
			for _, jobChannel := range jobs {
				close(jobChannel)
			}
			return Index{}, 0, err
		}

		index.processRecordMetadata(&movie)

		job := indexJob{
			id:             movie.ID,
			normalizedName: normalizeName(movie.PrimaryTitle),
		}

		for _, jobChannel := range jobs {
			jobChannel <- job
		}
		processedCount++
	}

	wg.Wait()
	index.finalize()

	return index, processedCount, nil
}
