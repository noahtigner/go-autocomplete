package autocomplete

import (
	"fmt"
	"strings"
	"unicode/utf8"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
	sets "github.com/noahtigner/go-autocomplete/internal/sets"
)

type SearchResult struct {
	Total  int
	Movies []movies.Movie
}

func retrieveSearchCandidateIds(reverseIndex Index, queryWords []string) sets.Set[int] {
	wordResults := make([]sets.Set[int], len(queryWords))

	for i, word := range queryWords {
		var gramSets []sets.Set[int]
		grams := sets.Unique(gramsForQueryWord(word))
		index := reverseIndex.nIndex(len(word))

		for _, gram := range grams {
			gramSet := sets.NewSet[int]()
			for _, match := range index[gram] {
				gramSet.Add(match)
			}
			gramSets = append(gramSets, gramSet)
		}

		wordResults[i] = sets.Intersection(gramSets)
	}

	intersection := sets.Intersection(wordResults)
	return intersection
}

func matchesAllQueryWords(movieTitle string, queryWords []string) bool {
	normalizedTitle := normalizeName(movieTitle)
	for _, word := range queryWords {
		if !strings.Contains(normalizedTitle, word) {
			return false
		}
	}
	return true
}

func queryWordsRequireVerification(queryWords []string) bool {
	// The byte-based index stores complete query words up to trigrams.
	// If all query words are short, we can skip matchesAllQueryWords which massively reduces allocations
	for _, word := range queryWords {
		if len(word) > 3 {
			return true
		}
	}
	return false
}

func (reverseIndex Index) oldSearch(query string, limit int) (SearchResult, error) {
	if limit < 0 || limit > 100 {
		return SearchResult{}, fmt.Errorf("A limit between 0 and 100 is required")
	}

	normalizedQuery := strings.ToLower(query)
	queryWords := strings.Fields(normalizedQuery)

	if len(queryWords) == 0 {
		return SearchResult{}, fmt.Errorf("At least one query word is required")
	}

	candidateIds := retrieveSearchCandidateIds(reverseIndex, queryWords)
	requiresVerification := queryWordsRequireVerification(queryWords)

	totalMatches := 0
	topResults := newMovieHeap(limit)

	for candidateId := range candidateIds {
		record := reverseIndex.records[candidateId]

		if requiresVerification && !matchesAllQueryWords(record.PrimaryTitle, queryWords) {
			continue
		}

		totalMatches += 1
		topResults.add(record)
	}

	heapResults := topResults.topKResults()

	return SearchResult{
		Total:  totalMatches,
		Movies: heapResults,
	}, nil
}

func (reverseIndex *Index) searchAllQueryWordsUnigrams(queryWords []string, limit int) SearchResult {
	uniqueQueryChars := strings.Join(sets.Unique(queryWords), "")
	candidateBitSets := make([]*sets.BitSet, 0)

	for i := range uniqueQueryChars {
		char := uniqueQueryChars[i]

		if reverseIndex.unigrams[char] == nil {
			return SearchResult{
				Total:  0,
				Movies: []movies.Movie{},
			}
		}

		candidateBitSets = append(candidateBitSets, reverseIndex.unigrams[char])
	}

	topResults := newMovieHeap(limit)

	var visit func(int)
	if limit > 0 {
		visit = func(slot int) {
			topResults.add(reverseIndex.recordBySlot[slot])
		}
	}

	totalMatches := sets.ForEachIntersection(candidateBitSets, visit)

	heapResults := topResults.topKResults()

	return SearchResult{
		Total:  totalMatches,
		Movies: heapResults,
	}
}

func (reverseIndex *Index) searchAllQueryWordsMultigrams(lookupWords []string, singleCharWords []string, limit int) SearchResult {
	// Collect pointers to each single-char word's bitSet
	singleCharBitSets := make([]*sets.BitSet, len(singleCharWords))
	for i, word := range singleCharWords {
		bitSet := reverseIndex.unigrams[word[0]]

		if bitSet == nil {
			return SearchResult{
				Total:  0,
				Movies: []movies.Movie{},
			}
		}

		singleCharBitSets[i] = bitSet
	}

	candidateIds := retrieveSearchCandidateIds(*reverseIndex, lookupWords)
	requiresVerification := queryWordsRequireVerification(lookupWords)
	totalMatches := 0
	topResults := newMovieHeap(limit)

	// Assess each candidate
	for candidateId := range candidateIds {
		record := reverseIndex.records[candidateId]

		// Check the single-character words
		matchesSingleChars := true
		for _, bitSet := range singleCharBitSets {
			if !bitSet.Contains(record.slot) {
				matchesSingleChars = false
				break
			}
		}
		if !matchesSingleChars {
			continue
		}

		// Check the multi-character words
		if requiresVerification && !matchesAllQueryWords(record.PrimaryTitle, lookupWords) {
			continue
		}

		totalMatches += 1
		if limit > 0 {
			topResults.add(record)
		}
	}

	heapResults := topResults.topKResults()

	return SearchResult{
		Total:  totalMatches,
		Movies: heapResults,
	}
}

func (reverseIndex *Index) newSearch(query string, limit int) (SearchResult, error) {
	if limit < 0 || limit > 100 {
		return SearchResult{}, fmt.Errorf("A limit between 0 and 100 is required")
	}

	normalizedQuery := strings.ToLower(query)
	queryWords := strings.Fields(normalizedQuery)

	if len(queryWords) == 0 {
		return SearchResult{}, fmt.Errorf("At least one query word is required")
	}

	lookupWords := make([]string, 0, len(queryWords))
	singleCharWords := make([]string, 0, len(queryWords))
	hasNonUnigram := false
	for _, word := range queryWords {
		if len(word) != 1 || word[0] >= utf8.RuneSelf {
			lookupWords = append(lookupWords, word)
			hasNonUnigram = true
		} else {
			singleCharWords = append(singleCharWords, word)
		}
	}

	if !hasNonUnigram {
		return reverseIndex.searchAllQueryWordsUnigrams(queryWords, limit), nil
	}
	return reverseIndex.searchAllQueryWordsMultigrams(lookupWords, singleCharWords, limit), nil

}

func (reverseIndex Index) Search(query string, limit int) (SearchResult, error) {
	useNew := true
	if useNew {
		return reverseIndex.newSearch(query, limit)
	}
	return reverseIndex.oldSearch(query, limit)
}
