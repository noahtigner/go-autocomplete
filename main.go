package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	// trie "github.com/noahtigner/go-autocomplete/1_trie"
	// trieOrdered "github.com/noahtigner/go-autocomplete/2_trie_ordered"
	// trieWordPrefixAnyPosition "github.com/noahtigner/go-autocomplete/3_trie_word_prefix_any_position"
	// trieWordPrefixAnyPositionConcurrent "github.com/noahtigner/go-autocomplete/4_trie_word_prefix_any_position_concurrent"
	// trieTrigrams "github.com/noahtigner/go-autocomplete/5_trie_trigrams"
	// trieTrigramsConcurrent "github.com/noahtigner/go-autocomplete/6_trie_trigrams_concurrent"
	// invertedIndexPrefixAnyPosition "github.com/noahtigner/go-autocomplete/7_inverted_index_prefix_any_position"
	// invertedIndexTrigrams "github.com/noahtigner/go-autocomplete/8_inverted_index_trigrams"
	// invertedIndexNGrams "github.com/noahtigner/go-autocomplete/9_inverted_index_ngrams"
	invertedIndexNGramsConcurrent "github.com/noahtigner/go-autocomplete/10_inverted_index_ngrams_concurrent"
	models "github.com/noahtigner/go-autocomplete/models"
)

func readFileIntoMemory(filename string) ([]models.Product, error) {
	bytes, err := os.ReadFile(filename)

	if err != nil {
		return nil, fmt.Errorf("Error: %v", err)
	}

	var products []models.Product

	err = json.Unmarshal(bytes, &products)

	if err != nil {
		return nil, fmt.Errorf("Error: %v", err)
	}

	return products, nil
}

func main() {
	ioStart := time.Now()

	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("Error: missing 1 positional argument for the search term")
		os.Exit(1)
	}

	query := args[0]
	if len(query) == 0 {
		fmt.Println("Error: the first positional argument must be a nonempty search term")
		os.Exit(1)
	}

	products, err := readFileIntoMemory("./data/products.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ioDuration := time.Since(ioStart)
	searchStart := time.Now()
	fmt.Printf("Loaded %d records in %.2fs\n", len(products), ioDuration.Seconds())

	matches := invertedIndexNGramsConcurrent.Search(products, query)
	searchDuration := time.Since(searchStart)

	for _, match := range matches[:min(len(matches), 10)] {
		fmt.Println(match)
	}

	fmt.Printf("Found %d results in %.2fs\n", len(matches), searchDuration.Seconds())
}
