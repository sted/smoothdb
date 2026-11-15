package database

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNoPasswordCheck refuses a db login when PostgreSQL did not ask for a
// password: under trust, peer, ident or certificate authentication a
// successful connection proves nothing about the password of the request.
var ErrNoPasswordCheck = errors.New("PostgreSQL did not ask for a password (trust, peer, ident or " +
	"certificate authentication): set LoginAllowTrust to accept such a login, for development only")

// Authentication request codes of the AuthenticationRequest message ('R')
// that ask the client for a password: cleartext, MD5 and SASL (SCRAM).
const (
	authCleartext = 3
	authMD5       = 5
	authSASL      = 10
)

// authWatcher passes a connection through and reads the first authentication
// message of the server, to learn whether it asked for a password. It sits
// after TLS (pgconn's AfterNetConnect), so it reads the plain protocol.
type authWatcher struct {
	net.Conn
	buf     []byte
	decided bool
	asked   atomic.Bool
}

func newAuthWatcher(conn net.Conn) *authWatcher {
	return &authWatcher{Conn: conn}
}

func (w *authWatcher) passwordAsked() bool {
	return w.asked.Load()
}

func (w *authWatcher) Read(p []byte) (int, error) {
	n, err := w.Conn.Read(p)
	if !w.decided && n > 0 {
		w.buf = append(w.buf, p[:n]...)
		w.scan()
	}
	return n, err
}

// scan reads the backend messages buffered so far until the first
// AuthenticationRequest. A message before it (NegotiateProtocolVersion) is
// skipped; an ErrorResponse ends the watch, since no authentication follows.
func (w *authWatcher) scan() {
	for len(w.buf) >= 5 {
		kind := w.buf[0]
		size := int(binary.BigEndian.Uint32(w.buf[1:5]))
		if size < 4 || size > 1<<20 {
			w.stop()
			return
		}
		if kind == 'R' {
			if len(w.buf) < 9 {
				return
			}
			switch binary.BigEndian.Uint32(w.buf[5:9]) {
			case authCleartext, authMD5, authSASL:
				w.asked.Store(true)
			}
			w.stop()
			return
		}
		if kind == 'E' {
			w.stop()
			return
		}
		if len(w.buf) < 1+size {
			return
		}
		w.buf = w.buf[1+size:]
	}
}

func (w *authWatcher) stop() {
	w.decided = true
	w.buf = nil
}

// VerifyAuthN checks a login by connecting to the main database as user.
// ctx bounds the dial and the ping: a client that hangs up cancels them.
// The connection proves the password only when PostgreSQL asked for one:
// otherwise the login is refused, unless allowTrust. An empty password and
// the authenticator's own name are refused before any connection.
func VerifyAuthN(ctx context.Context, user, password string, allowTrust bool) error {
	if password == "" {
		return errors.New("empty password")
	}
	config, err := pgx.ParseConfig(dbe.config.URL)
	if err != nil {
		return err
	}
	if user == config.User {
		return errors.New("the authenticator may not log in through /token")
	}
	config.User = user
	config.Password = password
	// the watcher of the connection that succeeds is the last one: a fallback
	// attempt (sslmode=prefer) dials again
	var last atomic.Pointer[authWatcher]
	config.AfterNetConnect = func(ctx context.Context, _ *pgconn.Config, conn net.Conn) (net.Conn, error) {
		w := newAuthWatcher(conn)
		last.Store(w)
		return w, nil
	}
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if !allowTrust {
		if w := last.Load(); w == nil || !w.passwordAsked() {
			return ErrNoPasswordCheck
		}
	}
	// Test the connection
	return conn.Ping(ctx)
}

// AuthenticatorPasswordAsked reports whether PostgreSQL asks the
// authenticator for a password. When it does not, it does not ask the users
// of /token either, and a db login without LoginAllowTrust always fails.
func AuthenticatorPasswordAsked(ctx context.Context) (bool, error) {
	config, err := pgx.ParseConfig(dbe.config.URL)
	if err != nil {
		return false, err
	}
	var last atomic.Pointer[authWatcher]
	config.AfterNetConnect = func(ctx context.Context, _ *pgconn.Config, conn net.Conn) (net.Conn, error) {
		w := newAuthWatcher(conn)
		last.Store(w)
		return w, nil
	}
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return false, err
	}
	defer conn.Close(ctx)
	w := last.Load()
	return w != nil && w.passwordAsked(), nil
}
