package main

import (
	"encoding/json"
	"fmt"
	"os"

	prefixTree "github.com/noahtigner/go-autocomplete/1_prefix_tree"
)

type Product struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func readFileIntoMemory(filename string) ([]Product, error) {
	bytes, err := os.ReadFile(filename)

	if err != nil {
		return nil, fmt.Errorf("Error: %v", err)
	}

	var products []Product

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

	trie := prefixTree.NewTrie()

	for _, product := range products {
		trie.Insert(product.Name)
	}
	matches := trie.Search(query)

	for _, match := range matches {
		fmt.Println(match)
	}
}
