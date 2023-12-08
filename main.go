package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/sted/smoothdb/server"
)

func stopHandler(s *server.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	fmt.Println("\nStarting shutdown...")
	s.Shutdown(ctx)
}

func main() {
	s, err := server.NewServer()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	go stopHandler(s)

	fmt.Println("Starting server...")
	fmt.Printf("Version %s\n", Version)
	fmt.Println("Listening at ", s.Config.Address)
	err = s.Start()
	if err != nil {
		if err == http.ErrServerClosed {
			fmt.Println("Stopped.")
		} else {
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}
}
