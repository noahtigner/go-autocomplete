package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	// trie "github.com/noahtigner/go-autocomplete/1_trie"
	// trieOrdered "github.com/noahtigner/go-autocomplete/2_trie_ordered"
	// trieWholeWordsAnyPosition "github.com/noahtigner/go-autocomplete/3_trie_whole_words_any_position"
	trieWholeWordsAnyPositionConcurrent "github.com/noahtigner/go-autocomplete/4_trie_whole_words_any_position_concurrent"
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
	start := time.Now()

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

	matches := trieWholeWordsAnyPositionConcurrent.Search(products, query)

	for _, match := range matches[:min(len(matches), 10)] {
		fmt.Println(match)
	}

	duration := time.Since(start)
	fmt.Printf("Found %d results in %s\n", len(matches), duration)
}
