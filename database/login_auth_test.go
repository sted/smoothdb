package database

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

// readerConn is a net.Conn that reads a fixed byte stream in small pieces.
type readerConn struct {
	net.Conn
	r     *bytes.Reader
	chunk int
}

func (c *readerConn) Read(p []byte) (int, error) {
	if len(p) > c.chunk {
		p = p[:c.chunk]
	}
	return c.r.Read(p)
}

func backendMessage(kind byte, payload []byte) []byte {
	m := []byte{kind, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(m[1:], uint32(4+len(payload)))
	return append(m, payload...)
}

func authMessage(code uint32) []byte {
	p := make([]byte, 4)
	binary.BigEndian.PutUint32(p, code)
	return backendMessage('R', p)
}

// The first authentication message of the server says whether it asked for a
// password: AuthenticationOk at once (0) is trust, peer, ident or a client
// certificate; 3 is cleartext, 5 MD5 and 10 SASL (SCRAM). The watcher reads it
// whatever the size of the reads, and after a NegotiateProtocolVersion.
func TestAuthWatcherSeesThePasswordRequest(t *testing.T) {
	cases := []struct {
		name   string
		stream []byte
		asked  bool
	}{
		{"trust", authMessage(0), false},
		{"cleartext", authMessage(3), true},
		{"md5", append(authMessage(5), 1, 2, 3, 4), true},
		{"scram", append(authMessage(10), []byte("SCRAM-SHA-256\x00\x00")...), true},
		{"gss", authMessage(7), false},
		{"after NegotiateProtocolVersion", append(backendMessage('v', []byte{0, 0, 0, 2, 0, 0, 0, 0}), authMessage(10)...), true},
	}
	for _, c := range cases {
		for _, chunk := range []int{1, 3, 64} {
			w := newAuthWatcher(&readerConn{r: bytes.NewReader(c.stream), chunk: chunk})
			buf := make([]byte, 64)
			var got []byte
			for {
				n, err := w.Read(buf)
				got = append(got, buf[:n]...)
				if err != nil {
					break
				}
			}
			if !bytes.Equal(got, c.stream) {
				t.Errorf("%s, chunk %d: the watcher changed the stream", c.name, chunk)
			}
			if w.passwordAsked() != c.asked {
				t.Errorf("%s, chunk %d: passwordAsked %v, want %v", c.name, chunk, w.passwordAsked(), c.asked)
			}
		}
	}
}
