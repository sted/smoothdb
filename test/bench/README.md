# Benchmarks

End-to-end throughput, measured with [k6](https://k6.io) against smoothdb
builds and PostgREST on the same PostgreSQL database. Everything here needs a
local PostgreSQL (`PGURL`, default `postgresql://postgres@localhost:5432/postgres`),
`k6`, and `postgrest` in the PATH for the comparisons.

- `fixtures.sql` builds the `k6bench` database: small tables for the to-one
  embed, 10 000 authors and 200 000 books, an events table for the writes, two
  functions, and the roles `bench_auth` (the authenticator, NOINHERIT),
  `bench_anon` and `bench_user`. It is loaded on the first run, or with `RELOAD=1`.
- `scenarios.js` holds the k6 scenarios, one per run: reads, filters, embeds,
  counts, aggregates, json paths, large responses, CSV, RPC, writes, and two
  with a JWT.
- `run.sh` runs scenarios against several targets at once, alternating
  their order, and prints req/s and p95 per scenario.
- `regress.sh` is the regression check: the working tree against a base ref on
  a fixed set of scenarios, three rounds each, medians compared. It exits 1
  when a scenario loses more than 5%.

## When to run what

- **Every change on a request path, before it is merged:** `test/bench/regress.sh`
  (about 3 minutes). A flagged scenario is run again before drawing a
  conclusion: one that loses on every round is not noise.
- **Comparing with PostgREST, or between releases:**

      REFS="v0.10.2:none WORKTREE:claims" test/bench/run.sh

  The table gives each smoothdb build against PostgREST.

## Reading the numbers

The builds, PostgreSQL and k6 share one machine, so the numbers are relative:
compare targets of the same run, never runs of different days. With more
virtual users than pool connections (`VUS` > `POOL`) the requests queue on the
pool, and the latencies measure the queue. Anything else running on the
machine moves the numbers by more than the regression threshold.
