package trie

import (
	"strings"

	"github.com/noahtigner/go-autocomplete/models"
)

// This approach only works when the search term is at the start of the product name
// It also does not handle deduplication or consistent result ordering

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

func (trieNode *TrieNode) findPhrasesIncludingPrefix(acc []string) []string {
	node := trieNode
	if len(node.PhrasesTerminatingHere) > 0 {
		acc = append(acc, node.PhrasesTerminatingHere...)
	}
	for _, nodePtr := range node.Pointers {
		if nodePtr != nil {
			acc = nodePtr.findPhrasesIncludingPrefix(acc)
		}
	}
	return acc
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
	for _, product := range products {
		trie.Insert(product.Name)
	}
	return trie.Search(query)
}
