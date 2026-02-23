# Change Log

## current

### Added
* Support for Computed Relationships (table-returning functions as virtual relationships)
* Support filtering embedded resources by FK column name (e.g. `client_id.id=eq.1`)

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
