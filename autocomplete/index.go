package autocomplete

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
	sets "github.com/noahtigner/go-autocomplete/internal/sets"
)

// This approach supports substring matching across each word in the query
// It builds the different n-gram indexes concurrently and uses an optimized set intersection algorithm

type IndexRecordItem struct {
	movies.Movie
	bayesianRating float64
	slot           int // dense bitmap offset for unigram postings
}

type indexJob struct {
	id             int // raw IMDb ID for bigram/trigram postings
	slot           int // dense bitmap offset for unigram postings
	normalizedName string
}

type Index struct {
	unigramsOld map[string][]int // TODO: remove
	unigrams    [utf8.RuneSelf]*sets.BitSet
	bigrams     map[string][]int
	trigrams    map[string][]int

	records      map[int]*IndexRecordItem
	recordBySlot []*IndexRecordItem

	lastSeenUnigrams map[string]int // TODO: remove
	lastSeenBigrams  map[string]int
	lastSeenTrigrams map[string]int
}

var maxRecords = 13_000_000

func (idx *Index) clean() {
	idx.lastSeenUnigrams = nil
	idx.lastSeenBigrams = nil
	idx.lastSeenTrigrams = nil
}

func (idx Index) nIndex(n int) map[string][]int {
	switch n {
	case 1:
		return idx.unigramsOld
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

func (idx *Index) processRecordMetadata(movie *movies.Movie, i int) {
	recordItem := &IndexRecordItem{
		Movie:          *movie,
		bayesianRating: movie.BayesianRating(),
		slot:           i,
	}
	idx.records[movie.ID] = recordItem
	idx.recordBySlot = append(idx.recordBySlot, recordItem)
}

func (idx Index) processRecordMultigram(id int, normalizedName string, n int) {
	index := idx.nIndex(n)
	lastSeen := idx.lastSeenNIndex(n)

	for word := range strings.FieldsSeq(normalizedName) {
		for _, gram := range getNGrams(word, n) {
			// prevent duplicate records caused by repeated grams
			if previous, exists := lastSeen[gram]; exists && previous == id {
				continue
			}

			lastSeen[gram] = id
			index[gram] = append(index[gram], id)
		}
	}
}

func (idx *Index) processRecordUnigram(slot int, normalizedName string) {
	for word := range strings.FieldsSeq(normalizedName) {
		for i, _ := range word {
			char := word[i]
			isASCII := char < utf8.RuneSelf
			if !isASCII {
				continue
			}

			if exists := idx.unigrams[char]; exists == nil {
				idx.unigrams[char] = sets.NewBitSet(maxRecords)
			}
			idx.unigrams[char].Set(slot)
		}
	}
}

func (idx *Index) processRecord(job indexJob, n int) {
	if n == 1 {
		idx.processRecordUnigram(job.slot, job.normalizedName)
		idx.processRecordMultigram(job.id, job.normalizedName, 1) // TODO: remove
	} else {
		idx.processRecordMultigram(job.id, job.normalizedName, n)
	}
}

func BuildIndexFromRecordStream(fileName string) (Index, int, error) {
	index := Index{
		unigramsOld:      make(map[string][]int), // TODO: remove
		bigrams:          make(map[string][]int),
		trigrams:         make(map[string][]int),
		records:          make(map[int]*IndexRecordItem),
		lastSeenUnigrams: make(map[string]int), // TODO: remove
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
				index.processRecord(job, workerNum)
			}
		})
	}

	file, decoder, err := openJsonlFile(fileName)
	if err != nil {
		for _, jobChannel := range jobs {
			close(jobChannel)
		}
		wg.Wait()
		return Index{}, 0, err
	}
	defer file.Close()

	processedCount := 0
	for {
		var movie movies.Movie
		err = decoder.Decode(&movie)
		if err != nil {
			for _, jobChannel := range jobs {
				close(jobChannel)
			}
			if err == io.EOF {
				break
			}
			wg.Wait()
			return Index{}, 0, err
		}

		if processedCount == maxRecords {
			for _, jobChannel := range jobs {
				close(jobChannel)
			}
			wg.Wait()
			return Index{}, 0, fmt.Errorf("Maximum of %d records exceeded", maxRecords)
		}

		if _, exists := index.records[movie.ID]; exists {
			for _, jobChannel := range jobs {
				close(jobChannel)
			}
			wg.Wait()
			return Index{}, 0, fmt.Errorf("Duplicate record with id %d", movie.ID)
		}

		index.processRecordMetadata(&movie, processedCount)

		job := indexJob{
			id:             movie.ID,
			normalizedName: normalizeName(movie.PrimaryTitle),
			slot:           processedCount,
		}

		for _, jobChannel := range jobs {
			jobChannel <- job
		}
		processedCount++
	}

	wg.Wait()
	index.clean()

	return index, processedCount, nil
}
