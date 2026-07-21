package main

import (
	"encoding/json"
	"fmt"
	"os"

	// trie "github.com/noahtigner/go-autocomplete/1_trie"
	// trieOrdered "github.com/noahtigner/go-autocomplete/2_trie_ordered"
	trieWholeWordsAnyPosition "github.com/noahtigner/go-autocomplete/3_trie_whole_words_any_position"
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
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("Error: missing 1 positional argument for the search term")
		os.Exit(1)
	}

	query := args[0]

	products, err := readFileIntoMemory("./data/products.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	matches := trieWholeWordsAnyPosition.Search(products, query)

	for _, match := range matches {
		fmt.Println(match)
	}
}
