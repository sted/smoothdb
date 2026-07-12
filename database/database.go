package database

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DatabaseInfo struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

type DatabaseUpdate struct {
	Name  *string `json:"name"`
	Owner *string `json:"owner"`
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
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Force text format for tsvector — pgx v5.9+ defaults to binary,
		// but we decode RawValues() as text.
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "tsvector",
			OID:   pgtype.TSVectorOID,
			Codec: &pgtype.TextFormatOnlyCodec{Codec: pgtype.TSVectorCodec{}},
		})
		var set string
		var err error
		if len(dbe.config.SchemaSearchPath) != 0 {
			set = "set_config('search_path', '"
			for i, schema := range dbe.config.SchemaSearchPath {
				if i != 0 {
					set += ", "
				}
				set += schema
			}
			set += "', false)"
			_, err = conn.Exec(ctx, "SELECT "+set)
			if err != nil {
				return err
			}
		}
		info := db.info.Load()
		if info == nil {
			return nil
		}
		for _, t := range info.cachedComposites {
			var fields []pgtype.CompositeCodecField
			for _, oid := range t.SubTypeIds {
				dt, ok := conn.TypeMap().TypeForOID(oid)
				if !ok {
					//return fmt.Errorf("unknown composite type field OID: %v", oid)
					continue
				}
				fields = append(fields, pgtype.CompositeCodecField{Name: dt.Name, Type: dt})
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
	conn, err := db.pool.Acquire(ctx)
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
	return nil
}

func (db *Database) AcquireConnection(ctx context.Context) (*pgxpool.Conn, error) {
	return db.pool.Acquire(ctx)
}
