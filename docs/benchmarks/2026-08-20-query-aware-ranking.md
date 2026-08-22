# Query-Aware Ranking

## Goal

Measure the cost of retaining normalized titles and applying bounded exact-title and title-prefix relevance boosts during top-K selection. The candidate implementation also serves request-scoped searches through `GET /search`; the synthetic benchmarks measure the shared index and search package, not HTTP serialization.

The heap caches each admitted candidate's query-aware score. Once full, it skips title-relevance evaluation when a candidate's Bayesian rating cannot reach the current heap minimum even with the maximum exact-title boost.

## Environment

| Item | Value |
| --- | --- |
| Date | 2026-08-20 |
| Baseline | `f88c616` |
| Candidate | working tree based on `154ede7` |
| Go | 1.25.1 |
| OS | macOS 14.6.1 |
| CPU | Apple M2 Max |
| Workload | Deterministic 100,000-record benchmark corpus |
| Samples | 10 per benchmark |
| Duration | 3 seconds per sample |

The baseline was checked out into a separate worktree. Baseline and candidate were collected sequentially on the same machine. Raw output is transient and was stored outside the repository before comparison.

## Commands

```bash
# Run in the f88c616 worktree, then repeat in the candidate worktree.
go test ./autocomplete -run '^$' \
  -bench '^(BenchmarkBuildIndex100K|BenchmarkEndToEndSearch100K)$' \
  -benchmem -count=10 -benchtime=3s

go test ./autocomplete -run '^$' \
  -bench '^BenchmarkSearchIndex100K$/(common-unigram|common-bigram|common-trigram|mixed-short-long|multiword-case-insensitive)$' \
  -benchmem -count=10 -benchtime=3s

go run golang.org/x/perf/cmd/benchstat@latest bench-baseline.txt bench-query-aware-ranking.txt
```

The search selection also includes `common-short-multiword-case-insensitive`, because its benchmark name matches the `multiword-case-insensitive` expression.

## Results

Values are medians of ten samples. Positive deltas are slower or larger than the baseline.

| Benchmark | Baseline | Candidate | Delta |
| --- | ---: | ---: | ---: |
| Build index, 100K records | 237.6 ms | 240.9 ms | +1.41% |
| End-to-end, 100K records | 247.5 ms | 252.7 ms | +2.13% |
| Common ASCII unigram | 676.0 us | 901.7 us | +33.39% |
| Common bigram | 5.035 ms | 6.876 ms | +36.57% |
| Common trigram | 1.194 ms | 1.546 ms | +29.51% |
| Common short multiword | 6.830 ms | 8.074 ms | +18.21% |
| Mixed short and long query | 8.100 ms | 9.062 ms | +11.87% |
| Multiword case-insensitive query | 7.563 ms | 8.150 ms | +7.77% |

| Memory | Baseline | Candidate | Delta |
| --- | ---: | ---: | ---: |
| Build index, 100K records | 219.7 MiB | 221.2 MiB | +0.69% |
| End-to-end, 100K records | 226.3 MiB | 227.9 MiB | +0.67% |
| Common ASCII unigram | 1.367 KiB | 1.727 KiB | +26.29% |

All other selected search workloads had no material per-operation memory change. Search allocation counts rose by 11 allocations for bounded heaps that fill, due to query-specific ranked heap entries; index-build and end-to-end allocation counts were unchanged.

## Interpretation

Retaining normalized titles adds roughly 1.5 MiB per 100,000-record synthetic index. The top-K score bound prevents relevance evaluation for candidates that cannot enter the heap and avoids repeated prefix checks during heap maintenance, keeping end-to-end latency within 2.13% of the baseline.

Search-only high-cardinality workloads still regress by 10% to 39%, because candidates near the top-K threshold require query-aware scoring. The relevance boosts are therefore an intentional latency tradeoff, protected by ranking tests rather than presented as a universal performance improvement. The synthetic corpus does not measure HTTP request parsing, server throughput, or full-corpus resident memory; collect a separate server-process capture before making claims in those areas.
