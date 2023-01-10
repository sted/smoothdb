package postgrest

import (
	"green/green-ds/server"
	"green/green-ds/test"
	"log"
	"os"
	"testing"
)

var testConfig = test.TestConfig{
	BaseUrl:       "http://localhost:8081/pgrest",
	CommonHeaders: map[string]string{"Accept-Profile": "test"},
}

func TestMain(m *testing.M) {
	s, err := server.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	go s.Start()
	code := m.Run()
	os.Exit(code)
}
