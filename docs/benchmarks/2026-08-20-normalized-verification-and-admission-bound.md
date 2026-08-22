# Normalized Verification And Admission Bound

## Goal

Measure two follow-up optimizations to query-aware ranking:

- Verify long-query candidates against the normalized title retained in each index record, avoiding per-candidate `strings.ToLower` allocations.
- Skip prefix evaluation for candidates that can only enter the top K through an exact-title boost and are not exact matches.

This is an incremental comparison against `c368b1f`, which already includes query-aware ranking and its initial title-aware benchmarked cost.

## Environment

| Item | Value |
| --- | --- |
| Date | 2026-08-20 |
| Baseline | `c368b1f` |
| Candidate | staged working tree based on `c368b1f` |
| Go | 1.25.1 |
| OS | macOS 14.6.1 |
| CPU | Apple M2 Max |
| Workload | Deterministic 100,000-record benchmark corpus |
| Samples | 10 per benchmark |
| Duration | 3 seconds per sample |

The baseline was checked out into a separate worktree. Baseline and candidate were collected sequentially on the same machine. Raw output is transient and was stored outside the repository before comparison.

## Commands

```bash
# Run in the c368b1f worktree, then repeat in the candidate worktree.
go test ./autocomplete -run '^$' \
  -bench '^(BenchmarkBuildIndex100K|BenchmarkEndToEndSearch100K)$' \
  -benchmem -count=10 -benchtime=3s

go test ./autocomplete -run '^$' \
  -bench '^BenchmarkSearchIndex100K$/(common-unigram|common-bigram|common-trigram|mixed-short-long|multiword-case-insensitive)$' \
  -benchmem -count=10 -benchtime=3s

go run golang.org/x/perf/cmd/benchstat@latest bench-before.txt bench-after.txt
```

The search selection also includes `common-short-multiword-case-insensitive`, because its benchmark name matches the `multiword-case-insensitive` expression.

## Results

Values are medians of ten samples. Negative deltas are faster or smaller than the baseline. All reported deltas except common-unigram and index construction were statistically significant at `p < 0.05`.

| Benchmark | Baseline | Candidate | Delta |
| --- | ---: | ---: | ---: |
| Build index, 100K records | 240.6 ms | 242.3 ms | no material change |
| End-to-end, 100K records | 255.0 ms | 250.3 ms | -1.85% |
| Common ASCII unigram | 910.0 us | 892.1 us | no material change |
| Common bigram | 7.116 ms | 5.462 ms | -23.25% |
| Common trigram | 1.570 ms | 1.272 ms | -18.99% |
| Common short multiword | 8.279 ms | 7.357 ms | -11.14% |
| Mixed short and long query | 9.028 ms | 6.586 ms | -27.04% |
| Multiword case-insensitive query | 8.326 ms | 6.167 ms | -25.94% |

| Memory | Baseline | Candidate | Delta |
| --- | ---: | ---: | ---: |
| Build index, 100K records | 221.2 MiB | 221.2 MiB | no material change |
| End-to-end, 100K records | 227.9 MiB | 227.4 MiB | -0.20% |
| Mixed short and long query | 6.663 MiB | 6.206 MiB | -6.87% |
| Multiword case-insensitive query | 6.663 MiB | 6.205 MiB | -6.87% |

Long-query allocations fell sharply:

| Benchmark | Baseline | Candidate | Delta |
| --- | ---: | ---: | ---: |
| End-to-end, 100K records | 1.235M | 1.215M | -1.62% |
| Mixed short and long query | 20,816 | 816 | -96.08% |
| Multiword case-insensitive query | 20,801 | 801 | -96.15% |

Allocation counts for all other selected search workloads were unchanged.

## Interpretation

Reusing `normalizedTitle` removes title-normalization allocations during complete-term verification. That is the primary cause of the large allocation and latency reductions for the mixed and multiword long-query workloads.

The exact-only admission band avoids unnecessary prefix checks for candidates between the exact and prefix boost thresholds. The benchmark measures both changes together, so it does not attribute the short-query improvements to that bound alone. It introduces no material index-build or per-operation memory cost.
