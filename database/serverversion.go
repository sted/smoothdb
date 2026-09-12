package database

import (
	"context"
	"fmt"
)

// MinServerVersion is the oldest PostgreSQL the server agrees to run on, as a
// server_version_num. It is PostgREST's minimumPgVersion (14.0): smoothdb's
// behaviour is specified as PostgREST's, which is only verified from 14 on,
// and every release below it has reached end of life. InitDbEngine refuses to
// start on an older server with an error naming both versions.
const MinServerVersion = 140000

// server_version_num is major*10000 + minor (160015 is 16.15); PostgREST reads
// the same setting (PostgREST/Config/Database.hs, queryPgVersion).
const serverVersionQuery = "SELECT current_setting('server_version_num')::int"

// GetServerVersion reads the PostgreSQL version of the connection in the
// context, as a server_version_num.
func GetServerVersion(ctx context.Context) (int, error) {
	conn := GetConn(ctx)
	var version int
	err := conn.QueryRow(ctx, serverVersionQuery).Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// ServerAtLeast reports whether the server runs PostgreSQL major or later
// (ServerAtLeast(17) is true on 17.0 and on 17.4, false on 16.15).
//
// Convention for version-gated features: a code path that needs SQL introduced
// in PostgreSQL N checks info.ServerAtLeast(N) and either falls back to the
// pre-N SQL (preferred, whenever one exists, as PostgREST does for pg_basetype
// in its schema cache) or refuses the request with a BuildError (a 400) whose
// message names the required version. It never lets PostgreSQL raise a syntax
// error at execution: that comes back as a 500 that says nothing about the
// cause. A gate on a minor version (a fix that shipped in 16.3) compares
// ServerVersion >= 160003 directly. A hand-built SchemaInfo has ServerVersion
// 0, so it takes the fallback branch of every gate.
func (si *SchemaInfo) ServerAtLeast(major int) bool {
	return si.ServerVersion >= major*10000
}

// checkServerVersion refuses a server older than MinServerVersion.
func checkServerVersion(version int) error {
	if version < MinServerVersion {
		return fmt.Errorf("cannot run on PostgreSQL %s: smoothdb needs at least %s",
			formatServerVersion(version), formatServerVersion(MinServerVersion))
	}
	return nil
}

// formatServerVersion renders a server_version_num the way the server names
// itself: 160015 is "16.15"; before 10 the number packs three parts, 90624
// is "9.6.24".
func formatServerVersion(version int) string {
	major := version / 10000
	if major < 10 {
		return fmt.Sprintf("%d.%d.%d", major, version%10000/100, version%100)
	}
	return fmt.Sprintf("%d.%d", major, version%10000)
}
