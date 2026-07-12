# Change Log

## Unreleased

### Security
* **`::cast` injection** — cast targets in `select` are validated at parse time and rejected with 400 otherwise; a cast can't be a bind parameter, so it was interpolated verbatim and could alter the SELECT (reachable by any role that can read).
* **DDL quoting** — quote the remaining `CREATE`/`ALTER DATABASE` identifiers, and whitelist grant/revoke verbs and object types instead of interpolating them.

### Added
* Shutdown now waits for in-flight requests to complete; the wait was previously hardcoded to 1 second, so every restart killed any request slower than that. The new `GracefulShutdownTimeout` config key (seconds, default 0 = wait until done) bounds the wait for deployments that want a hard cap below their supervisor's stop grace period. A second signal during the wait forces an immediate exit, and `Shutdown()` is now idempotent.
* `/ready` now reports `503 {"status":"draining"}` as soon as a graceful shutdown begins, while `/live` keeps answering 200 until the process exits — the standard probe contract for zero-downtime rolling deploys. The new `DrainDelay` config key (seconds, default 0 = disabled) keeps the listener serving for that long after readiness flips, giving load balancers time to deregister the instance before it stops accepting connections. The delay applies to SIGTERM only; an interactive Ctrl-C (SIGINT) shuts down immediately, and a second signal during the window skips it.

### Fixed
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
