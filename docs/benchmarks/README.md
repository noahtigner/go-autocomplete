# Benchmarks

This directory contains a dated record for each material performance change. A record describes the change, collection environment, reproducible commands, measured results, and expected tradeoffs.

Do not edit an existing record with results from another run. Add a new dated record instead, so historical comparisons retain their original context.

Raw benchmark output is transient and is not checked in. Store it under `data/bench-*.txt`, then compare runs with `benchstat` before removing it.

## Collection

Run benchmarks on an otherwise idle machine. Record the Go version, operating system, CPU, corpus size, and benchmark flags in the report. Compare the current commit with a baseline using the same corpus, commands, and sample count.

```bash
go test ./autocomplete -run '^$' \
  -bench '^(BenchmarkBuildIndex100K|BenchmarkSearchIndex100K|BenchmarkEndToEndSearch100K)$' \
  -benchmem -count=10 -benchtime=3s > data/bench-baseline.txt

# Check out the commit being evaluated, then run the identical command.
go test ./autocomplete -run '^$' \
  -bench '^(BenchmarkBuildIndex100K|BenchmarkSearchIndex100K|BenchmarkEndToEndSearch100K)$' \
  -benchmem -count=10 -benchtime=3s > data/bench-candidate.txt

go run golang.org/x/perf/cmd/benchstat@latest data/bench-baseline.txt data/bench-candidate.txt
```

## Record Format

Each report should include:

1. Goal and implementation change.
2. Environment and workload details.
3. Exact collection and comparison commands.
4. A concise results table or selected `benchstat` output.
5. Interpretation, including meaningful regressions and limitations.
