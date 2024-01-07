package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
	smoothdb "github.com/sted/smoothdb/server"
)

func prepareContent(s *smoothdb.Server) error {

	dbe_ctx, _, err := database.ContextWithDb(context.Background(), nil, "postgres")
	if err != nil {
		return err
	}
	// create a database
	db, err := s.DBE.CreateActiveDatabase(dbe_ctx, "example", true)
	if err != nil {
		return err
	}
	ctx, _, err := database.ContextWithDb(context.Background(), db, "postgres")
	if err != nil {
		return err
	}
	// delete previous table if exists
	err = database.DeleteTable(ctx, "products", true)
	if err != nil {
		return err
	}
	// create a table 'products'
	_, err = database.CreateTable(ctx, &database.Table{
		Name: "products",
		Columns: []database.Column{
			{Name: "name", Type: "text"},
			{Name: "price", Type: "int4"},
			{Name: "avail", Type: "bool"},
		},
		IfNotExists: true,
	})
	if err != nil {
		return err
	}
	// insert records
	_, _, err = db.CreateRecords(ctx, "products", []database.Record{
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
	if err != nil {
		return err
	}
	// grant read access to everyone
	_, err = database.CreatePrivilege(ctx, &database.Privilege{
		TargetName: "products",
		TargetType: "table",
		Types:      []string{"select"},
		Grantee:    "public",
	})
	return err
}

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
		err = t.Execute(w, results)
		if err == nil {
			return http.StatusOK, nil
		} else {
			return http.StatusInternalServerError, err
		}
	})
	return nil
}

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
	s, err := smoothdb.NewServerWithConfig(baseConfig, nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// prepare db content
	err = prepareContent(s)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// create template and a view route
	err = prepareView(s)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// run
	s.Run()
}
