package database

import (
	"context"

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
	info          *SchemaInfo
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
		for _, t := range db.info.cachedComposites {
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
	db.info, err = NewSchemaInfo(c, db)
	if err != nil {
		return err
	}

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

func (db *Database) AcquireConnection(ctx context.Context) (*pgxpool.Conn, error) {
	return db.pool.Acquire(ctx)
}
