package main

import (
	"flag"
	"fmt"
	"os"
)

var ThisServer *Server

func main() {
	var addr string
	var dburl string
	var err error
	flag.StringVar(&addr, "addr", ":8081", "Address")
	flag.StringVar(&dburl, "dburl", "postgres://localhost:5432/", "DatabaseURL")
	flag.Parse()

	ThisServer, err = NewServer(addr, dburl)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Listening at ", addr)
	err = ThisServer.Start()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
