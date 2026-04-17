# Change Log

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
