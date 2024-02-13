# SmoothDB  ![Tests](https://github.com/sted/smoothdb/actions/workflows/tests.yml/badge.svg)

SmoothDB provides a RESTful API to PostgreSQL databases.

Configured databases and schemas can be accessed and modified easily with a REST JSON-based interface.

It is mostly compatible with [PostgREST](https://postgrest.org/en/stable/), with which it shares many characteristics.

The main differences are:

* SmoothDB is in development and alpha quality. Prefer PostgREST for now, which is rock solid
* SmoothDB is faster and has a lower CPU load 
* It is written in Go
* Can be used both stand-alone and as a library (the main motivation for writing this)
* It also supports DDL operations (create / alter / drop databases, tables, manage constraints, roles, etc)
* Supports multiple databases with a single instance

See [TODO.md](TODO.md) for the many things to be completed.
Please create issues to let me know your priorities.

## Getting started

### Install and build

If you have Go installed, installing the binary is easy:

```
go install github.com/sted/smoothdb@latest
```

To test your installation (provided $GOPATH/bin is in your path) by typing `smoothdb`.

### Start

Starting SmoothDB, it creates a configuration file named **config.jsonc** in the current directory, with default values:  edit it for further customizations (see [Configuration](#configuration-file)).

## API

Here you find some examples for the API.
For more detailed information, see [PostgREST API](https://postgrest.org/en/stable/references/api.html).

The compability with PostgREST has also the great advantage of being able to use the many existing client libraries, starting from [postgrest-js](https://github.com/supabase/postgrest-js). See the [complete list of available client libraries](https://github.com/supabase/postgrest-js).

The default Content-Type is "**application/json**".

### Authentication

Like PostgREST (see [PostgREST Authentication](https://postgrest.org/en/stable/references/auth.html)), smoothdb is designed to keep the database at the center of API security.

To make an authenticated request, the client must include an Authorization HTTP header with the value **Bearer \<jwt\>**, where **jwt** is a [Java Web Token](jwt.io). 

A valid JWT for SmoothDB must include at least the **role** claim in the payload:

```json
{
    "role": "user1"
}
```

To generate a JWT for testing, you can use the generator at [jwt.io](jwt.io), using as a secret the same value configured in the configuration file for JWTSecret.

This is an example of an authenticated API call:

```http
GET /test HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoic3RlZCJ9.-XquFDiIKNq5t6iov2bOD5k_LljFfAN7LqRzeWVuv7k
```
We will omit this header in the following examples.

### Create a database
```http
POST /admin/databases HTTP/1.1

{ "name": "testdb" }
```

### Create a table

```http
POST /admin/databases/testdb/tables HTTP/1.1

{ 
    "name": "test",
    "columns": [
		{"name": "col1", "type": "text", "notnull": true},
		{"name": "col2", "type": "boolean"},
		{"name": "col3", "type": "integer", "default": "42", "constraints": ["CHECK (col3 > 40)"]},
		{"name": "col4", "type": "timestamp"},
		{"name": "arr", "type": "integer[]"},
		{"name": "extra", "type": "json"},
		{"name": "duration", "type": "tsrange"},
		{"name": "other", "type": "text", "constraints": ["REFERENCES test (col1)"]}
	],
	"constraints": ["PRIMARY KEY (col1)"]
}
```

### Insert records

Insert one record:

```http
POST /api/testdb/test HTTP/1.1

{
	"col1": "",
	"col2": false,
	"extra": {
		"a": "pippo",
		"b": 4444,
		"c": [1,2,3,"d"]
	},
	"arr": [1,2,3],
	"duration": "['2022-12-31 11:00','2023-01-01 06:00']"
}
```

Insert multiple records:

```http
POST /api/testdb/test HTTP/1.1

[
	{ "col1": "one", "col3": 43},
	{ "col1": "two", "col3": 44}
]
```

> [!IMPORTANT]
> In these example we use the default configuration for SmoothDB.
> To have fully PostgREST API compliancy, you should configure:
> ```json 
> 	EnableAdminRoute: false
> 	BaseAPIURL: ""
> 	ShortAPIURL: true
> 	Database.AllowedDatabases: [<single_db_name>]
> ```
> With these configurations the "/admin" is no longer accessible and "/api/testdb/test..." becomes simply "/test...".

### Select records

```http
GET /api/testdb/test?col3=gt.42 HTTP/1.1
```
```json
[
  { "col1": "one", "col2": null, "col3": 43, "col4": null, "arr": null, "extra": null, "duration": null, "other": null },
  { "col1": "two", "col2": null, "col3": 44, "col4": null, "arr": null, "extra": null, "duration": null, "other": null }
]
```

Most operators in [PostgREST Operators](https://postgrest.org/en/stable/references/api/tables_views.html#operators) are supported.

More conditions can be combined with the **and**, **or**, **not** operators ('and' being the default):

```http
GET /api/testdb/people?grade=gte.90&student=is.true&or=(age.eq.14,not.and(age.gte.11,age.lte.17)) HTTP/1.1
```

Use the **select** parameter to specify which column to show:

```http
GET /api/testdb/test?select=col1,col3&col3.gt=42 HTTP/1.1
```
```json
[
  { "col1": "one", "col3": 43 },
  { "col1": "two", "col3": 44 }
]
```

Pagination is controlled with **limit** and **offset** query parameters:

```http
GET /api/testdb/pages?limit=15&offset=30 HTTP/1.1
```

The **Range** header will be added soon.

### Relationships

You can include related resources in a single API call.

SmoothDB uses Foreign Keys to determine which tables can be joined together, allowing many-to-one, many-to-many, one-to-many and one-to-one relationships.

To make a request joining data from multiple tables, you use again the **select** parameter, specifying the additional tables and the required columns for each:

```http
GET /api/testdb/orders?select=id,amount,companies(name,category) HTTP/1.1
```
```json
[
  { "id": "1234", "amount": 1000 , "companies": { "name": "audi", "category": "cars"}},
  { "id": "5678", "amount": 2000 , "companies": { "name": "bmw", "category": "cars"}}
]
```

## Example for using smoothdb in your application

You can embed smoothdb functionalities in your backend app with relative ease.

This short example is a minimal app that exposes a **/products** GET route to obtain the JSON array of the products and a **/view** route to view them in a formatted HTML table.

In this note we omit error handling for brevity, see the whole example in [examples/server.go](examples/server.go).

> [!WARNING]
> While you can already be confident with the retro compatibility of the API, because of the goal of
> compatibility with PostgREST, this is not yet the case for the exported functions in the various
> packages.

```go
import (
	...
	
	"github.com/sted/heligo"
	"github.com/sted/smoothdb/api"
	"github.com/sted/smoothdb/database"
	smoothdb "github.com/sted/smoothdb/server"
)

func main() {
	// base configuration
	baseConfig := map[string]any{
		"Address":                   ":8085",
		"AllowAnon":                 true,
		"BaseAPIURL":                "",
		"ShortAPIURL":               true,
		"Logging.FilePath":          "./example.log",
		"Database.AllowedDatabases": []string{"example"},
	}
	// smoothdb initialization
	s, _ := smoothdb.NewServerWithConfig(baseConfig, nil)
	
	// -- here the database is connected and the standard routes are prepared
	
	// prepare db content
	prepareContent(s)
	// create template and a view route
	prepareView(s)
	// run
	s.Run()
}
```
In *prepareContent* we see the basic interactions with the database.

```go
func prepareContent(s *smoothdb.Server) error {

	dbe_ctx, _, _ := database.ContextWithDb(context.Background(), nil, "postgres")
	// create a database
	db, _ := s.DBE.CreateActiveDatabase(dbe_ctx, "example", true)
	ctx, _, err := database.ContextWithDb(context.Background(), db, "postgres")
	// delete previous table if exists
	database.DeleteTable(ctx, "products", true)
	// create a table 'products'
	database.CreateTable(ctx, &database.Table{
		Name: "products",
		Columns: []database.Column{
			{Name: "name", Type: "text"},
			{Name: "price", Type: "int4"},
			{Name: "avail", Type: "bool"},
		},
		IfNotExists: true,
	})
	// insert records
	database.CreateRecords(ctx, "products", []database.Record{
		{"name": "QuantumDrive SSD 256GB", "price": 59, "avail": true},
		{"name": "SolarGlow LED Lamp", "price": 99, "avail": false},
		{"name": "AquaPure Water Filter", "price": 20, "avail": true},
		{"name": "BreezeMax Portable Fan", "price": 5, "avail": true},
		{"name": "Everlast Smartwatch", "price": 200, "avail": false},
		{"name": "JavaPro Coffee Maker", "price": 45, "avail": true},
		{"name": "SkyView Drone", "price": 150, "avail": true},
		{"name": "EcoCharge Solar Charger", "price": 30, "avail": false},
		{"name": "GigaBoost WiFi Extender", "price": 75, "avail": true},
		{"name": "ZenSound Noise-Canceling Headphones", "price": 10, "avail": false},
	}, nil)
	// grant read access to everyone
	database.CreatePrivilege(ctx, &database.Privilege{
		TargetName: "products",
		TargetType: "table",
		Types:      []string{"select"},
		Grantee:    "public",
	})
	return nil
}
```

In *prepareView* we create a standard html/template and register a route to view the content.

```go
func prepareView(s *smoothdb.Server) error {
	// create the template
	t, _ := template.New("").Parse(`
		<html>
		<head>
		<style>
			table {
				margin-left: auto;
    			margin-right: auto;
				border-collapse: collapse;
				border: 2px solid rgb(200, 200, 200);
				letter-spacing: 1px;
				font-family: sans-serif;
				font-size: 0.8rem;
			}
			th {
				background-color: #3f87a6;
				color: #fff;
		  	}
			td {
				background-color: #e4f0f5;
			}
			td,th {
				border: 1px solid rgb(190, 190, 190);
				padding: 5px 10px;
			}  
		</style>
		</head>
		<body>
		<h1>Products</h1>
		<table>
			<tr><th>Name</th><th>Price</th><th>Avail</th></tr>
			{{range .}}
				<tr>
					<td><b>{{.Name}}</b></td><td>{{.Price}}</td><td>{{.Avail}}</td>
				</tr>
			{{end}}
		</table>
		</body>`)
	
	// register a route
	r := s.Router()
	m := s.MiddlewareWithDbName("example")
	g := r.Group("/view", m)
	g.Handle("GET", "", func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		results, err := database.GetStructures(ctx, "products")
		if err != nil {
			return api.WriteError(w, err)
		}
		err = t.Execute(w, results)
		if err == nil {
			return http.StatusOK, nil
		} else {
			return http.StatusInternalServerError, err
		}
	})
	return nil
}
```

To try the example

	go run server.go

in the examples directory and browse to *localhost:8085/products* and *localhost:8085/view*.

## Configuration

Configuration parameters can be provided via configuration file, environment variables and command line, with increasing priority.

### Configuration file

The configuration file *config.jsonc* (JSON with Comments) is created automatically on the first start, in the working directory. It contains the following parameters with their defaults:

| Name | Description | Default |
| --- | --- | --- |
| Address | Server address and port | 4000 |
| CertFile | TLS certificate file | "" |
| KeyFile | TLS certificate key file | "" |
| AllowAnon | Allow unauthenticated connections | false |
| JWTSecret | Secret for JWT tokens | "" |
| SessionMode | Session mode: "none", "role" | "role" |
| EnableAdminRoute | Enable administration of databases and tables | false |
| EnableAPIRoute | Enable API access | true |
| BaseAPIURL | Base URL for the API | "/api" |
| ShortAPIURL | Skip database name in API URL. Database.AllowedDatabases must contain a single db | false |
| BaseAdminURL | Base URL for the Admin API | "/admin" |
| CORSAllowedOrigins | CORS Access-Control-Allow-Origin | ["*"] |
| CORSAllowCredentials | CORS Access-Control-Allow-Credentials | false |
| General transaction mode for operations | Enable debug access | false |
| Database.URL | Database URL as postgresql://user:pwd@host:port/database | "postgres://localhost:5432" |
| Database.MinPoolConnections | Miminum connections per pool | 10 |
| Database.MaxPoolConnections | Maximum connections per pool | 100 |
| Database.AnonRole | Anonymous role | "anon" |
| Database.AllowedDatabases | Allowed databases | [] for all |
| Database.SchemaSearchPath | Schema search path | [] for Postgres search path |
| Database.TransactionMode | General transaction mode for operations: "none", "commit", "rollback" | "none" |
| Logging.Level | Log level: trace, debug, info, warn, error, fatal, panic | "info" |
| Logging.FileLogging | Enable logging to file | true |
| Logging.FilePath | File path for file-based logging | "./smoothdb.log" |
| Logging.MaxSize | MaxSize is the maximum size in megabytes of the log file before it gets rotated | 25 |
| Logging.MaxBackups |  MaxBackups is the maximum number of old log files to retain | 3 |
| Logging.MaxAge | MaxAge is the maximum number of days to retain old log files | 5 |
| Logging.Compress | True to compress old log files | false |
| Logging.StdOut | Enable logging to stdout | false |
| Logging.PrettyConsole | Enable pretty and colorful output for stdout | false |

### Environment variables

| Name | Description |
| --- | --- | 
| SMOOTHDB_DATABASE_URL | Database.URL |
| SMOOTHDB_ALLOW_ANON | AllowAnon |
| SMOOTHDB_ENABLE_ADMIN_ROUTE | EnableAdminRoute | 
| SMOOTHDB_DEBUG | true forces: AllowAnon: true, EnableAdminRoute: true, Logging.Level: "trace", Logging.StdOut: true, EnableDebugRoute: true |
	
### Command line parameters

You can pass some configuration parameters in the command line:

```
$ ./smoothdb -h

Usage: smoothdb [options]

Server Options:
	-a, --addr <host>                Bind to host address (default: localhost:4000)
	-d, --dburl <url>                Database URL (default: postgres://localhost:5432)			
	-c, --config <file>              Configuration file (default: ./config.jsonc)
	-h, --help                       Show this message
```

### Database configuration

The way SmoothDB connect to PostgreSQL is through the Database.URL configuration:
	
	postgres[ql]://[user:password@]host:port[/database]

The specified user will be used as the **authenticator**, so it should be a user with limited privileges.

Specifying the database in the URL is a way to restrict the allowed databases just to this one and should not conflict with the Database.AllowedDatabases configuration (ie the latter should be empty or contain just the same db name).

## Development

Contributions are warmly welcomed in the form of Pull Requests and Issue reporting.

Some areas needing particular attention:

* Security
* Completing features present in PostgREST
* Verifying compatibility
* Documentation
* Performances and benchmarks

### Tests

There are three categories of tests: 

* Internal unit tests
* API tests 
* PostgREST tests

The last ones are taken directly from the original project.

To launch all the tests:

```
make test
```

To initialize and reset PostgREST fixtures:

```
make prepare-postgrest-tests
```
