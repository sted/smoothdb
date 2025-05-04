package main

import (
	"fmt"
	"os"

	"github.com/sted/smoothdb/server"
	"github.com/sted/smoothdb/version"
)

func main() {
	s, err := server.NewServer()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println("Starting server...")
	fmt.Printf("Version %s\n", version.Version)
	fmt.Println("Listening at ", s.Config.Address)
	s.Run()
}
