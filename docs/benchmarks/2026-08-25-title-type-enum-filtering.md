# Title Type Enum Filtering

## Goal

Measure the replacement of persisted title-type strings with a `uint8` enum and the new repeatable OR title-type filter. Index benchmarks use an all-`movie` corpus for comparison with the genre-filter snapshot. Search benchmarks use a deterministic mix of `movie`, `short`, `tvSeries`, and `tvMovie` values.

## Environment

| Field | Value |
| --- | --- |
| Date | 2026-08-25 |
| Go | go1.25.1 |
| OS | darwin/arm64 |
| CPU | Apple M2 Max |
| Workload | Deterministic 100,000-record benchmark corpus |
| Samples | 3 |
| Duration | 1 second per sample |

## Command

```bash
go test ./autocomplete -run '^$' \
  -bench '^(BenchmarkBuildIndex100K|BenchmarkSearchIndex100K|BenchmarkEndToEndSearch100K)$' \
  -benchmem -count=3 -benchtime=1s
```

## Comparison

The 2026-08-25 genre-filter record is the baseline for the all-`movie` index workload.

| Benchmark | Genre filter baseline | Title type enum | Delta |
| --- | ---: | ---: | ---: |
| Build index, 100K records | 235.5 ms, 231.9 MB, 1,214,513 allocs | 237.5 ms, 227.9 MB, 1,114,511 allocs | +0.8%, -1.7%, -100,002 allocs |
| End-to-end index plus search, 100K records | 245.3 ms, 238.5 MB, 1,215,328 allocs | 243.8 ms, 234.5 MB, 1,115,327 allocs | -0.6%, -1.7%, -100,001 allocs |

The enum removes roughly one allocation per indexed record and 4 MB of total index-build allocations per 100,000 records.

## Title Type Filter Results

| Benchmark | Time | Memory | Allocations |
| --- | ---: | ---: | ---: |
| Common unigram, `type=movie` | 823.8 us | 1,640 B | 17 |
| Common unigram, `type=movie`, limit 0 | 585.2 us | 72 B | 2 |
| Common unigram, `type=movie` or `type=short` | 860.8 us | 1,640 B | 17 |
| Common unigram, `genre=drama` and `type=movie` | 821.6 us | 1,640 B | 17 |
| Common unigram, no matching title type | 586.4 us | 232 B | 3 |
| Common trigram, `type=movie` | 1.328 ms | 1.18 MB | 163 |

The unfiltered count-only unigram path remains bitmap-popcount only at 187.0 us. Title-type filtering, like genre filtering, must visit candidates to compute its filtered total. It adds no allocations to matching queries.
