# Bitmap Unigram Search

## Goal

Replace legacy unigram postings with ASCII unigram bitmaps addressed by dense record slots. The change targets allocation and latency for one-character queries while preserving old-search results.

## Environment

| Item | Value |
| --- | --- |
| Date | 2026-08-12 |
| Go | 1.25.1 or newer, as required by `go.mod` |
| OS | macOS |
| CPU | Apple M2 Max |
| Workload | Deterministic 100,000-record benchmark corpus |
| Samples | 10 per benchmark |
| Duration | 3 seconds per sample |

This historical comparison used a temporary legacy implementation for the baseline and the bitmap implementation for the candidate. Both runs used identical commands and workloads. The legacy path has since been removed; use the repository benchmark workflow for future comparisons.

## Commands

```bash
go test ./autocomplete -run '^$' \
  -bench '^(BenchmarkBuildIndex100K|BenchmarkSearchIndex100K|BenchmarkEndToEndSearch100K)$' \
  -benchmem -count=10 -benchtime=3s > data/bench-old.txt

go test ./autocomplete -run '^$' \
  -bench '^(BenchmarkBuildIndex100K|BenchmarkSearchIndex100K|BenchmarkEndToEndSearch100K)$' \
  -benchmem -count=10 -benchtime=3s > data/bench-new.txt

go run golang.org/x/perf/cmd/benchstat@latest data/bench-old.txt data/bench-new.txt
```

## Results

Negative deltas favor the bitmap strategy. Reported results are statistically significant unless marked otherwise.

| Benchmark | Time | Memory | Allocations |
| --- | ---: | ---: | ---: |
| Build index, 100K records | -4.44% | -14.64% | -21.90% |
| Common unigram query | -89.83% | -99.97% | -98.70% |
| Rare unigram query | -68.59% | -99.88% | -95.42% |
| Unigram intersection | -93.50% | -99.99% | -99.40% |
| Empty unigram intersection | -100.00% | -100.00% | -99.63% |
| Mixed short and long query | -29.48% | -43.23% | -2.81% |
| Common bigram query | +10.01% | no material change | no material change |
| Common trigram query | +4.66% | no material change | no material change |
| End-to-end, 100K records | -2.07% | -15.89% | -21.64% |

## Interpretation

Bitmap intersections remove large temporary posting sets for one-character queries, which accounts for the large unigram allocation and latency reductions. Mixed queries also benefit from bitmap membership checks.

Bigram- and trigram-only searches do not use the bitmap path and regressed by roughly 5% to 10% in this run. The end-to-end benchmark remains faster overall because index construction and unigram work require less memory and fewer allocations.

Raw results from this comparison were intentionally left out of version control. The commands are retained as historical provenance; the legacy baseline cannot be reproduced from the current tree.
