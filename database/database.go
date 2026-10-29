package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

type DatabaseInfo struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

type DatabaseUpdate struct {
	Name  *string `json:"name"`
	Owner *string `json:"owner"`
}

// textFormatOnlyTypes are the types pgx would request in binary format but
// the serializers only read as text (the wire-format rule is documented on
// JSONSerializer). AfterConnect re-registers each of them on every connection
// with a TextFormatOnlyCodec around its default codec, so the server sends
// PostgreSQL's text form of the value and the serializer passes it through as
// to_json would print it; Scan keeps working through the text path of the
// original codec. The array of each type is listed too: an array codec
// captures its element type when pgx builds its default map, so it keeps
// preferring binary otherwise. A type that is neither here nor decoded by the
// serializers' binary switch fails loudly at serialization: that is where a
// new entry starts.
var textFormatOnlyTypes = []uint32{
	pgtype.ByteaOID, pgtype.ByteaArrayOID,
	pgtype.QCharOID, pgtype.QCharArrayOID, // "char"
	pgtype.InetOID, pgtype.InetArrayOID,
	pgtype.CIDROID, pgtype.CIDRArrayOID,
	pgtype.MacaddrOID, pgtype.MacaddrArrayOID,
	pgtype.Macaddr8OID, // pgx has no codec for macaddr8[]: it is text already
	pgtype.TimeOID, pgtype.TimeArrayOID,
	pgtype.PointOID, pgtype.PointArrayOID,
	pgtype.LineOID, pgtype.LineArrayOID,
	pgtype.LsegOID, pgtype.LsegArrayOID,
	pgtype.BoxOID, pgtype.BoxArrayOID,
	pgtype.PathOID, pgtype.PathArrayOID,
	pgtype.PolygonOID, pgtype.PolygonArrayOID,
	pgtype.CircleOID, pgtype.CircleArrayOID,
	pgtype.BitOID, pgtype.BitArrayOID,
	pgtype.VarbitOID, pgtype.VarbitArrayOID,
	pgtype.TIDOID, pgtype.TIDArrayOID,
	pgtype.XIDOID, pgtype.XIDArrayOID,
	pgtype.CIDOID, pgtype.CIDArrayOID,
	pgtype.XID8OID, pgtype.XID8ArrayOID,
	pgtype.TSVectorOID, pgtype.TSVectorArrayOID,
	// xml prefers text on its own, but pgx's array and composite codecs pick
	// binary whenever the element supports it, and the serializers have no
	// binary xml decoder (its binary form is the text, but the switch does
	// not know that): listed so xml[] and composites with an xml field stay
	// in text.
	pgtype.XMLOID, pgtype.XMLArrayOID,
	// The builtin ranges: a range is one JSON string, range_out's text with
	// its quoting of the bounds (only those with a bracket, a comma, a quote,
	// a backslash or whitespace in them, a quote doubled) and PostgreSQL's
	// own text for each bound (a space between date and time, the session
	// time zone for tstzrange), which a decoder of the binary bounds would
	// have to reproduce for every subtype. A custom range type is unknown to
	// pgx and text already.
	pgtype.Int4rangeOID, pgtype.Int4rangeArrayOID,
	pgtype.Int8rangeOID, pgtype.Int8rangeArrayOID,
	pgtype.NumrangeOID, pgtype.NumrangeArrayOID,
	pgtype.DaterangeOID, pgtype.DaterangeArrayOID,
	pgtype.TsrangeOID, pgtype.TsrangeArrayOID,
	pgtype.TstzrangeOID, pgtype.TstzrangeArrayOID,
	// The builtin multiranges, one JSON string each like the ranges
	// (multirange_out's braces around range_out's text of each range, '{}'
	// for the empty one). pgx's multirange codec follows the range codec it
	// captured when its default map was built, so the range entries above do
	// not reach them. pgx registers no codec for their arrays, text already;
	// listed so that a pgx that does keeps them in text.
	pgtype.Int4multirangeOID, pgtype.Int4multirangeArrayOID,
	pgtype.Int8multirangeOID, pgtype.Int8multirangeArrayOID,
	pgtype.NummultirangeOID, pgtype.NummultirangeArrayOID,
	pgtype.DatemultirangeOID, pgtype.DatemultirangeArrayOID,
	pgtype.TsmultirangeOID, pgtype.TsmultirangeArrayOID,
	pgtype.TstzmultirangeOID, pgtype.TstzmultirangeArrayOID,
}

// isoDateStyle returns the DateStyle to ask the server for: ISO output, plus
// the field order (MDY, DMY, YMD) of a DateStyle already in the connection
// parameters (from the URL), whatever the case of its key. The other keys
// spelling it are dropped, so one value reaches the server.
func isoDateStyle(params map[string]string) string {
	order, implied := "", ""
	for key, value := range params {
		if !strings.EqualFold(key, "datestyle") {
			continue
		}
		for _, part := range strings.Split(value, ",") {
			switch p := strings.ToUpper(strings.TrimSpace(part)); p {
			case "MDY", "DMY", "YMD":
				order = p
			case "EURO", "EUROPEAN":
				order = "DMY"
			case "US", "NONEURO", "NONEUROPEAN":
				order = "MDY"
			case "GERMAN": // implies day first, unless an order is given
				implied = "DMY"
			}
		}
		delete(params, key)
	}
	if order == "" {
		order = implied
	}
	if order == "" {
		return "ISO"
	}
	return "ISO, " + order
}

// searchPathSet is the set_config that gives a connection
// Database.SchemaSearchPath, "" when none is configured. It is a session
// setting set with SQL, not a startup parameter, which a pooler such as
// PgBouncer refuses by default; so a RESET does not restore it, and the
// release of an autocommit connection runs this instead (see roleSettings).
func searchPathSet() string {
	if len(dbe.config.SchemaSearchPath) == 0 {
		return ""
	}
	// an escape-string literal, read the same whatever
	// standard_conforming_strings says: a backslash or an apostrophe in a
	// quoted schema name is escaped
	path := strings.Join(dbe.config.SchemaSearchPath, ", ")
	path = strings.NewReplacer(`\`, `\\`, "'", "''").Replace(path)
	return "set_config('search_path', E'" + path + "', false)"
}

type Database struct {
	DatabaseInfo
	activation    chan struct{}
	activationErr error
	pool          *pgxpool.Pool
	// info is the schema cache. It is read by every request goroutine and
	// replaced wholesale on reload, so it is held in an atomic pointer: readers
	// call info.Load(), the reloader calls info.Store(). Never copy Database by
	// value (the atomic makes that unsafe).
	info atomic.Pointer[SchemaInfo]
	// statements says how the connections of the pool prepare the statements
	// of a batch (see statementsInBatch)
	statements int
	// acquiring counts the acquires in progress on the pool: the requests and
	// the internal users that wait for a connection (see Acquiring)
	acquiring atomic.Int32
}

// activate initializes a database and starts its connection pool
func (db *Database) activate(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			db.activationErr = err
		}
		close(db.activation)
	}()

	connString := dbe.config.URL
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return err
	}
	config.ConnConfig.Config.Database = db.Name
	config.MinConns = dbe.config.MinPoolConnections
	config.MaxConns = dbe.config.MaxPoolConnections
	config.ConnConfig.Tracer = dbe.dbtracer
	db.statements = statementsInBatch(config.ConnConfig)
	// The text form of a timestamp or a date (a field of a composite that
	// arrives in text) is converted to what to_json prints on the assumption
	// that it is ISO, the output style pgx assumes as well: pin it at startup
	// (only the output style: the order part, MDY/DMY, that reads ambiguous
	// input, keeps the server's setting).
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["DateStyle"] = isoDateStyle(config.ConnConfig.RuntimeParams)
	// built here: AfterConnect runs on the pool's goroutines, also after the
	// engine that created the pool is replaced (a library or a test that
	// starts another), so it reads nothing of the engine
	searchPath := searchPathSet()
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.PgConn().CustomData()[loginUserKey] = conn.PgConn().ParameterStatus("session_authorization")
		// Force text format for the types the serializers do not decode in
		// binary (pgx v5.9+ defaults to binary for them). Registered before
		// the composites below, so that a composite with such a field is
		// requested in text too (its codec supports binary only when every
		// field does).
		for _, oid := range textFormatOnlyTypes {
			dt, ok := conn.TypeMap().TypeForOID(oid)
			if !ok {
				continue
			}
			conn.TypeMap().RegisterType(&pgtype.Type{
				Name:  dt.Name,
				OID:   oid,
				Codec: &pgtype.TextFormatOnlyCodec{Codec: dt.Codec},
			})
		}
		if searchPath != "" {
			if _, err := conn.Exec(ctx, "SELECT "+searchPath); err != nil {
				return err
			}
		}
		info := db.info.Load()
		if info == nil {
			return nil
		}
		for _, t := range info.cachedComposites {
			var fields []pgtype.CompositeCodecField
			known := true
			for _, oid := range t.SubTypeIds {
				dt, ok := conn.TypeMap().TypeForOID(oid)
				if !ok {
					known = false
					break
				}
				fields = append(fields, pgtype.CompositeCodecField{Name: dt.Name, Type: dt})
			}
			if !known {
				// A field of a type pgx does not know (a custom range, a
				// domain, a composite not registered yet): left unregistered,
				// the composite is requested in text, which the serializers
				// parse at any nesting. Registered without the field, it
				// would be requested in binary and carry that field in a
				// format the serializers cannot decode.
				continue
			}
			conn.TypeMap().RegisterType(&pgtype.Type{Name: t.Name, OID: t.Id, Codec: &pgtype.CompositeCodec{Fields: fields}})
		}
		return nil
	}

	conn, err := pgx.ConnectConfig(ctx, config.ConnConfig)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	c := ContextWithDbConn(context.Background(), db, conn)
	info, err := NewSchemaInfo(c, db)
	if err != nil {
		return err
	}
	db.info.Store(info)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	db.pool = pool

	return nil
}

func (db *Database) Close() {
	db.pool.Close()
}

// @@ to be implemented and used
func (db *Database) refreshTable(ctx context.Context, name string) {
	// table := Table{}
	// table.Columns, _ = db.GetColumns(ctx, name)
	// table.Struct = fieldsToStruct(table.Columns)
	// db.cachedTables[name] = table
}

// ReloadSchemaCache reloads the schema cache without restarting the server
func (db *Database) ReloadSchemaCache(ctx context.Context) error {
	conn, err := db.AcquireConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Create a context with the database connection
	c := ContextWithDbConn(ctx, db, conn.Conn())
	
	// Reload the schema info
	newInfo, err := NewSchemaInfo(c, db)
	if err != nil {
		return fmt.Errorf("failed to reload schema info: %w", err)
	}

	// Atomically replace the old schema info with the new one
	db.info.Store(newInfo)

	// The DDL that motivates a reload also invalidates the plans the
	// pool's connections keep for their cached statements: a view
	// recreated with other columns, or a column added to a table read with
	// RETURNING *, makes Postgres answer the next execution on each of them
	// with 0A000 "cached plan must not change result type". Drop the
	// connections along with the old schema info (what PostgREST does on a
	// schema cache reload): the idle ones close now, the ones in use when
	// released, and the pool stays open.
	db.pool.Reset()
	return nil
}

func (db *Database) AcquireConnection(ctx context.Context) (*pgxpool.Conn, error) {
	if ctx.Value(handOffKey{}) == nil {
		// a session's wait is counted by the session manager itself
		db.acquiring.Add(1)
		defer db.acquiring.Add(-1)
	}
	return db.pool.Acquire(ctx)
}

// PoolOf is the database whose pool serves db (nil is the main database).
func PoolOf(db *Database) *Database {
	if db == nil && dbe != nil {
		return dbe.mainDb
	}
	return db
}

// SamePool says whether a and b take their connections from the same pool.
func SamePool(a, b *Database) bool {
	return PoolOf(a) == PoolOf(b)
}

// Acquiring says whether a connection of db's pool (nil: the main database)
// is being acquired outside the sessions that keep one ("role" mode): such a
// session gives its own back instead, so that it does not idle while a
// request waits. The waits of the sessions are counted by their manager.
func Acquiring(db *Database) bool {
	if db == nil {
		if dbe == nil || dbe.mainDb == nil {
			return false
		}
		db = dbe.mainDb
	}
	return db.acquiring.Load() > 0
}

// Exhausted says whether db's pool (nil: the main database) has no idle
// connection and cannot open another one.
func Exhausted(db *Database) bool {
	if db == nil {
		db = dbe.mainDb
	}
	st := db.pool.Stat()
	return st.IdleConns() == 0 && st.TotalConns() >= st.MaxConns()
}

type handOffKey struct{}

// WithHandOff marks the context of an acquire that a session may answer
// instead of the pool (see authn): its cancellation is not an error to log.
func WithHandOff(ctx context.Context) context.Context {
	return context.WithValue(ctx, handOffKey{}, true)
}

// poolTracer is the engine's tracer, quiet about an acquire cancelled because
// a session handed its connection over.
type poolTracer struct {
	*tracelog.TraceLog
}

func (t poolTracer) TraceAcquireEnd(ctx context.Context, pool *pgxpool.Pool, data pgxpool.TraceAcquireEndData) {
	if errors.Is(data.Err, context.Canceled) && ctx.Value(handOffKey{}) != nil {
		return
	}
	t.TraceLog.TraceAcquireEnd(ctx, pool, data)
}
