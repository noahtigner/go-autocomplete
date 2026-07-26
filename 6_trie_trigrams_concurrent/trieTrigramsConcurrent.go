package trieTrigramsConcurrent

import (
	"slices"
	"strings"
	"sync"

	dataStructures "github.com/noahtigner/go-autocomplete/data_structures"
	"github.com/noahtigner/go-autocomplete/models"
)

// This approach supports partial matches for words in any order

type TrieNode struct {
	Pointers               map[string]*TrieNode
	PhrasesTerminatingHere []string
}

type Trie struct {
	root TrieNode
}

func NewTrie() *Trie {
	return &Trie{
		root: TrieNode{
			Pointers:               map[string]*TrieNode{},
			PhrasesTerminatingHere: make([]string, 0),
		},
	}
}

func (trie *Trie) Insert(phrase string, prefix string) {
	chars := strings.Split(prefix, "")
	node := &trie.root
	for _, char := range chars {
		if _, includes := node.Pointers[char]; !includes {
			node.Pointers[char] = &TrieNode{
				Pointers:               map[string]*TrieNode{},
				PhrasesTerminatingHere: make([]string, 0),
			}
		}
		node = node.Pointers[char]
	}
	node.PhrasesTerminatingHere = append(node.PhrasesTerminatingHere, phrase)
}

func (node *TrieNode) findPhrasesIncludingPrefix(matches []string) []string {
	matches = append(matches, node.PhrasesTerminatingHere...)

	// deterministic/stable iteration order
	keys := make([]string, 0, len(node.Pointers))
	for key := range node.Pointers {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	// recursive calls for each child
	for _, key := range keys {
		child := node.Pointers[key]
		matches = child.findPhrasesIncludingPrefix(matches)
	}

	return matches
}

func (trie *Trie) Search(phrase string) []string {
	chars := strings.Split(phrase, "")
	node := &trie.root
	for _, char := range chars {
		if _, includes := node.Pointers[char]; includes == false {
			return []string{}
		}
		node = node.Pointers[char]
	}
	return node.findPhrasesIncludingPrefix(nil)
}

func getTrigrams(word string) []string {
	if len(word) < 4 {
		return []string{word}
	}
	trigrams := make([]string, len(word)-2)
	for i := range len(word) - 2 {
		trigrams[i] = word[i : i+3]
	}
	return trigrams
}

func Search(products []models.Product, query string) []string {
	trie := NewTrie()

	// insert each word of each product into the Trie
	for _, product := range products {
		for word := range strings.FieldsSeq(product.Name) {
			for _, trigram := range getTrigrams(word) {
				trie.Insert(product.Name, trigram)
			}
		}
	}

	queryWords := strings.Fields(query)

	if len(queryWords) == 0 {
		return []string{}
	}

	wordResults := make([]dataStructures.Set[string], len(queryWords))
	var wg sync.WaitGroup

	for i, word := range queryWords {
		wg.Go(func() {
			var trigramSets []dataStructures.Set[string]

			for _, trigram := range getTrigrams(word) {
				set := dataStructures.NewSet[string]()
				for _, result := range trie.Search(trigram) {
					set.Add(result)
				}
				trigramSets = append(trigramSets, set)
			}

			wordResults[i] = trigramSets[0].Intersection(trigramSets[1:])
		})
	}

	wg.Wait()

	intersection := wordResults[0].Intersection(wordResults[1:])
	intersected_results := make([]string, 0, len(intersection))
	for key := range intersection {
		intersected_results = append(intersected_results, key)
	}
	return intersected_results
}
