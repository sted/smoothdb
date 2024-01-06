# SmoothDB  ![Tests](https://github.com/sted/smoothdb/actions/workflows/tests.yml/badge.svg)

SmoothDB provides a RESTful API to PostgreSQL databases.

Configured databases and schemas can be accessed and modified easily with a REST JSON-based interface.

It is mostly compatible with [PostgREST](https://postgrest.org/en/stable/), with which it shares many characteristics.

The main differences are:

* SmoothDB is in development and alpha quality. Prefer PostgREST for now, which is rock solid
* SmoothDB is faster and has a lower CPU load 
* It is written in Go
* Can be used both stand-alone and as a library (my main motivation for writing this)
* It also supports DDL (create / alter / drop databases, tables, manage constraints, roles, etc)
* Supports multiple databases

See [TODO.md](TODO.md) for the many things to be completed.
Please create issues to let me know your priorities.

## Getting started

### Install and build

```
go install github.com/sted/smoothdb@latest
```

### Start

```
smoothdb
```

Starting SmoothDB this way, it creates a configuration file named **config.jsonc** in the current directory, which can be edited for further customizations.

## API

Here you find some examples for the API.
For more detailed information, see [PostgREST API](https://postgrest.org/en/stable/references/api.html).

The default Content-Type is "**application/json**".

### Authentication

Like PostgREST (see [PostgREST Authentication](https://postgrest.org/en/stable/references/auth.html)), smoothdb is designed to keep the database at the center of API security.

To make an authenticated request, the client must include an Authorization HTTP header with the value **Bearer \<jwt\>**. For instance:

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

```go
import (
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
	db.CreateRecords(ctx, "products", []database.Record{
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
	t, err := template.New("").Parse(`
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
	if err != nil {
		return err
	}
	// register a route
	r := s.GetRouter()
	m := smoothdb.DatabaseMiddlewareWithName(s, "example")
	g := r.Group("/view", m)
	g.Handle("GET", "", func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		db := database.GetDb(ctx)
		results, err := db.GetStructures(ctx, "products")
		if err != nil {
			return smoothdb.WriteError(w, err)
		}
		t.Execute(w, results)
		return 200, nil
	})
	return nil
}
```

To try the example

	go run server.go

in the examples directory and browse to *localhost:8085/products* and *localhost:8085/view*.

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
