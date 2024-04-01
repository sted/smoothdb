# Change Log

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
