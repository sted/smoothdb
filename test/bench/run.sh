#!/bin/bash
# Comparison benchmark: the k6 scenarios of scenarios.js against one or more
# smoothdb builds and PostgREST, on the same database (fixtures.sql), with the
# same pool size. Every server runs for the whole benchmark, and each scenario
# runs on every target in turn, starting from a different target each time,
# so that the order and the drift of the machine do not favour one target.
# Prints req/s and p95 per scenario and target, and writes them to
# $OUT/results.csv and $OUT/results.md; k6's stderr (a sample of the failed
# requests) goes to $OUT/<target>-<scenario>.log.
#
#   test/bench/run.sh                      # every scenario
#   test/bench/run.sh pk_lookup count_exact
#   REFS="v0.10.2:none WORKTREE:none WORKTREE:role" test/bench/run.sh
#
# Environment:
#   REFS       smoothdb builds, space separated: a git ref, or WORKTREE for the
#              working tree as it is, optionally followed by :none, :role or
#              :claims for its SessionMode (default: "WORKTREE", SessionMode
#              "role", smoothdb's default)
#   POSTGREST  1 to include PostgREST (default 1), from $POSTGREST_BIN
#   VUS        concurrent virtual users (default 50)
#   DURATION   measured time per scenario and target (default 5s)
#   WARMUP     unmeasured time before it (default 1s)
#   POOL       connections per pool, both servers (default 10, PostgREST's)
#   PGURL      admin connection (default postgresql://postgres@localhost:5432/postgres)
#   RELOAD     1 to reload the fixtures even if the database exists
#   OUT        output directory (default a new one under $TMPDIR)
#   BINDIR     where the smoothdb builds go (default $OUT): set it to reuse
#              them across runs (regress.sh does); WORKTREE is rebuilt only
#              when its binary is missing
#
# Both servers log only errors; smoothdb runs with TransactionMode "none" (its
# default), PostgREST with its defaults.
#
# SessionMode "role" keeps a connection on each session, and the requests of
# one token (the anonymous ones among them) beyond the pool size wait for one
# of them: with VUS > POOL they wait until the load stops, or answer 504 after
# Database.PoolAcquisitionTimeout where that exists. Compare throughput with
# :none, or with VUS <= POOL.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../.." && pwd)
REFS=${REFS:-WORKTREE}
POSTGREST=${POSTGREST:-1}
POSTGREST_BIN=${POSTGREST_BIN:-postgrest}
VUS=${VUS:-50}
DURATION=${DURATION:-5s}
WARMUP=${WARMUP:-1s}
POOL=${POOL:-10}
PGURL=${PGURL:-postgresql://postgres@localhost:5432/postgres}
OUT=${OUT:-$(mktemp -d "${TMPDIR:-/tmp}/smoothdb-bench.XXXXXX")}
DB=k6bench
SECRET=k6-bench-secret-of-at-least-32-characters

mkdir -p "$OUT"
unset SMOOTHDB_JWT_SECRET SMOOTHDB_DATABASE_URL

scenarios=("$@")
if [ ${#scenarios[@]} -eq 0 ]; then
  scenarios=($(sed -n 's/^    \([a-z_0-9]*\): .*/\1/p' "$here/scenarios.js"))
fi
writes=" insert insert_repr bulk_insert_100 patch "

# --- database ---------------------------------------------------------------
dbadmin="${PGURL%/*}/$DB"
if [ "${RELOAD:-0}" = 1 ] || ! psql "$PGURL" -Atc "select 1 from pg_database where datname = '$DB'" | grep -q 1; then
  echo "loading fixtures into $DB"
  psql "$PGURL" -q -c "drop database if exists $DB" -c "create database $DB"
  psql "$dbadmin" -q -v ON_ERROR_STOP=1 -f "$here/fixtures.sql" >/dev/null
fi
dburl="postgresql://bench_auth@localhost:5432/$DB"

reset_writes() {
  psql "$dbadmin" -q -c "truncate bench.events restart identity" \
    -c "insert into bench.events (kind, payload) select 'seed', jsonb_build_object('n', i) from generate_series(1, 10000) i" \
    -c "vacuum analyze bench.events"
}

# JWT for bench_user, HS256 with $SECRET
token=$(python3 -I - "$SECRET" <<'EOF'
import base64, hashlib, hmac, json, sys
b = lambda d: base64.urlsafe_b64encode(d).rstrip(b"=").decode()
head = b(json.dumps({"alg": "HS256", "typ": "JWT"}).encode())
body = b(json.dumps({"role": "bench_user"}).encode())
sig = hmac.new(sys.argv[1].encode(), f"{head}.{body}".encode(), hashlib.sha256).digest()
print(f"{head}.{body}.{b(sig)}")
EOF
)

# --- servers ----------------------------------------------------------------
pids=()
stop_all() {
  for p in "${pids[@]:-}"; do
    [ -n "$p" ] && kill "$p" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap stop_all EXIT

wait_ready() { # base url
  for _ in $(seq 1 100); do
    if [ "$(curl -s -o /dev/null -w '%{http_code}' "$1/clients")" = 200 ]; then
      return 0
    fi
    sleep 0.1
  done
  echo "server at $1 did not become ready" >&2
  return 1
}

build_smoothdb() { # ref -> path of the binary
  local ref=$1 bin="${BINDIR:-$OUT}/smoothdb-${1//\//_}"
  [ -x "$bin" ] && { echo "$bin"; return; }
  if [ "$ref" = WORKTREE ]; then
    (cd "$repo" && go build -trimpath -o "$bin" .)
  else
    local src="$OUT/src-${ref//\//_}"
    git -C "$repo" worktree add -q --detach "$src" "$ref"
    (cd "$src" && go build -trimpath -o "$bin" .)
    git -C "$repo" worktree remove --force "$src"
  fi
  echo "$bin"
}

start_smoothdb() { # name binary port sessionmode
  local cfg="$OUT/$1.jsonc"
  cat > "$cfg" <<EOF
{
  "Address": "127.0.0.1:$3",
  "SessionMode": "$4",
  "AllowAnon": true,
  "JWTSecret": "$SECRET",
  "Database": {
    "URL": "$dburl",
    "AnonRole": "bench_anon",
    "SchemaSearchPath": ["bench"],
    "MinPoolConnections": $POOL,
    "MaxPoolConnections": $POOL,
    "AggregatesEnabled": true
  },
  "Logging": { "Level": "error", "FileLogging": false, "StdOut": true }
}
EOF
  (cd "$OUT" && exec "$2" -c "$cfg") > "$OUT/$1.server.log" 2>&1 &
  pids+=($!)
  wait_ready "http://127.0.0.1:$3/api/$DB"
}

start_postgrest() { # port
  PGRST_DB_URI="$dburl" PGRST_DB_SCHEMAS=bench PGRST_DB_ANON_ROLE=bench_anon \
  PGRST_JWT_SECRET="$SECRET" PGRST_DB_POOL=$POOL PGRST_DB_AGGREGATES_ENABLED=true \
  PGRST_SERVER_HOST=127.0.0.1 PGRST_SERVER_PORT=$1 PGRST_LOG_LEVEL=error \
    "$POSTGREST_BIN" > "$OUT/postgrest.server.log" 2>&1 &
  pids+=($!)
  wait_ready "http://127.0.0.1:$1"
}

targets=()
bases=()
port=4190
for spec in $REFS; do
  ref=${spec%%:*}
  mode=role
  [ "$spec" != "$ref" ] && mode=${spec#*:}
  bin=$(build_smoothdb "$ref")
  name="smoothdb@${ref//\//_}:$mode" # a name for files too: no slash
  start_smoothdb "$name" "$bin" $port "$mode"
  echo "$name: $(git -C "$repo" rev-parse --short "${ref/WORKTREE/HEAD}")$([ "$ref" = WORKTREE ] && echo ' + working tree')"
  targets+=("$name")
  bases+=("http://127.0.0.1:$port/api/$DB")
  port=$((port + 1))
done
if [ "$POSTGREST" = 1 ]; then
  start_postgrest 3190
  echo "postgrest: $("$POSTGREST_BIN" --version | head -1)"
  targets+=("postgrest")
  bases+=("http://127.0.0.1:3190")
fi

# --- runs -------------------------------------------------------------------
echo "target,scenario,rps,p50_ms,p95_ms,max_ms,ok,failed" > "$OUT/results.csv"
n=${#targets[@]}
i=0
for s in "${scenarios[@]}"; do
  for k in $(seq 0 $((n - 1))); do
    j=$(( (i + k) % n ))
    name=${targets[$j]}
    base=${bases[$j]}
    case "$writes" in *" $s "*) reset_writes ;; esac
    k6 run -q --vus "$VUS" --duration "$WARMUP" -e BASE="$base" -e SCENARIO="$s" \
      -e TOKEN="$token" -e SUMMARY=/dev/null "$here/scenarios.js" >/dev/null 2>&1 || true
    k6 run -q --vus "$VUS" --duration "$DURATION" -e BASE="$base" -e SCENARIO="$s" \
      -e TOKEN="$token" -e SUMMARY="$OUT/$name-$s.json" \
      "$here/scenarios.js" > "$OUT/$name-$s.log" 2>&1 || true
    python3 -I - "$OUT/$name-$s.json" "$name" "$s" >> "$OUT/results.csv" <<'EOF'
import json, sys
try:
    d = json.load(open(sys.argv[1]))
    failed = " ".join(f"{k}:{v}" for k, v in sorted(d.get("failed", {}).items()))
    print(f"{sys.argv[2]},{sys.argv[3]},{d['rps']:.0f},{d['p50']:.2f},{d['p95']:.2f},{d['max']:.0f},{d['ok']:.4f},{failed}")
except Exception:
    print(f"{sys.argv[2]},{sys.argv[3]},,,,,,error")
EOF
    tail -1 "$OUT/results.csv"
  done
  i=$((i + 1))
done

# --- table ------------------------------------------------------------------
python3 -I - "$OUT/results.csv" "${targets[@]}" > "$OUT/results.md" <<'EOF'
import csv, sys
rows = list(csv.DictReader(open(sys.argv[1])))
targets = sys.argv[2:]
by = {(r["target"], r["scenario"]): r for r in rows}
scen = list(dict.fromkeys(r["scenario"] for r in rows))
head = ["scenario"] + [f"{t} req/s (p95 ms)" for t in targets]
pg = "postgrest" in targets
others = [t for t in targets if t != "postgrest"]
if pg:
    head += [f"{t} / postgrest" for t in others]
if len(others) > 1:
    head += [f"{others[-1]} / {others[0]}"]
print("| " + " | ".join(head) + " |")
print("|" + "---|" * len(head))
rate = lambda t, s: float(by[(t, s)]["rps"]) if by.get((t, s)) and by[(t, s)]["rps"] else None
for s in scen:
    cells = [s]
    for t in targets:
        r = by.get((t, s))
        if not r or not r["rps"]:
            cells.append("error")
            continue
        flag = f" ⚠ {r['failed']}" if r["failed"] else ""
        if r["max_ms"] and float(r["max_ms"]) > 5000:
            flag += f" ⚠ max {float(r['max_ms']) / 1000:.1f}s"
        cells.append(f"{r['rps']} ({r['p95_ms']}){flag}")
    ratios = [(t, "postgrest") for t in others] if pg else []
    if len(others) > 1:
        ratios.append((others[-1], others[0]))
    for a, b in ratios:
        x, y = rate(a, s), rate(b, s)
        cells.append(f"{x / y:.2f}x" if x and y else "")
    print("| " + " | ".join(cells) + " |")
EOF
echo
cat "$OUT/results.md"
echo
echo "results in $OUT"
