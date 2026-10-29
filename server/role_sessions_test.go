package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

// loadResult is what a burst of concurrent requests got.
type loadResult struct {
	mu       sync.Mutex
	statuses map[int]int
	served   map[string]int // per client, the requests answered 200
	maxWait  time.Duration
	wrong    []string // answers that carried another client's role or claims
}

// burst runs, for each client (a token, "" for anonymous, and the role and
// sub its answers must carry), workers goroutines that call authz_whoami in a
// loop until the time is up. Every answer must be the client's own.
func burst(t *testing.T, base string, clients []whoClient, workers int, d time.Duration) *loadResult {
	t.Helper()
	res := &loadResult{statuses: map[int]int{}, served: map[string]int{}}
	// one kept-alive HTTP connection per worker: the default keeps two, and
	// the others would be opened and closed by the thousand
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: len(clients) * workers}}
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(d)
	var wg sync.WaitGroup
	for _, c := range clients {
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(c whoClient) {
				defer wg.Done()
				for time.Now().Before(deadline) {
					req, _ := http.NewRequest("GET", base+"/rpc/authz_whoami", nil)
					if c.token != "" {
						req.Header.Set("Authorization", "Bearer "+c.token)
					}
					start := time.Now()
					resp, err := client.Do(req)
					wait := time.Since(start)
					if err != nil {
						res.mu.Lock()
						res.statuses[0]++
						if res.statuses[0] == 1 {
							t.Log(err)
						}
						res.mu.Unlock()
						continue
					}
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					var who struct{ User, Sub string }
					json.Unmarshal(body, &who)
					res.mu.Lock()
					res.statuses[resp.StatusCode]++
					res.maxWait = max(res.maxWait, wait)
					if resp.StatusCode == 200 {
						res.served[c.name]++
						if who.User != c.role || who.Sub != c.sub {
							res.wrong = append(res.wrong, fmt.Sprintf("%s got %s", c.name, body))
						}
					}
					res.mu.Unlock()
				}
			}(c)
		}
	}
	wg.Wait()
	return res
}

type whoClient struct{ name, token, role, sub string }

func roleSessionsServer(t *testing.T, port string) (*Server, string) {
	t.Helper()
	authzSetup(t)
	superuserSetup(t, []string{"DROP FUNCTION IF EXISTS authz_whoami()"}, []string{
		`CREATE FUNCTION authz_whoami() RETURNS json LANGUAGE sql AS $$
			SELECT json_build_object('user', current_user,
				'sub', coalesce(current_setting('request.jwt.claims', true)::json->>'sub', ''))
			$$`,
	})
	s := newAuthzServer(t, port, map[string]any{
		"SessionMode":                     "role",
		"Database.MinPoolConnections":     int32(10),
		"Database.MaxPoolConnections":     int32(10),
		"Database.PoolAcquisitionTimeout": 3,
	})
	done := startTestServer(t, s)
	t.Cleanup(func() {
		s.Shutdown(context.Background())
		<-done
	})
	return s, "http://localhost:" + port + "/api/postgres"
}

func checkBurst(t *testing.T, res *loadResult, clients []whoClient, bound time.Duration) {
	t.Helper()
	if n := res.statuses[http.StatusGatewayTimeout]; n != 0 {
		t.Errorf("%d requests answered 504 waiting for a connection (statuses %v)", n, res.statuses)
	}
	for status, n := range res.statuses {
		if status != 200 && status != http.StatusGatewayTimeout {
			t.Errorf("%d requests answered %d", n, status)
		}
	}
	for _, c := range clients {
		if res.served[c.name] == 0 {
			t.Errorf("client %s was never served", c.name)
		}
	}
	if res.maxWait > bound {
		t.Errorf("the slowest request took %v, want under %v", res.maxWait, bound)
	}
	for _, w := range res.wrong {
		t.Errorf("an answer carried another client's role or claims: %s", w)
	}
}

// In "role" mode a session keeps its connection between its requests. The
// requests of one token beyond the pool size each built another session,
// which waited for a connection that the busy sessions never gave back: with
// 50 concurrent anonymous clients on a pool of 10, 40 waited until the load
// stopped, or until the acquisition timeout answered them 504. PostgREST,
// which keeps no connection between requests, serves all of them. A waiting
// request must get the connection a session of its token leaves.
func TestRoleSessionsOfOneTokenBeyondThePool(t *testing.T) {
	_, base := roleSessionsServer(t, "8164")
	clients := []whoClient{{"anonymous", "", "authz_anon", ""}}
	res := burst(t, base, clients, 50, 4*time.Second)
	checkBurst(t, res, clients, 2*time.Second)
}

// The same with many tokens, more than the pool has connections, under two
// roles: a connection kept by the session of one token must go to a request of
// another one that waits, and never with the first one's role or claims.
func TestRoleSessionsOfManyTokensBeyondThePool(t *testing.T) {
	_, base := roleSessionsServer(t, "8165")
	var clients []whoClient
	for i := 0; i < 24; i++ {
		role := "authz_user"
		if i%3 == 0 {
			role = "authz_anon"
		}
		sub := fmt.Sprintf("client-%d", i)
		clients = append(clients, whoClient{sub, signClaims(t, map[string]any{"role": role, "sub": sub}, postgrestJWTSecret), role, sub})
	}
	clients = append(clients, whoClient{"anonymous", "", "authz_anon", ""})
	res := burst(t, base, clients, 2, 4*time.Second)
	checkBurst(t, res, clients, 2*time.Second)
}
