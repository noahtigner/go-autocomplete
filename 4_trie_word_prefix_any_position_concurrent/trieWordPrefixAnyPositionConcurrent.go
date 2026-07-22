package trieWordPrefixAnyPositionConcurrent

import (
	"slices"
	"strings"
	"sync"

	dataStructures "github.com/noahtigner/go-autocomplete/data_structures"
	"github.com/noahtigner/go-autocomplete/models"
)

// This approach matches query prefixes at the beginning of words anywhere within the product name.
// Multiple query words may match words in any order.

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

func Search(products []models.Product, query string) []string {
	trie := NewTrie()

	// insert each word of each product into the Trie
	for _, product := range products {
		for word := range strings.FieldsSeq(product.Name) {
			trie.Insert(product.Name, word)
		}
	}

	// search for each word in the query and intersect them
	queryWords := strings.Fields(query)
	single_word_results := make([]dataStructures.Set[string], len(queryWords))
	var wg sync.WaitGroup

	for i, word := range queryWords {
		wg.Add(1)

		go func(i int, word string) {
			defer wg.Done()

			set := dataStructures.NewSet[string]()
			for _, result := range trie.Search(word) {
				set.Add(result)
			}
			single_word_results[i] = set
		}(i, word)

		wg.Wait()
	}

	intersection := single_word_results[0].Intersection(single_word_results[1:])
	intersected_results := make([]string, 0, len(intersection))
	for key := range intersection {
		intersected_results = append(intersected_results, key)
	}
	return intersected_results
}
