package main

import (
	"fmt"
	"green/green-ds/server"
	"os"
)

func main() {

	s, err := server.NewServer()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println("Listening at ", s.Config.Address)
	err = s.Start()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
