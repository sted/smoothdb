package database

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// roleSettings are the settings of the roles the authenticator can become
// (ALTER ROLE ... SET), which every request run as one of them gets, as in
// PostgREST (Config/Database.hs, queryRoleSettings; Query/PreQuery.hs). They
// are read at startup and again with every schema cache reload.
type roleSettings struct {
	roles map[string]*roleConfig
	// names are the settings loaded since startup, and reset their RESET
	// (for search_path, the configured one set again: see searchPathSet),
	// appended to the statement that returns an autocommit connection to the
	// pool: a setting applied for one role must not reach the next role that
	// takes the connection, even after a reload dropped it from the cache.
	names map[string]struct{}
	reset string
}

// roleConfig holds one role's settings and the statements that apply them
// together with the role switch and the claims, in one round trip. In a
// transaction (sqlTX) the settings, the role and the claims are
// transaction-local and the isolation is the BEGIN's; sqlLocal is sqlTX with
// the access mode, for the implicit transaction of a batch (see
// requestvars.go); with autocommit (sqlAuto) they are session settings, the
// isolation among them, that last while the connection serves the role.
type roleConfig struct {
	isolation                string // REPEATABLE READ, SERIALIZABLE, "" for the default
	txArgs, autoArgs         []any  // name, value, name, value, ...
	sqlTX, sqlLocal, sqlAuto string
}

// The statements of a role without settings. The role is set with set_config
// like the settings, so that the whole preparation is one statement: with
// autocommit a refused switch then rolls back the settings before it. In a
// transaction everything is transaction-local, as in PostgREST (txVarQuery):
// it ends with the transaction, and the connection goes back to the pool with
// nothing of the request.
const (
	prepareTX    = "SELECT set_config('role', $1, true), set_config('request.jwt.claims', $2, true)"
	prepareLocal = "SELECT set_config('role', $1, true), set_config('request.jwt.claims', $2, true), set_config('" + txReadOnlyGUC + "', $3, true)"
	prepareAuto  = "SELECT set_config('role', $1, false), set_config('request.jwt.claims', $2, false), set_config('" + readOnlyGUC + "', $3, false)"
)

// roleSettingsQuery is PostgREST's queryRoleSettings, as rows. The
// authenticator is session_user rather than current_user, which on a pooled
// connection may be the role it was last switched to.
const roleSettingsQuery = `
	WITH role_setting AS (
		SELECT r.rolname, unnest(r.rolconfig) AS setting
		FROM pg_auth_members m
		JOIN pg_roles r ON r.oid = m.roleid
		WHERE m.member = (SELECT oid FROM pg_roles WHERE rolname = session_user)
	),
	kv_settings AS (
		SELECT rolname,
			substr(setting, 1, strpos(setting, '=') - 1) AS key,
			substr(setting, strpos(setting, '=') + 1) AS value
		FROM role_setting
	)
	SELECT kv.rolname, kv.key, kv.value
	FROM kv_settings kv
	JOIN pg_settings ps ON ps.name = kv.key AND (ps.context = 'user' %s)
	ORDER BY 1, 2`

// isolationLevel is PostgREST's toIsolationLevel: anything but repeatable
// read and serializable is read committed, the default.
func isolationLevel(value string) string {
	switch strings.ToLower(value) {
	case "repeatable read":
		return "REPEATABLE READ"
	case "serializable":
		return "SERIALIZABLE"
	}
	return ""
}

// loadRoleSettings reads the role settings with conn, a connection of the
// authenticator. The setting names of prev stay in the reset.
func loadRoleSettings(ctx context.Context, conn *DbConn, version int, prev *roleSettings) (*roleSettings, error) {
	privilege := ""
	if version >= 150000 {
		privilege = "OR has_parameter_privilege((SELECT oid FROM pg_roles WHERE rolname = session_user), ps.name, 'set')"
	}
	rows, err := conn.Query(ctx, fmt.Sprintf(roleSettingsQuery, privilege))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rs := &roleSettings{roles: map[string]*roleConfig{}, names: map[string]struct{}{}}
	if prev != nil {
		for name := range prev.names {
			rs.names[name] = struct{}{}
		}
	}
	for rows.Next() {
		var role, name, value string
		if err := rows.Scan(&role, &name, &value); err != nil {
			return nil, err
		}
		if strings.EqualFold(name, "datestyle") {
			// the order of the role's DateStyle, but ISO output: the text
			// decoders read ISO (see isoDateStyle), and PostgREST's JSON is
			// ISO whatever the DateStyle
			value = isoDateStyle(map[string]string{name: value})
		}
		rc := rs.roles[role]
		if rc == nil {
			rc = &roleConfig{}
			rs.roles[role] = rc
		}
		if name == "default_transaction_isolation" {
			// the isolation of the transaction, not a setting (MainTx.hs)
			rc.isolation = isolationLevel(value)
			if rc.isolation == "" {
				continue
			}
			value = strings.ToLower(rc.isolation)
		} else {
			rc.txArgs = append(rc.txArgs, name, value)
		}
		rc.autoArgs = append(rc.autoArgs, name, value)
		rs.names[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rs.names))
	for name := range rs.names {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if set := searchPathSet(); set != "" && strings.EqualFold(name, "search_path") {
			// back to the configured path, which a RESET would not restore
			rs.reset += "; SELECT " + set
		} else {
			rs.reset += "; RESET " + quote(name)
		}
	}
	for _, rc := range rs.roles {
		rc.sqlTX = prepareStatement(rc.txArgs, true, withSwitch)
		rc.sqlLocal = prepareStatement(rc.txArgs, true, withTxAccessMode)
		rc.sqlAuto = prepareStatement(rc.autoArgs, false, withAccessMode)
	}
	return rs, nil
}

// The ends of a preparation statement.
const (
	withSwitch       = iota // the role switch and the claims
	withAccessMode          // those, and the access mode of an autocommit session
	withTxAccessMode        // those, and the access mode of the transaction
)

// prepareStatement writes a statement that applies the settings in args
// before the role switch, as PostgREST orders them: a GRANT SET ON PARAMETER
// to the authenticator is what lets it set a superuser setting, so it has to
// come before the switch. With no settings it is prepareTX, prepareLocal or
// prepareAuto. local makes every set_config of the statement
// transaction-local.
func prepareStatement(args []any, local bool, end int) string {
	var b strings.Builder
	b.WriteString("SELECT ")
	p := 1
	for ; p < len(args); p += 2 {
		if p > 1 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "set_config($%d, $%d, %t)", p, p+1, local)
	}
	if p > 1 {
		b.WriteString(", ")
	}
	fmt.Fprintf(&b, "set_config('role', $%d, %t), set_config('request.jwt.claims', $%d, %t)", p, local, p+1, local)
	switch end {
	case withAccessMode:
		fmt.Fprintf(&b, ", set_config('%s', $%d, %t)", readOnlyGUC, p+2, local)
	case withTxAccessMode:
		fmt.Fprintf(&b, ", set_config('%s', $%d, %t)", txReadOnlyGUC, p+2, local)
	}
	return b.String()
}

// roleConfigOf returns the settings of role, nil when it has none.
func roleConfigOf(role string) *roleConfig {
	rs := dbe.roleSettings.Load()
	if rs == nil {
		return nil
	}
	return rs.roles[role]
}

// roleSettingsReset is the RESET of every role setting loaded so far, to
// append to a multi-statement string.
func roleSettingsReset() string {
	rs := dbe.roleSettings.Load()
	if rs == nil {
		return ""
	}
	return rs.reset
}

// ReloadRoleSettings reads the role settings again, with a connection of the
// main pool.
func (dbe *DbEngine) ReloadRoleSettings(ctx context.Context) error {
	conn, err := dbe.mainDb.AcquireConnection(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	rs, err := loadRoleSettings(ctx, conn.Conn(), dbe.mainDb.info.Load().ServerVersion, dbe.roleSettings.Load())
	if err != nil {
		return err
	}
	dbe.roleSettings.Store(rs)
	return nil
}
