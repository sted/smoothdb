package database

import (
	"github.com/jackc/pgx/v4/pgxpool"
)

func PrepareStressTest(conn *pgxpool.Conn) {
}

func CleanStressTest() {
}

func StressTest() {
	// db, _ := dbe.CreateDatabase("bench")
	// defer dbe.DeleteDatabase("bench")

	// db.CreateSource("b1")
	// db.CreateField(&Field{Name: "name", Type: "text", Source: "b1"})
	// db.CreateField(&Field{Name: "number", Type: "integer", Source: "b1"})
	// db.CreateField(&Field{Name: "date", Type: "timestamp", Source: "b1"})

	// for i := 0; i < 10000; i++ {
	// 	db.CreateRecord("b1", &Record{"name": "Morpheus", "number": 42})
	// }

	// _, err := db.GetRecords5("b1")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//fmt.Println(string(j))

}
