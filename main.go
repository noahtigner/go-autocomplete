package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/noahtigner/go-autocomplete/autocomplete"
	"github.com/noahtigner/go-autocomplete/products"
)

func readFileIntoMemory(filename string) ([]products.Product, error) {
	bytes, err := os.ReadFile(filename)

	if err != nil {
		return nil, fmt.Errorf("Error: %v", err)
	}

	var products []products.Product

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
	fmt.Printf("Loaded %d records in %.2fs\n", len(products), ioDuration.Seconds())

	indexingStart := time.Now()
	index := autocomplete.BuildIndex(products)
	indexingDuration := time.Since(indexingStart)
	fmt.Printf("Indexed %d records in %.2fs\n", len(products), indexingDuration.Seconds())

	searchStart := time.Now()
	matches := index.Search(query)
	searchDuration := time.Since(searchStart)

	for _, match := range matches[:min(len(matches), 10)] {
		fmt.Printf("\t%s\n", match)
	}

	fmt.Printf("Found %d results in %.2fs\n", len(matches), searchDuration.Seconds())
}
