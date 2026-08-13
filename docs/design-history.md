# Design History

The current engine began as a trie-based autocomplete exercise. Each iteration exposed a constraint in the prior approach and led toward the current in-memory substring-search design.

1. Character trie for prefix matching.
2. Ordered trie traversal for deterministic results.
3. Word-prefix matching at any position in a title.
4. Concurrent word-prefix searches.
5. Trigram trie for substring candidates.
6. Concurrent trigram searches.
7. Inverted index for word prefixes.
8. Inverted trigram index.
9. Inverted unigram, bigram, and trigram indexes.
10. Concurrent inverted n-gram indexing with candidate verification and set intersection.
11. IMDb ETL pipeline joining title basics with title ratings.
12. JSONL output and streaming record ingestion.
13. Three long-lived indexing workers with separate ownership for unigram bitmaps and bigram/trigram posting lists.
14. Full movie-record retention by ID and Bayesian rating-based result ordering.
15. Bounded top-K result selection with a fixed-size min-heap.
16. Bitmap unigram intersections and related time and memory optimizations.

Earlier implementations are preserved in Git history rather than in the working tree.
