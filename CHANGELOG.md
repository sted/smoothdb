# Change Log

## Unreleased

### Security
* **GET and HEAD run read-only** — a function that writes could be called with `GET /rpc/fn`, and any view or function reached by a `GET` could mutate: the method every proxy, prefetcher, crawler and cache treats as safe and replays freely fired side effects. Like PostgREST, `GET` and `HEAD` now run in a read-only transaction (`BEGIN READ ONLY` with a `TransactionMode`, the session setting `default_transaction_read_only` with the default `none`, switched per request on the connection), and so does a `POST` calling a `STABLE` or `IMMUTABLE` function, whose `provolatile` is now part of the schema cache. A write attempted in such a request fails inside PostgreSQL with SQLSTATE `25006`, now answered as `405 Method Not Allowed` with `Allow: POST`, and nothing is written. Volatility is not the gate: a `VOLATILE` function that only reads still answers `GET`. `DELETE`, `PATCH` and `PUT` on `/rpc/` answer `405` (with `Allow: GET, HEAD, POST`) instead of `404`, as PostgREST does.
* **Anonymous requests never run as the authenticator** — with `AllowAnon: true` and an empty `Database.AnonRole`, an unauthenticated request reached Postgres as the connecting (authenticator) role, skipping both the `SET ROLE` and the `request.jwt.claims` GUC, so every RLS policy keyed off `current_user` or `request.jwt.claims` evaluated in the wrong context and the request carried the pool's own privileges. Anonymous access with no configured anon role is now refused with `401 "Anonymous access is disabled"`, mirroring PostgREST (`db-anon-role` unset → PGRST302); with a configured anon role the `request.jwt.claims` GUC is now set (`{"role":"<anon>"}`) for anonymous requests too. Configure `Database.AnonRole` as an explicit non-superuser role, never the connecting role.

### Fixed
* **Empty and unbounded ranges** — `empty`, `[10,)`, `(,10)` and `(,)` in a range column crashed the serializer with an index-out-of-range panic, recovered as a 406 with an empty body: the decoder read both bound lengths unconditionally, while the wire value carries only the bounds its flags announce (none at all for `empty` and `(,)`). The bounds are now read as the flags say, `empty` prints as `"empty"`, and an unbounded side keeps its bracket as PostgreSQL prints it (`[10,)`, `(,10)`, `(,)`), in JSON and CSV alike.
* **Bulk insert with mismatched keys** — `POST /t` with `[{"id":1},{"id":2,"body":"y"}]` inserted two rows and discarded `"y"`, answering 201: the column list was taken from the first object, so a key missing there was dropped for every row and a key present only in a later object never reached the INSERT. Like PostgREST, an array whose objects do not share one key set is now refused with a 400 (`All object keys must match`). `?columns=` stays the explicit opt-in and now means what it says: the listed columns are inserted for every row, a key absent from an object as NULL (it used to get the column default, and a listed column absent from the first object was dropped even where later objects carried it), and the keys not listed are ignored. The INSERT column list is also emitted in alphabetical order, so the SQL text is stable across requests.
* **`order=` on writes** — `order` was parsed and dropped on `POST`, `PATCH` and `DELETE`, so a `Prefer: return=representation` came back in physical row order whatever the request asked. The mutation is now wrapped in the `_source` CTE and the outer select orders the representation, as PostgREST does since 13.0.0 (#3013): top-level and related orders, with the embedded `x.order=` already applied inside the embed. `limit`/`offset` on `PATCH`/`DELETE` keep being ignored — every matching row is written — which is PostgREST's behaviour since the same release dropped limited updates/deletes; the guard for "touch at most N rows" is `Prefer: max-affected` (not yet implemented).
* **Generated columns** — a `GENERATED ALWAYS AS … STORED` column (or `VIRTUAL`, the default kind on PostgreSQL 18) was introspected as an ordinary one, and writing it was answered with a 500. The ordinary way in is a `select=*` or CSV export posted back, which any import or seeding workflow does. `pg_attribute.attgenerated` is now read with the columns, so `$info` and the admin column listing report such a column with `"generated": "stored"` (or `"virtual"`), `"readonly": true` and no default, and a client can build a form or a code generator without discovering it by trial. The write itself is handled as PostgREST does: the column is not dropped from the statement, PostgreSQL refuses it and the refusal (`428C9`) is a 400 carrying PostgreSQL's message.
* **Filter values with dots, commas and quotes** — a filter value was cut at the second dot, or at the first comma, colon, parenthesis or quote, and the query ran on what was left: `?ver=eq.1.2.3` filtered on `1.2`, `?mail=eq.a.b@c.com` on `a.b@c`, `?name=eq.a,b` on `a`, `?name=eq.O'Brien` on `O`, answering 200 with the wrong rows. As in PostgREST, a top-level value now runs to the end of the query parameter; inside `in.()`, `any`/`all` lists and `and`/`or` trees it ends at the next separator unless quoted, and a quote is a quote only at the start of a value and when followed by a separator, otherwise it is an ordinary character (`in.(")` and `in.(Double"Quote"McGraw")` are the literal values, as in the PostgREST spec); a backslash escapes only inside quotes. `?col=eq.` (empty value) now matches the empty string, `?col=eq` without the delimiter is a 400 instead of a recovered panic, and the operand of `is` is checked whole (`is.null.x` is refused). Quoting a top-level value (`eq."a,b"`) keeps working: a smoothdb leniency, PostgREST takes those quotes literally.
* **Binary-format types** — `bytea`, `inet`, `cidr`, `macaddr`, `macaddr8`, `time`, `point` and the other geometric types, `bit`, `bit varying`, `"char"`, `tid`/`xid`/`cid`/`xid8`, `tsvector` inside arrays, `jsonb` inside arrays and composites, and arrays of all of these came out corrupted: pgx requests them in binary format, and the serializers copied the raw bytes through as if they were text (a `time` or an `inet` looked plausible and was wrong). The serializers now dispatch on the wire format of each column: a binary value is decoded by the existing switch and a binary value of any other type is refused with a `SerializeError` naming the type, never copied through; a text value is PostgreSQL's own output converted as `to_json` converts it, so every type without a binary decoder is now sent in text (`bytea` prints as the `\x` hex form, as PostgREST does) and needs no code of its own. Text-format arrays (`enum[]`, `bytea[]`, `inet[]`…) and composites, and their nesting, are parsed from their literals into JSON arrays and objects, which also fixes arrays of enums.
### Fixed
* **`via()` walks enumerated paths, not nodes** — the recursive CTE carried the whole row plus a growing key array and guarded cycles per path, so it materialised every simple path of the graph: exponential in depth, and `MaxRecursiveDepth` was no bound at all. On a layered DAG of 650 nodes and 1800 edges (out-degree 3, depth 12) a walk from the root built 797,161 CTE rows for 490 nodes and took 791 ms per request; the CTE now carries only `(node, depth)` and dedups with `UNION`, so a node is one row per depth it is reached at, the seed is never re-entered, the table is joined back onto the deduplicated nodes, and the same walk takes 1.1 ms (`BenchmarkViaLayeredDag` in `test/recursive`, which also checks the walk against a BFS: same node set, same shortest depth per node). The edge table is joined as a derived table of `(from, to)` pairs — both orientations for `via!both` — so each arm uses an index on the known node instead of an OR join that scanned every edge per row. A cycle now walks until the depth cap, one row per node per level at most (a `via!both` walk of a 650-node tree at the default cap of 100: 33 ms). Three visible changes: `__path` is a selectable pseudo-column (the array of keys from the seed to the row; on a `via` walk it restores the path-enumerating shape, documented as exponential); ordering a `via` walk by an unselected `__depth` no longer leaks `__depth` into the result; and a result filter on `__depth` in a `via` walk now sees the node's shortest depth — it used to run before the dedup, per path, so `__depth=gte.2` returned a node at depth 1 that was also reachable at depth 2.
* **Ranges with a quoted subtype** — a `daterange`, `tsrange` or `tstzrange` column (alone, in an array, in a composite, or returned by a function), and a custom range inside a composite, came out as a JSON string with the bounds' own quotes unescaped, `"["2024-01-01","2024-06-01")"` as raw bytes on the wire, which no JSON parser accepts; only `int4range`, `int8range` and `numrange`, whose bounds print bare, were valid. The builtin range types are now sent in text and returned as PostgreSQL's `range_out` string, exactly as PostgREST returns them: `"[2024-01-01,2024-06-01)"`, `"[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")"` (a bound is quoted only when it needs it, timestamps in PostgreSQL's text form, `tstzrange` in the session time zone), in JSON and CSV alike. The binary range decoder is gone with it: a range that still arrived in binary would be refused with a `SerializeError`, like every other type without a binary decoder. A composite with a field pgx does not know (a custom range, a domain) is no longer registered without that field, which made the server send it in binary with the field in a format the serializers could not decode.
* **Multirange columns** — any `int4multirange`, `int8multirange`, `nummultirange`, `datemultirange`, `tsmultirange` or `tstzmultirange` column (alone, in an array, in a composite, or returned by a function) failed with a 406 and an empty body: a recovered nil dereference (`malformed value from database`) in the released serializer, `no binary decoder for type int4multirange` since the wire-format dispatch above. pgx requests the builtin multiranges in binary (its multirange codec follows the range codec it captured before the ranges were re-registered as text-only), and the schema cache took a multirange for a range with no subtype, classifying by `typcategory`, which a multirange shares with its range. The builtin multiranges are now sent in text and returned as PostgreSQL's own multirange text in one JSON string, exactly as PostgREST (`to_json`) returns them — `"{[1,3),[5,7)}"`, `"{}"` for the empty one, not an array of ranges — in JSON and CSV alike; a custom multirange (`create type textrange as range …` creates `textmultirange`) already arrived in text. The schema cache tells a multirange (`typtype` `m`) from a range (`r`), so nothing can dereference a range subtype a multirange does not have.

### Added
* **Server version in the schema cache** — nothing in the server knew which PostgreSQL it was talking to. `SchemaInfo.ServerVersion` now carries `server_version_num` (160015 is 16.15), read with the rest of the cache and refreshed on every schema reload, and `ServerAtLeast(N)` gates a feature on it: a code path that needs SQL introduced in PostgreSQL N falls back to the older SQL or refuses the request with a 400 naming the required version, instead of letting Postgres raise a syntax error as a 500. Startup logs the version it connected to and, like PostgREST, refuses a server older than 14.0 (`MinServerVersion`) with an error naming both versions; every release below 14 has reached end of life.

## 0.8.3 - 2026-09-05

### Security
* **Schema owner quoting** — `CREATE SCHEMA … AUTHORIZATION <owner>` interpolated the owner verbatim; it is now quoted as an identifier like every other DDL name. Reachable only by a caller who already holds DDL privileges, but it was the last unquoted DDL sink from the last review.
* **Bearer tokens with no secret** — with `LoginMode: "none"` the server may run without a `JWTSecret`, and a bearer token was then verified against the empty HMAC key, which anyone can sign with. Such a token is now refused with 401.
* **`VerboseErrors` defaults to false** — database error hints and details help fingerprint the schema, so they are now opt-in. Deployments that relied on the old default must set `VerboseErrors: true` explicitly.
* **jq caps** — a single jq evaluation output is now capped at `JQ.MaxOutputBytes` (default 1 MiB), and a whole `POST /jq` batch runs under one wall-clock budget, `JQ.BatchTimeout` (default 2 s): items left when it is spent are answered with a per-item error instead of being evaluated, so 200 evals at the per-evaluation timeout can no longer hold a request for 50 seconds. Memory during an evaluation is still not accounted for (gojq has no allocation cap); bound the process with `GOMEMLIMIT` where that matters.
* **`/token` rate limit** — the unauthenticated login route now has a per-client-address token bucket, `LoginRateLimit` attempts per minute (default 30, 0 disables), answering `429` with `Retry-After` beyond that, and its body is capped by `RequestMaxBytes` like every other route (it sits outside the standard middleware and was previously capped only by the HTTP library's 1 MB reader).
* **Session cache** (`SessionMode` other than `none`, the default is `role`) — three defects fixed. A session was created before the token was verified and survived a failed request: the next request with the same token found it, skipped verification and ran with nil claims (a panic, answered 500) or, for a valid token naming a missing database, against the main database. A session hit never re-checked the token expiry, so an expired token kept working as long as one request every five seconds kept the session warm. The session map had no bound. Half-built and expired sessions are now dropped, expiry is checked on every hit, and beyond `MaxSessions` (default 10000) a request runs without a session.

### Added
* `SessionMode: "claims"` caches the verified claims like `role` but returns the database connection to the pool at the end of every request. With `role` and `TransactionMode: "none"` the connection stays attached to the session for about a second after each request, so pool demand tracks active sessions rather than in-flight queries (measured on CollHub: 40 users against a 20-connection pool pushed p95 latency to 2.6 s while the SQL ran in 0.4 ms). `claims` keeps the verification cache without that retention. The default stays `role`.

### Fixed
* **`RequestMaxBytes: 0`** did the opposite of its documentation: `MaxBytesReader(0)` fails on the first byte, so 0 blocked every request body instead of lifting the limit. A non-positive value now means unlimited, as the comment, the README and the startup warning already said.
* **`is.null` on a JSON path** — `?col->>k=is.null` (and `is.not_null`, `is.true`, `is.false`, `is.unknown`, `not.is.*`) reached Postgres as `col->>'k' IS $1` and failed with 42601 as a 500, because the keyword literal was bound as a parameter whenever the field carried a JSON path. The `IS` keywords now stay literal on JSON paths too; every other operator on a JSON path keeps binding its value, so `->>k=eq.null` still compares against the string, as PostgREST does. The operand of `is` is also checked when quoted: `is."foo"` (and `is."null"`) used to skip the keyword check and reach Postgres as `IS foo`; like PostgREST, it is now a 400.
* **Dotted source names** — a request whose table or function segment contains a dot (`GET /api/db/doc.derived.member_choices`) is now refused by the request parser with a 400 before any SQL runs. The name is schema-qualified and quoted by splitting on dots, so it reached Postgres as `"public"."doc"."derived"."member_choices"` and came back as a 500 (`42601`, improper qualified name) after a prepare, a query and a rollback per request — a misbehaving client on devtest produced 25,000 of them in four days. The error names the offending segment and points at the `Accept-Profile`/`Content-Profile` header for schema selection.

## 0.8.2 - 2026-08-08

### Security
* **`::cast` injection** — cast targets in `select` are validated at parse time and rejected with 400 otherwise; a cast can't be a bind parameter, so it was interpolated verbatim and could alter the SELECT (reachable by any role that can read).
* **DDL quoting** — quote the remaining `CREATE`/`ALTER DATABASE` identifiers, and whitelist grant/revoke verbs and object types instead of interpolating them.

### Added
* Shutdown now waits for in-flight requests to complete; the wait was previously hardcoded to 1 second, so every restart killed any request slower than that. The new `GracefulShutdownTimeout` config key (seconds, default 0 = wait until done) bounds the wait for deployments that want a hard cap below their supervisor's stop grace period. A second signal during the wait forces an immediate exit, and `Shutdown()` is now idempotent.
* `/ready` now also answers for the database, not just for the process: it takes a connection from the main pool and pings it, bounded at 2 seconds, and reports `503 {"status":"unavailable","reason":"database"}` when it cannot. An instance whose pool hands out no connection serves nothing, yet previously still reported ready, so no orchestrator ever replaced it. `/live` stays unconditional — a database outage must not get the process killed.
* `/ready` now reports `503 {"status":"draining"}` as soon as a graceful shutdown begins, while `/live` keeps answering 200 until the process exits — the standard probe contract for zero-downtime rolling deploys. The new `DrainDelay` config key (seconds, default 0 = disabled) keeps the listener serving for that long after readiness flips, giving load balancers time to deregister the instance before it stops accepting connections. The delay applies to SIGTERM only; an interactive Ctrl-C (SIGINT) shuts down immediately, and a second signal during the window skips it.

### Fixed
* **Mutations with an embed** — a write whose `select` carries an embedded resource failed with `42702` (ambiguous column). With embeds the `RETURNING` clause feeds the `_source` CTE, which the outer select reads by column name, so a formatted or repeated foreign-key column made those references ambiguous; each base column is now exposed once, raw, and a star subsumes them all.
* **PostgreSQL 18** — grants and constraints introspection work again on PG 17 and 18: the `MAINTAIN` privilege letter is accepted (and an unknown letter is now named in the error), and the constraints query skips `contype = 'n'` rows, which PG 18 materializes in `pg_constraint` for `NOT NULL`, so column introspection stays identical across versions.
* **Connection leak on a failed acquire** — a request that failed after taking a pool connection returned without giving it back, because the release runs only on the success path. The ordinary way in is a client hanging up while the role is being set, which fails `SET ROLE` with a canceled context. Each occurrence lost a pool slot for good and left the session marked in use, so after `MaxPoolConnections` such requests the instance served nothing at all: every further request waited in `Acquire` until its own deadline while Postgres sat idle.
* Schema cache held in an atomic pointer (was a data race on reload).
* Serializers return an error on a malformed wire buffer or a type/dimension mismatch instead of panicking or silently misparsing.
* `limit`/`offset` reject non-integer or negative values (was a silent `LIMIT 0`).
* `*`→`%` wildcard rewrite scoped to `like`/`ilike` (was applied to every value).
* Boolean-filter nesting capped at 100 levels (stack-overflow DoS).
* Invalid grant input returns 400 instead of 500.
* Session watcher no longer deadlocks when paused.
* **SIGTERM** — the server now shuts down gracefully on SIGTERM (the default stop signal of Docker, Kubernetes and systemd), not only on SIGINT/Ctrl-C. A raw SIGTERM previously terminated the process immediately, cutting in-flight requests and resetting database connections.
* **Exit status** — a graceful shutdown now exits with status 0; it previously exited 1, which supervisors interpret as a crash.
* **Database connections on shutdown** — connection pools and the notification listener's dedicated connection are now closed after the HTTP server drains, so Postgres sees clean disconnects instead of connection resets on every stop. `NotificationListener.Stop()` now interrupts a pending `WaitForNotification` and waits for the listener goroutine to exit before returning.

## 0.8.1 - 2026-07-08

### Security
* **SQL injection** — closed four sinks where request tokens were interpolated into SQL with bare quotes instead of being escaped/parameterized (a backslash escape in the parser let a client smuggle a literal `'`/`"`): the `select` alias, embedded-resource filter values, full-text-search config arguments, and JSON-path members. Top-level filter values and identifiers were already parameterized and unaffected.
* **`SET ROLE`** — the JWT `role` claim is now quoted as an identifier so it cannot break out of the statement.
* **Empty JWT secret** — the server now refuses to start when authentication is enabled and `JWTSecret` is empty (an empty HMAC key lets anyone forge tokens); previously only a warning. Debug mode auto-generates a random secret.
* **TLS fail-closed** — a configured certificate that fails to load is now a fatal startup error instead of silently serving plaintext HTTP.
* **Config file permissions** — written `0600` (was `0777`), since it holds the JWT secret and the database password.

## 0.8.0 - 2026-07-07

### Added
* Generic jq support via [gojq](https://github.com/itchyny/gojq), disabled by default (`JQ.Enabled`). Evaluation is bounded: no I/O, per-evaluation timeout (`JQ.Timeout`), program size cap (`JQ.MaxProgramBytes`), compiled-program LRU cache (`JQ.CacheEntries`), and an exactly-one-output convention. Three surfaces over one shared core (new `jqeval` package):
  * **jq updates**: `PATCH /table?filters&jq=<program>` performs an atomic read-modify-write of the matched rows — rows are selected `FOR UPDATE` inside the request transaction, each row is fed to the program, and its object output becomes that row's `UPDATE ... SET`. All-or-nothing, capped by `JQ.MaxUpdateRows`; RLS and triggers apply unchanged. The program can also be sent as the raw request body with `Content-Type: application/vnd.smoothdb.jq`
  * **Response transforms**: `jq=` on table reads and RPC calls transforms the JSON response body; `Content-Range`/count headers reflect the pre-transform result set
  * **`POST /jq`**: standalone batch evaluation endpoint with per-item errors and `parse_only` compile-checking for authoring-time validation
  * `jq_args=` (URL-encoded JSON object) binds values as jq variables on both updates and transforms

## 0.7.2 - 2026-06-30

### Added
* `recurse!up` operator for single-table recursion — walks the FK toward ancestors instead of descendants (e.g. `parent_id=recurse!up.all` returns a node's ancestor chain). Rejected with `via()`, where the same reversal is done by swapping the edge columns.

### Fixed
* Ordering a `via()` recursive query by `__depth` without selecting it no longer 500s with "ORDER BY expressions must appear in select list"; `__depth` is now surfaced in the projection and routed through the min-depth dedup wrapper.

## 0.7.1 - 2026-06-18

### Fixed
* Computed-relationship embeds (a function of the parent row, e.g. `labels_objects(document)`) now compose with single-table recursion — they previously failed with `column "<table>" does not exist` because the function's row argument wasn't repointed to the recursive CTE. Plain FK and spread embeds were unaffected.

## 0.7.0 - 2026-06-15

### Added
* `__depth` selectable pseudo-column on recursive queries — reports each row's traversal depth (seed = 0); in `via` mode it reports the shallowest depth at which a node is reached
* `walk.` filter prefix to prune the traversal — skips a non-matching node and everything beyond it
* Bidirectional graph traversal with `via!both(src,dst)` — follows edges in either direction
* Resource embedding now composes with single-table recursion (not supported together with `via()` traversal)

### Changed
* **Breaking:** plain filters on recursive queries now filter the *result* instead of pruning the traversal; the previous prune-the-walk behavior is now opt-in via the `walk.` prefix

## 0.6.1 - 2026-06-12

### Fixed
* JSONB filter behavior now matches PostgREST (#19): JSON-path filters with quoted string values, containment against JSON string arrays, and whole-column JSONB equality/inequality

### Improved
* Improved release process (goreleaser config and `make release` target)
* Updated Go and UI dependencies

## 0.6.0 - 2026-04-17

### Added
* Configurable JWT token expiry (`TokenExpiry` config, default 24h)
* Configurable error verbosity (`VerboseErrors` config, default true) to control whether database hints/details are returned to clients
* Security headers middleware (`X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`)
* Config startup validations: warn on empty JWT secret, reject CORS credentials with wildcard origins, warn on unlimited request body size

### Fixed
* Fixed parallel test suites port collision (postgrest suite moved to port 8084)
* SQL injection in `GetPolicies`, `GetDatabasePrivileges`, and `GetPrivileges` — switched from string concatenation to parameterized queries
* JWT algorithm validation now pinned to HS256 (was accepting any HMAC variant)
* CORS default changed from wildcard `*` to empty (must be explicitly configured)
* CORS allowed headers restricted to specific list instead of wildcard
* Admin `/sessions` endpoint now requires authentication
* TLS minimum version enforced to TLS 1.2
* Session keys now use SHA-256 hash instead of raw token strings
* Removed hardcoded JWT secret from sample config
* RPC calls now emit named parameters in a deterministic order (function-signature order, falling back to alphabetical), so `pg_stat_statements` aggregates identical calls under a single `queryid` instead of one per JSON key permutation
* M2M view relationship synthesis now deterministically prefers view junctions over base-table junctions (fixes a flaky "permission denied" error when the base junction lived in a schema the caller lacked USAGE on)

### Improved
* Session improvements and tests

## 0.5.0 - 2026-04-01

### Added
* Support for recursive queries
* Support for any/all operator modifiers on eq, like, ilike, gt, gte, lt, lte, match, imatch (e.g. `last_name=like(any).{O*,P*}`)
* Support for `is.not_null` filter value (and `not.is.not_null`, case-insensitive)
* Support for `isdistinct` operator (`IS DISTINCT FROM`)
* Support for full-text search on text and jsonb columns (auto `to_tsvector()` wrapping)
* Resource embedding with views and materialized views (including chained views, aliased columns, M2M through view junctions)


## 0.4.1 - 2026-03-18

### Added
* Support for setting and retrieving comments on tables and columns via the DDL Admin API

### Fixes
* Fixed incorrect native type casting for boolean and null values when filtering JSONB via `->` operator
* Fixed table creation in non-default schema returning 404
* Fixed owner change ordering in table creation and update (owner is now set last to avoid permission errors)


## 0.4.0 - 2026-02-23

### Added
* Support for Computed Relationships (table-returning functions as virtual relationships)
* Support filtering embedded resources by FK column name (e.g. `client_id.id=eq.1`)
* Related orders: order parent rows by columns in to-one embedded resources (e.g. `order=clients(name).desc`)

### Fixes
* Fixed serialization bugs for composite and unrecognized types
* Fixed CI race condition

## 0.3.0 - 2025-05-05

### Added
* Schema Cache Reload via PostgreSQL NOTIFY
* Route for listing columns in /api (eg /api/db/$info/table)
* Server version in the HTTP header
* CORS environment variables
* Health endpoints

## 0.2.12 - 2025-04-15

### Added
* Route for listing tables in /api
* Disambigue embedded resources by fk contraints; hints by fk and related cols; !left; tests
* Support comma-separated Prefer values

### Fixes
* Fix for 32bit arch
* Dependencies

## 0.2.11 - 2024-11-25

### Added
* Login route
* Env variable for JWT secret
* Support for Char and UUID types
* Flexible JWT claims

### Fixes
* Support quotes on columns and on_conflict fields
* Fixes on array serialization
* Fields in JSON struct for errors in lowercase

## 0.2.10 - 2024-10-10

### Fixes
* Possible loop in the parser with invalid select list
* Small refactoring for UI code
* Dependencies updates

## 0.2.9 - 2024-08-04

### Fixes
* Better docs for installing and some other minor fixes

## 0.2.8 - 2024-08-03

### Added
* Using goreleaser to publish binaries and support homebrew, still experimental

## 0.2.7 - 2024-07-11

### Added
* Admin UI, first steps
* Plugins, first experimental release
* Create db with owner
* Roles and Databases update
* count=exact and tests
* Separate pretty and color log configuration
* Explicit schema
* Schema for DDL operations as header

### Fixes
* Some fixes to logging

## 0.2.6 - 2024-04-15

### Added
* Support for application/octet-stream in input and output
* Support for HEAD calls
* Configuration for ReadTimeout, WriteTimeout and RequestMaxBytes

### Fixed
* Possible panic on logging

## 0.2.5 - 2024-04-01

### Added
* CSV as input and output
* Support for application/x-www-form-urlencoded input data
* Support for Date Postgres type

### Fixed
* Fix table columns quoting
* Fix json keys quoting (allow names with ")
* Fix to properly order function args when getting function info

## 0.2.4 - 2024-03-19

### Added
* Support for Range header
* Functions with GET
* Support for domain types for functions
* Better quoting to prevent SQL injection
* Concept of main database, no more special dbe pool
* Accept empty IN clauses 

### Fixed
* Logging.FileLogging config was not effective
