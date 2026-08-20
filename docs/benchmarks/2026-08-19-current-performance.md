# Current Performance Snapshot

## Goal

Capture reproducible current performance after separating raw search-parameter parsing from search execution. This is a baseline snapshot, not a comparison against a previous implementation.

## Environment

| Item | Value |
| --- | --- |
| Date | 2026-08-19 |
| Commit | `f88c616` |
| Go | 1.25.1 |
| OS | macOS 14.6.1 |
| CPU | Apple M2 Max |
| Synthetic workload | Deterministic 100,000-record benchmark corpus |
| Samples | 10 per benchmark |
| Duration | 3 seconds per sample |

## Commands

The full-corpus capture used a prebuilt executable so `/usr/bin/time` reports the engine process rather than the Go compiler and executable together.

```bash
go build -o /tmp/go-autocomplete .
/usr/bin/time -l /tmp/go-autocomplete "Star Wars"

go test ./autocomplete -run '^$' \
  -bench '^(BenchmarkBuildIndex100K|BenchmarkSearchIndex100K|BenchmarkEndToEndSearch100K)$' \
  -benchmem -count=10 -benchtime=3s
```

`BenchmarkSearchIndex100K` measures execution after `RawSearchParams` has been parsed into `SearchParams`. `BenchmarkEndToEndSearch100K` includes index construction and search execution, but not one-time query parsing.

## Full Corpus Result

The local `data/movies.jsonl` corpus contained 12,699,818 titles. A single `Star Wars` capture, using the default result limit of ten, produced:

| Measurement | Result |
| --- | ---: |
| Index build | 35.61 s |
| Search | 30 ms |
| Matches | 8,179 |
| Maximum RSS | 8.24 GiB |

## Synthetic Benchmark Results

Values are medians of the ten samples. Bytes per operation and allocations were stable across samples.

| Benchmark | Time | Memory | Allocations |
| --- | ---: | ---: | ---: |
| Build index, 100K records | 235.5 ms | 230.3 MB | 1,214,510 |
| Common ASCII unigram, limit 10 | 0.678 ms | 1.4 KB | 6 |
| Common bigram | 4.995 ms | 4.69 MB | 533 |
| Common trigram | 1.184 ms | 1.18 MB | 152 |
| Mixed short and long query | 7.906 ms | 6.99 MB | 20,805 |
| End-to-end, 100K records | 245.4 ms | 237.3 MB | 1,235,315 |

## Interpretation

The bitmap unigram path remains markedly cheaper than multigram searches, while mixed queries still pay for candidate-set materialization and complete-term verification. The historical bitmap comparison remains in [Bitmap Unigram Search](2026-08-bitmap-unigram-search.md); its percentage deltas should not be compared directly with this snapshot because the current benchmark excludes parsing from `Search` execution.
