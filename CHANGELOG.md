# Change Log

## 0.9.1 - 2026-09-14

### Fixed
* **Columns hidden from the connecting role** — the schema cache read the columns from `information_schema.columns`, which shows a role only the tables it has a privilege on. Loaded by an authenticator that holds none and only `SET ROLE`s per request (the PostgREST deployment), the cache knew no column of any table: 0.9.0's `?columns=` check then refused every bulk insert and upsert with `PGRST204`, and a generated column went undetected. The columns come from `pg_catalog` now, whatever the role's privileges.

## 0.9.0 - 2026-09-14

### Security
* **GET and HEAD run read-only** — like PostgREST, `GET`/`HEAD` (and a `POST` calling a `STABLE`/`IMMUTABLE` function) run in a read-only transaction: a write reached that way fails inside PostgreSQL (`25006`) and is answered `405` with `Allow: POST`, while a `VOLATILE` function that only reads still answers `GET`. `DELETE`/`PATCH`/`PUT` on `/rpc/` answer `405` instead of `404`.
* **Anonymous requests never run as the authenticator** — with `AllowAnon: true` and an empty `Database.AnonRole` the request ran as the connecting role with no `SET ROLE` and no claims GUC. It is now refused with `401 "Anonymous access is disabled"`, as PostgREST does; with a configured anon role the `request.jwt.claims` GUC is set for anonymous requests too.

### Fixed
* **SQLSTATE to HTTP status** — the mapping follows PostgREST's `mapSQLtoHTTP`: an explicit list of codes and classes (foreign-key violation 409, read-only transaction 405, connection and resource errors 503, `PTxyz` chooses its status) and 400 for everything else, where a client-caused error (type mismatch, bad literal, violated constraint, `raise`) used to be a 500. `42501` stays 401 and the DDL "already exists" codes 409.
* **Bulk insert with mismatched keys** — an array whose objects do not share one key set is refused with 400 (`All object keys must match`, as PostgREST) instead of silently dropping the keys the first object lacked. With `?columns=` the listed columns are inserted for every row, an absent key as NULL (it used to get the column default).
* **`order=` on writes** — `order` now orders the representation of `POST`, `PATCH` and `DELETE`, as PostgREST does since 13.0; it was parsed and dropped. `limit`/`offset` on `PATCH`/`DELETE` stay ignored like in PostgREST, which dropped limited updates in the same release.
* **Generated columns** — `attgenerated` is introspected: `$info` and the admin column listing report `"generated"` and `"readonly": true`, and writing such a column is refused with 400 and PostgreSQL's message (`428C9`) instead of a 500, as PostgREST does.
* **Filter values with dots, commas and quotes** — a value was cut at the second dot or at the first comma, colon or quote (`?ver=eq.1.2.3` filtered on `1.2`). As in PostgREST a top-level value now runs to the end of the parameter, and inside `in.()` and logic trees to the next separator unless quoted; `?col=eq.` matches the empty string and `?col=eq` is a 400. Quoting a top-level value (`eq."a,b"`) keeps working, a smoothdb leniency.
* **Malformed requests answer what PostgREST answers** — a filter on a relation that is not embedded in the request, or does not exist (`?nonexistent.id=eq.1`, `projects.tasks2.name=…` with `tasks2` not selected), was silently dropped and the request answered as if it were not there; it is now a 400 naming the relation (PostgREST `PGRST108`), on writes too when a representation is requested. `?columns=` naming a column the table does not have is a 400 (`PGRST204`) instead of a PostgreSQL syntax error, and `?columns=` that leaves nothing of a `PATCH` body to set is a no-op instead of an `UPDATE` with an empty `SET`, so a `PATCH` on a missing table answers 404 whatever the columns. A `PATCH` body with more than one object is a 400 with a message (it was a nil dereference, recovered as 500); an unknown embed in the `select` of a write's representation is refused as on a `GET` instead of running a broken statement.
* **`in.(a."b,c",d)`** — a quote after a dot inside an unquoted element opened a quoted token that swallowed the comma, and the list was refused with 400; as in PostgREST a quote only opens at the start of an element, so this is `a."b`, `c"`, `d`.
* **Wire formats** — `bytea`, `inet`, `cidr`, `macaddr`, `time`, `bit`, the geometric types, `tid`/`xid`, `tsvector`, `xml` and their arrays came out as raw binary bytes. The serializers now dispatch on the wire format of each column, request such types in text and print them as `to_json` does (`bytea` as the `\x` hex string); text-format arrays and composites are parsed into JSON arrays and objects (this also fixes arrays of enums), a domain prints as its base type, an unknown binary type is refused instead of copied through, `DateStyle` is pinned to ISO, and an `application/octet-stream` download of a `bytea` still returns the bytes.
* **Ranges and multiranges** — `empty` and unbounded ranges crashed the serializer (a 406 with an empty body), `daterange`/`tsrange`/`tstzrange` came out as invalid JSON, multiranges failed on a nil subtype. Every range and multirange type is now sent in text and returned as PostgreSQL's own string (`"[2024-01-01,2024-06-01)"`, `"{[1,3),[5,7)}"`), exactly as PostgREST returns it.
* **Stale plans after a migration** — the schema reload (the `reload schema` notify) now resets the connection pool as well, as PostgREST does: a view recreated with other columns, or a column added to a table read with `RETURNING *`, no longer answers `0A000` "cached plan must not change result type" once per pooled connection until a restart.
* **`via()` walks enumerated paths, not nodes** — the recursive CTE carried the whole row plus a key array and materialised every simple path (797,161 rows and 791 ms for 490 nodes on a 650-node DAG); it now carries `(node, depth)`, dedups with `UNION` and joins the table back: 1 ms for the same walk, `via!both` joins the edges as a derived table, and a walk over a table with a `json`, `xml` or `point` column no longer fails. `__path` is a selectable pseudo-column (on `via` it restores the path-enumerating shape, documented as exponential), ordering by an unselected `__depth` no longer leaks it, and a `__depth` filter sees the node's shortest depth.

### Added
* **`Database.ExposedSchemas`** — the schemas a request may select with `Accept-Profile`/`Content-Profile`, the first being the default (PostgREST's `db-schemas`): a header naming another schema answers 406 with the exposed list as hint (`PGRST106`), before any SQL. Unset, every schema stays reachable as before, and the default is the first of `SchemaSearchPath`, which keeps meaning only the connection's search path (PostgREST's `db-extra-search-path`).
* **Server version** — `SchemaInfo.ServerVersion` (`server_version_num`) and `ServerAtLeast(N)` for version-gated features; startup logs the PostgreSQL version and, like PostgREST, refuses a server older than 14.

### Changed
* **Homebrew: a Cask instead of a Formula** — GoReleaser deprecated formulas of pre-built binaries, so `sted/tap` now publishes `smoothdb` as a Cask, on Linux too. `brew update` finds an installed formula in the tap's `tap_migrations.json` and moves it to the cask: by itself up to Homebrew 6, while Homebrew 7, which loads a third-party cask only once trusted, prints the commands to run (`brew uninstall --formula smoothdb`, `brew trust --cask sted/tap/smoothdb`, `brew install --cask sted/tap/smoothdb`). The binaries are not notarized: the cask clears the quarantine flag Homebrew sets on a cask download (a formula download never had it).
* **Go 1.27** — the modules and the release binaries are built with Go 1.27.1; plugins must be built with the same toolchain.

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
