# Go Autocomplete

This project explores several autocomplete implementations over a product catalog. The final implementation uses a concurrently built inverted n-gram index.

## Progression

1. Character trie for prefix matching.
2. Ordered trie traversal for deterministic results.
3. Word-prefix matching at any position in a product name.
4. Concurrent word-prefix searches.
5. Trigram trie for substring candidates.
6. Concurrent trigram searches.
7. Inverted index for word prefixes.
8. Inverted trigram index.
9. Inverted n-gram index.
10. Concurrent inverted n-gram index with product IDs, candidate verification, optimized set intersections, and sorted results.
11. The above, plus an etl script for 12M IMDB records, partial streaming implementation

The earlier implementations are preserved in Git history rather than in the working tree.

## Current Approach

The active implementation:

- Builds unigram, bigram, and trigram indexes concurrently.
- Matches multiple query words in any order.
- Uses n-grams to retrieve candidate products quickly.
- Verifies candidates with case-insensitive substring matching.
- Stores product IDs in postings instead of repeating product names.
- Returns sorted product names.

## Run

Provide a search query as the first argument:

```bash
go run . "red phone"
```

The program loads `data/products.json`, builds the index, and prints up to ten matches with timing information.
