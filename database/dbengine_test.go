package database

import (
	"context"
	"fmt"
)

func ExampleT() {
	ctx := context.Background()
	names := []string{"t1", "t2", "t3", "t4"}

	for _, n := range names {
		dbe.CreateDatabase(ctx, n)
	}
	databases := dbe.GetActiveDatabases(ctx)
	for _, d := range databases {
		fmt.Println(d.Name)
	}
	// Unordered output:
	// t1
	// t2
	// t3
	// t4
}
