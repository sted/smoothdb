package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/test"
)

// What a session hit saves per request: the JWT verification and the two
// round trips that prepare the connection (SET ROLE, set_config of the claims).
// "role" skips all of them on a hit, "claims" skips only the verification,
// "none" pays everything every time. Run with:
//
//	go test ./server -run '^$' -bench BenchmarkSessionModeRequest -benchtime=500x
func BenchmarkSessionModeRequest(b *testing.B) {
	modes := []struct{ mode, port string }{{"none", "8099"}, {"role", "8100"}, {"claims", "8101"}}
	for _, m := range modes {
		b.Run(m.mode, func(b *testing.B) {
			s, err := NewServerWithConfig(map[string]any{
				"Address":             "localhost:" + m.port,
				"Database.URL":        "postgresql://postgres:postgres@localhost:5432/postgres",
				"Logging.FileLogging": false,
				"Logging.StdOut":      false,
				"SessionMode":         m.mode,
				"JWTSecret":           "bench-secret",
				"EnableAdminRoute":    false,
			}, &ConfigOptions{ConfigFilePath: b.TempDir() + "/config.jsonc", SkipFlags: true})
			if err != nil {
				b.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- s.Start() }()
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			test.WaitForServer("http://localhost:" + m.port)

			token, _ := authn.GenerateToken("postgres", "bench-secret")
			req, _ := http.NewRequest("GET", "http://localhost:"+m.port+"/api/postgres", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			client := &http.Client{}
			do := func() {
				resp, err := client.Do(req)
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					b.Fatalf("expected 200, got %d", resp.StatusCode)
				}
			}
			do() // warm up: activates the database and builds the session
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				do()
			}
		})
	}
}
