package autocomplete

import (
	"strings"
	"unicode/utf8"

	movies "github.com/noahtigner/go-autocomplete/internal/movies"
	sets "github.com/noahtigner/go-autocomplete/internal/sets"
)

type SearchParams struct {
	normalizedQuery      string
	normalizedQuerySlice []string
	limit                int
}

type SearchResult struct {
	Total  int
	Movies []movies.Movie
}

func retrieveSearchCandidateIds(reverseIndex Index, queryWords []string) sets.Set[int] {
	wordResults := make([]sets.Set[int], len(queryWords))

	for i, word := range queryWords {
		var gramSets []sets.Set[int]
		grams := sets.Unique(gramsForQueryWord(word))
		index := reverseIndex.multigramIndex(len(word))

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

func matchesAllQueryWords(normalizedTitle string, queryWords []string) bool {
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

func (reverseIndex *Index) searchAllQueryWordsUnigrams(query SearchParams) (*movieHeap, int) {
	uniqueQueryChars := strings.Join(sets.Unique(query.normalizedQuerySlice), "")
	candidateBitSets := make([]*sets.BitSet, 0)

	for i := range uniqueQueryChars {
		char := uniqueQueryChars[i]

		if reverseIndex.unigrams[char] == nil {
			return nil, 0
		}

		candidateBitSets = append(candidateBitSets, reverseIndex.unigrams[char])
	}

	topResults := newMovieHeap(query)

	var visit func(int)
	if query.limit > 0 {
		visit = func(slot int) {
			topResults.add(reverseIndex.recordBySlot[slot])
		}
	}

	totalMatches := sets.ForEachIntersection(candidateBitSets, visit)

	return topResults, totalMatches
}

func (reverseIndex *Index) searchAllQueryWordsMultigrams(query SearchParams, lookupWords []string, singleCharWords []string) (*movieHeap, int) {
	// Collect pointers to each single-char word's bitSet
	singleCharBitSets := make([]*sets.BitSet, len(singleCharWords))
	for i, word := range singleCharWords {
		bitSet := reverseIndex.unigrams[word[0]]

		if bitSet == nil {
			return nil, 0
		}

		singleCharBitSets[i] = bitSet
	}

	candidateIds := retrieveSearchCandidateIds(*reverseIndex, lookupWords)
	requiresVerification := queryWordsRequireVerification(lookupWords)
	totalMatches := 0
	topResults := newMovieHeap(query)

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
		if requiresVerification && !matchesAllQueryWords(record.normalizedTitle, lookupWords) {
			continue
		}

		totalMatches += 1
		if query.limit > 0 {
			topResults.add(record)
		}
	}

	return topResults, totalMatches
}

func (reverseIndex Index) Search(query SearchParams) SearchResult {
	wordCount := len(query.normalizedQuerySlice)

	lookupWords := make([]string, 0, wordCount)
	singleCharWords := make([]string, 0, wordCount)
	hasNonUnigram := false
	for _, word := range query.normalizedQuerySlice {
		if len(word) != 1 || word[0] >= utf8.RuneSelf {
			lookupWords = append(lookupWords, word)
			hasNonUnigram = true
		} else {
			singleCharWords = append(singleCharWords, word)
		}
	}

	var heap *movieHeap
	var count int
	if !hasNonUnigram {
		heap, count = reverseIndex.searchAllQueryWordsUnigrams(query)
	} else {
		heap, count = reverseIndex.searchAllQueryWordsMultigrams(query, lookupWords, singleCharWords)
	}

	if heap == nil {
		return SearchResult{
			Total:  0,
			Movies: []movies.Movie{},
		}
	}
	return SearchResult{
		Total:  count,
		Movies: heap.topKResults(),
	}
}
