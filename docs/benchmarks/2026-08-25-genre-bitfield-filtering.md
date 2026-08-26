# Genre Bitfield Filtering

## Goal

Measure index construction and query execution after genres became a persisted 32-bit mask and search gained repeatable OR genre filters. The deterministic corpus assigns `drama` to even records and `action` to odd records.

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

## Results

| Benchmark | Time | Memory | Allocations |
| --- | ---: | ---: | ---: |
| Build index, 100K records | 235.5 ms | 231.9 MB | 1,214,513 |
| Common unigram | 792.0 us | 1,768 B | 17 |
| Common unigram, `drama` | 744.2 us | 1,768 B | 17 |
| Common unigram, `drama`, limit 0 | 586.4 us | 72 B | 2 |
| Common unigram, `drama` or `action` | 831.3 us | 1,768 B | 17 |
| Common trigram | 1.175 ms | 1.18 MB | 163 |
| Common trigram, `drama` | 1.414 ms | 1.18 MB | 163 |
| End-to-end index plus search, 100K records | 245.3 ms | 238.5 MB | 1,215,328 |

The unfiltered zero-limit unigram path remains bitmap-popcount only at 178.5 us. A genre filter must visit title candidates to compute its filtered total, so its zero-limit query costs 586.4 us. Genre checks add no per-query allocations.

No directly comparable baseline is recorded for the filtered workloads because they are new, and the benchmark corpus now includes genre data rather than null genre fields. Future changes should compare against this record using the same corpus and command.
