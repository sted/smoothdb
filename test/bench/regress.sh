#!/bin/bash
# Performance regression check: the working tree against a base ref, on a
# fixed set of k6 scenarios that covers the request paths (reads, filters,
# embeds, counts, aggregates, serializers, RPC, writes, JWT). Each scenario
# runs ROUNDS times on both builds, in alternating order (run.sh), and the
# medians are compared. Exit status 1 when a scenario of the working tree
# loses more than THRESHOLD percent against the base.
#
#   test/bench/regress.sh                 # against main's HEAD
#   BASE=v0.10.2 test/bench/regress.sh
#   test/bench/regress.sh json_path       # some scenarios only
#
# Environment (and those of run.sh: POOL, PGURL, RELOAD):
#   BASE       git ref to compare with (default: HEAD)
#   MODE       SessionMode of both builds (default: claims)
#   ROUNDS     runs per scenario and build (default 3)
#   VUS        virtual users (default 10, the pool size: no queueing noise)
#   DURATION   measured time per run (default 5s)
#   THRESHOLD  loss in percent that fails the check (default 5)
#
# About 3 minutes with the defaults. Run it on a quiet machine: the builds,
# PostgreSQL and k6 share it, and anything else running moves the numbers by
# more than the threshold. A flagged scenario is worth a second run before a
# conclusion; one that loses on every round is not noise.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
BASE=${BASE:-HEAD}
MODE=${MODE:-claims}
ROUNDS=${ROUNDS:-3}
THRESHOLD=${THRESHOLD:-5}
export VUS=${VUS:-10} DURATION=${DURATION:-5s} WARMUP=${WARMUP:-1s} POSTGREST=0
OUT=$(mktemp -d "${TMPDIR:-/tmp}/smoothdb-regress.XXXXXX")
export BINDIR="$OUT"

scenarios=("$@")
if [ ${#scenarios[@]} -eq 0 ]; then
  scenarios=(pk_lookup page_100 filter_multi count_exact embed_to_one embed_to_many
             aggregate json_path big_response csv rpc_get rpc_count insert patch jwt_pk_lookup)
fi

base=$(git -C "$here/../.." rev-parse --short "$BASE")
echo "base $BASE ($base) against the working tree, SessionMode $MODE, $ROUNDS rounds of $DURATION at $VUS VUs"
for r in $(seq 1 "$ROUNDS"); do
  echo "round $r/$ROUNDS"
  OUT="$OUT/r$r" REFS="$BASE:$MODE WORKTREE:$MODE" "$here/run.sh" "${scenarios[@]}" >"$OUT/r$r.log" 2>&1 || {
    echo "run.sh failed, see $OUT/r$r.log" >&2
    exit 2
  }
done

python3 -I - "$OUT" "$ROUNDS" "smoothdb@$BASE:$MODE" "smoothdb@WORKTREE:$MODE" "$THRESHOLD" <<'EOF'
import csv, statistics, sys
out, rounds, base, cur, threshold = sys.argv[1], int(sys.argv[2]), sys.argv[3], sys.argv[4], float(sys.argv[5])
runs = {}
for r in range(1, rounds + 1):
    for row in csv.DictReader(open(f"{out}/r{r}/results.csv")):
        if row["rps"]:
            runs.setdefault((row["target"], row["scenario"]), []).append(float(row["rps"]))
        if row["failed"]:
            runs.setdefault(("failed", row["scenario"]), []).append(f"{row['target']} {row['failed']}")
failed = []
print(f"\n| scenario | base req/s | working tree req/s | change | runs (base / tree) |")
print("|---|---|---|---|---|")
for s in dict.fromkeys(k[1] for k in runs if k[0] != "failed"):
    b, c = runs.get((base, s), []), runs.get((cur, s), [])
    if not b or not c:
        print(f"| {s} | | | missing | |")
        failed.append(s)
        continue
    mb, mc = statistics.median(b), statistics.median(c)
    change = (mc / mb - 1) * 100
    flag = ""
    if change < -threshold:
        flag = " ⚠"
        failed.append(s)
    if ("failed", s) in runs:
        flag += " ⚠ errors: " + "; ".join(runs[("failed", s)])
        failed.append(s)
    spread = f"{min(b):.0f}–{max(b):.0f} / {min(c):.0f}–{max(c):.0f}"
    print(f"| {s} | {mb:.0f} | {mc:.0f} | {change:+.1f}%{flag} | {spread} |")
print()
if failed:
    print(f"REGRESSION (more than {threshold:.0f}% slower, or errors): {' '.join(dict.fromkeys(failed))}")
    print(f"results in {out}")
    sys.exit(1)
print(f"no scenario more than {threshold:.0f}% slower")
EOF
