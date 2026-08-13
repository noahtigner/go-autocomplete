# Go Autocomplete

This project explores autocomplete and substring-search implementations in Go. The current dataset is derived from IMDb title metadata and ratings rather than a synthetic product catalog.

The active implementation builds a concurrent inverted n-gram index over approximately 12 million movie, television, and video titles. It supports multi-word, case-insensitive substring matching and returns a bounded set of full movie records ordered by a Bayesian rating score.

## Progression

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
16. The above, plus time & memory optimizations.

Earlier implementations are preserved in Git history rather than in the working tree.

## Current Architecture

### Data pipeline

The ETL program downloads two official IMDb datasets:

- `title.basics.tsv.gz`
- `title.ratings.tsv.gz`

It joins records by `tconst`, converts IMDb's TSV records into `models.Movie`, and writes one JSON object per line to `data/movies.jsonl`.

The importer streams both compressed input files and streams JSONL output. The generated files are ignored by Git because they are large and IMDb data is refreshed regularly.

### Index construction

The application reads `data/movies.jsonl` one record at a time. A producer:

- Stores each complete movie record by ID alongside a precomputed Bayesian rating.
- Normalizes its primary title.
- Sends a small indexing job to each n-gram worker.

Worker 1 builds one bitmap for each ASCII byte. Each bit is addressed by a dense record slot, allowing one-character query terms to be intersected efficiently with bitwise operations. Workers 2 and 3 build deduplicated inverted posting lists for byte bigrams and trigrams, keyed by raw movie ID.

This avoids concurrent map writes without putting locks on the indexing hot path.

### Search

ASCII one-character query terms use bitmap intersection directly. Two- and three-byte terms use bigram and trigram posting lists; longer terms intersect their trigrams to retrieve candidates, then verify the complete term with case-insensitive substring matching. Mixed queries first retrieve multigram candidates and filter them through the relevant unigram bitmaps. Query words may appear in any order.

Search returns a `SearchResult` containing the total number of matching records and up to the requested number of full `models.Movie` values. Results are selected with a fixed-size min-heap and sorted by the Bayesian rating score precomputed during indexing. Movie ID is used as the descending tie-breaker. Empty queries and negative limits return an error.

The Bayesian score combines:

- The IMDb average rating.
- The number of IMDb votes.
- A global-average prior of `6.5`.
- A minimum-vote prior of `1,000`.

Titles without a ratings record use `NumVotes: 0` and no average rating, causing them to rank below rated titles.

## Setup

The project requires Go 1.25.1 or newer, as specified in `go.mod`.

IMDb data is not checked into the repository. The ETL command downloads it locally.

## Generate Data

Run the ETL program from the repository root:

```bash
go run ./cmd/etl
```

This downloads the current IMDb title and ratings files and generates:

```text
data/movies.jsonl
```

The source downloads are stored locally as:

```text
data/title.basics.tsv.gz
data/title.ratings.tsv.gz
```

These files, along with the generated JSONL file, are ignored by Git.

IMDb provides these datasets for personal and non-commercial use subject to its terms. Review the [IMDb dataset terms](https://developer.imdb.com/non-commercial-datasets/) before using or redistributing the data.

## Run Search

After generating `data/movies.jsonl`, provide a query as the first argument:

```bash
go run . "Star Wars"
```

The application builds the index, searches the query, prints up to ten results, and reports processing and search timings.

Example output has the general form:

```text
Processed 12679821 records in 30.00s
        Star Wars: Episode V - The Empire Strikes Back (1980) [votes -> score]
Found 45 results in 0.10s
```

## Testing

The test suite can be run with:

```bash
go test ./...
```

Race conditions can be checked with:

```bash
go test -count=1 -race ./...
```

### Code Coverage

Per-package code coverage can be reported with:

```bash
go test -count=1 -cover ./...
```

Per-function code coverage can be reported with:

```bash
go tool cover -func=/tmp/go-autocomplete.cover
```

Executed and missed statements can be viewed in the browser with:

```bash
go tool cover -html=/tmp/go-autocomplete.cover
```

#### Benchmarks

Benchmark collection instructions and historical optimization records are in [docs/benchmarks](docs/benchmarks/README.md). The first record documents the [bitmap unigram optimization](docs/benchmarks/2026-08-bitmap-unigram-search.md).

## Known Limitations

- There is currently no minimum query-length requirement.
- One- and two-character queries can produce very large candidate sets.
- A precomputed cache of popular results for one- and two-character query words is planned.
- Candidate retrieval still materializes n-gram candidate sets before top-K ranking.
- Exact top-K ranking still verifies every candidate that survives n-gram retrieval.
