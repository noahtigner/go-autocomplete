package prefixTreeOrdered

import (
	"slices"
	"strings"
)

// This approach only works when the search term is at the start of the product name
// It handles consistent results ordering
// It also does not handle deduplication

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

func (trie *Trie) Insert(phrase string) {
	chars := strings.Split(phrase, "")
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
