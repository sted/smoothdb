# TODO

## Bugs
* [ ] Fix: dots in filter values without quoting (e.g. eq.autoexec.bat)
* [ ] Fix: empty value after operator (eq. for empty string search)
* [ ] Fix: embedded filter on non-selected resource should return 400
* [ ] Fix: PATCH on non-existent table returns 500 instead of 404
* [ ] Fix: RPC with FTS language operator (fts(english)) type cast

## Query / Filtering
* [ ] Related limit and offset on embedded resources (projects.limit=2)
* [ ] Null filtering on embedded resources (&relation=is.null)
* [ ] OR filtering across embedded resources (or=(rel.not.is.null,...))
* [ ] count=estimated
* [ ] count=planned

## Embedding / Relationships
* [x] Embeds with views
* [x] Chained views embedding (views of views)
* [ ] Spread to-many relationships (correlated arrays)
* [ ] Spread join table columns (junction table fields in M2M spread)
* [ ] Overriding FK relationships with computed functions
* [ ] Recursive relationships (self-referential computed functions)

## API / Headers
* [ ] missing=default header (column DEFAULT for missing values on insert)
* [ ] handling=strict/lenient Prefer header
* [ ] max-affected Prefer header
* [ ] Server-Timing response header
* [ ] location header
* [ ] return=headers-only
* [ ] PUT for upsert
* [ ] Reject JSON arrays with mismatched object keys on bulk insert
* [ ] prefer single-object (now just unnamed functions)
* [ ] Generate OpenAPI
* [ ] OPTIONS support

## Functions / Types
* [ ] variadic function
* [ ] Computed fields
* [ ] More types: Multirange, Domain
* [ ] $ in table/column names

## Infrastructure
* [ ] SQL injection mitigation and related tests
* [ ] Admin
* [ ] Projects
* [ ] Reload config
* [ ] Change log level at runtime
* [ ] Activity / stats
* [ ] Triggers
* [ ] Events / websockets
* [ ] Open telemetry
* [ ] Indexes

## Future
* [ ] Queries, filters: [ ] django mode, [ ] other
* [ ] Db Encryption
* [ ] Table inheritance
* [ ] pg_vector
* [ ] Migrations (Tern?)
* [ ] Versioning
* [ ] Localizations
* [ ] vault integration ?

## Done
* [x] Databases
* [x] Tables
* [x] Columns
* [x] Generic records: different strategies: reflect, database, optimized
* [x] Bulk inserts
* [x] Table creation with columns
* [x] Queries, filters: postgrest mode
* [x] Checks, constraints
* [x] Views
* [x] Alter tables, view and columns
* [x] Some more operator: IN, etc
* [x] Full Text Search (FTS)
* [x] FTS on text/jsonb columns (auto to_tsvector)
* [x] Resource embedding
* [x] Sessions, JWT, Auth
* [x] Roles
* [x] Policies
* [x] Grants
* [x] Configuration (shared with other apps)
* [x] Logger
* [x] pgx v5
* [x] Schema management
* [x] O2O: recognize unique and primary
* [x] Upsert
* [x] M2M Relationships
* [x] aliases for embeds
* [x] spreads for embeds (...table)
* [x] uniform log for db and api
* [x] nested embeds
* [x] nested filters
* [x] "singular" response
* [x] Composite / Array Columns
* [x] !inner joins
* [x] CORS
* [x] TLS
* [x] Range
* [x] Functions (rpc)
* [x] function params with GET
* [x] CSV
* [x] application/x-www-form-urlencoded as input
* [x] application/octet-stream as output
* [x] HEAD support
* [x] range headers
* [x] Related orders
* [x] hints for rels
* [x] Plugins
* [x] Computed table relationships
* [x] count=exact
* [x] Support isdistinct, all, any
* [x] is.not_null value
* [x] Aggregate functions: avg(), count(), max(), min(), and sum()
