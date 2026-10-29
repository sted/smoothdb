package database

import (
	"container/list"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The variables of a request are its role, its claims, its access mode and
// the settings of its role. PostgREST sets them with set_config(..., true) in
// the transaction of the request (Query/PreQuery.hs, txVarQuery), so they end
// with it and the connection goes back to the pool clean, with no reset.
//
// smoothdb defers them to the first statement the request sends: they travel
// in front of it, transaction-local, in the same pgx batch. With
// TransactionMode "none" the scope is the implicit transaction of the batch
// (one Sync), so every batch of the request carries them again; otherwise the
// batch begins the request's transaction and they last until its end. A
// request that sends no statement (it fails on its own parameters first, as
// in PostgREST, where the plan comes before the transaction) takes no round
// trip at all.
//
// pgx prepares a statement it has not cached yet in a Sync of its own,
// before the batch, so before the role is in force: as the authenticator,
// which may not even reach the schema (USAGE is checked at the parse). So the
// variables ride with a statement only when the connection already has it
// prepared (statementCache); otherwise they are applied on their own first
// (flush): in the request's transaction, or session-level with
// TransactionMode "none", reset when the connection goes back to the pool.
// The same flush serves every path that takes the bare connection (GetConn).
type requestVars struct {
	role, claims string
	readOnly     bool
	rc           *roleConfig
	// keep is the session that keeps the connection between its requests
	// ("role" mode): there the variables are applied session-level, once
	keep    string
	applied bool
}

// PrepareRequest gets the connection of the request in ctx (FillContext) ready
// for role and claims, and for a read-only request if readOnly. keep is the
// session that keeps the connection after the request, if any ("role" mode).
// With deferred the variables wait for the first statement of the request
// (see requestVars), otherwise they are applied now, as PrepareConnection does.
func PrepareRequest(ctx context.Context, role, claims string, readOnly bool, keep string, deferred bool) error {
	gi := GetSmoothContext(ctx)
	conn := gi.Conn
	needTX := dbe.config.TransactionMode != "none"
	if keep != "" && !needTX {
		if mark, ok := conn.PgConn().CustomData()[sessionMarkKey].(string); ok && mark == keep {
			// the session's own connection, already switched: only the access
			// mode may differ
			return setSessionReadOnly(ctx, conn, readOnly)
		}
	}
	if role == "" {
		// no role to switch to (a library user): the connection as it is
		return prepareConnection(ctx, conn, role, claims, true, readOnly)
	}
	gi.vars = &requestVars{role: role, claims: claims, readOnly: readOnly, rc: roleConfigOf(role), keep: keep}
	if !deferred {
		_, err := gi.conn(ctx)
		return err
	}
	return nil
}

// Transactional says whether every request runs in a transaction of its own
// (a TransactionMode other than "none"): its variables end with it, and no
// connection can keep them between requests.
func Transactional() bool {
	return dbe.config.TransactionMode != "none"
}

// PrepareNow applies the deferred variables of the request in ctx, for a
// handler that is about to use the connection outside the query executor.
func PrepareNow(ctx context.Context) error {
	_, err := GetSmoothContext(ctx).conn(ctx)
	return err
}

// pending answers the variables still to send with the request's statements.
func (gi *SmoothContext) pending() *requestVars {
	if gi.vars == nil || gi.vars.applied {
		return nil
	}
	return gi.vars
}

// conn returns the connection with the request's variables in force,
// applying them on their own if they are still pending.
func (gi *SmoothContext) conn(ctx context.Context) (*DbConn, error) {
	v := gi.pending()
	if v == nil {
		return gi.Conn, nil
	}
	conn := gi.Conn
	if dbe.config.TransactionMode != "none" {
		b := &pgx.Batch{}
		b.Queue(v.begin())
		v.queueLocal(b, false)
		if err := sendAll(ctx, conn, b); err != nil {
			return nil, err
		}
		v.applied = true
		return conn, nil
	}
	// session-level. A connection that still carries a role (another
	// session's) loses it first: the settings of that role are not all
	// overwritten by this one's.
	if err := resetSession(ctx, conn); err != nil {
		return nil, err
	}
	var err error
	if v.rc != nil {
		_, err = conn.Exec(ctx, v.rc.sqlAuto, append(v.rc.autoArgs[:len(v.rc.autoArgs):len(v.rc.autoArgs)], v.role, v.claims, onOff(v.readOnly))...)
	} else {
		_, err = conn.Exec(ctx, prepareAuto, v.role, v.claims, onOff(v.readOnly))
	}
	if err != nil {
		return nil, err
	}
	markSession(conn, v.keep)
	v.applied = true
	return conn, nil
}

// begin is the BEGIN of the request's transaction.
func (v *requestVars) begin() string {
	begin := "BEGIN"
	if v.rc != nil && v.rc.isolation != "" {
		begin += " ISOLATION LEVEL " + v.rc.isolation
	}
	if v.readOnly {
		begin += " READ ONLY"
	}
	return begin
}

// queueLocal queues the transaction-local variables, with the access mode if
// withAccessMode (a batch without a BEGIN of its own).
func (v *requestVars) queueLocal(b *pgx.Batch, withAccessMode bool) {
	var args []any
	if v.rc != nil {
		args = append(args, v.rc.txArgs...)
	}
	args = append(args, v.role, v.claims)
	switch {
	case withAccessMode && v.rc != nil:
		b.Queue(v.rc.sqlLocal, append(args, onOff(v.readOnly))...)
	case withAccessMode:
		b.Queue(prepareLocal, append(args, onOff(v.readOnly))...)
	case v.rc != nil:
		b.Queue(v.rc.sqlTX, args...)
	default:
		b.Queue(prepareTX, args...)
	}
}

// sendAll sends b and reads every result, with the first error.
func sendAll(ctx context.Context, conn *DbConn, b *pgx.Batch) error {
	br := conn.SendBatch(ctx, b)
	for range b.QueuedQueries {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return err
		}
	}
	return br.Close()
}

// requestBatch is a batch of the request's statements, behind the variables
// they need when those are not on the connection yet.
type requestBatch struct {
	pgx.Batch
	lead   int  // the statements of the variables, before the request's own
	commit bool // a transaction of the batch's own, to commit after them
	tx     bool // the batch begins the request's transaction
}

// newBatch starts the batch of the request's statements sqls (the empty ones
// are ignored), with the variables in front when every statement is prepared
// on the connection already; otherwise the variables are applied first, on
// their own.
func (gi *SmoothContext) newBatch(ctx context.Context, sqls ...string) (*requestBatch, error) {
	b := &requestBatch{}
	v := gi.pending()
	if v == nil {
		return b, nil
	}
	if v.keep != "" || !gi.prepared(sqls) {
		_, err := gi.conn(ctx)
		return b, err
	}
	switch {
	case dbe.config.TransactionMode != "none":
		b.Queue(v.begin())
		v.queueLocal(&b.Batch, false)
		b.lead, b.tx = 2, true
	case v.rc != nil && v.rc.isolation != "":
		// an isolation is set before the first query of a transaction, which
		// the variables already are: a transaction of the batch's own
		b.Queue(v.begin())
		v.queueLocal(&b.Batch, false)
		b.lead, b.commit = 2, true
	default:
		v.queueLocal(&b.Batch, true)
		b.lead = 1
	}
	return b, nil
}

// send sends the batch and reads the results of the variables in front of
// the request's statements, whose results come next.
func (gi *SmoothContext) send(ctx context.Context, b *requestBatch) (pgx.BatchResults, error) {
	if b.commit {
		b.Queue("COMMIT")
	}
	br := gi.Conn.SendBatch(ctx, &b.Batch)
	for i := 0; i < b.lead; i++ {
		_, err := br.Exec()
		if err == nil {
			continue
		}
		br.Close()
		var pre pgx.ErrPreprocessingBatch
		var pgErr *pgconn.PgError
		if !errors.As(err, &pre) || !errors.As(err, &pgErr) {
			gi.rollback(ctx, b)
			return nil, err
		}
		// the server refused to prepare a statement the connection had
		// dropped (its cache is not ours), as the authenticator: nothing ran,
		// so the variables go first on their own, and the statements again
		// behind them, prepared as the role this time
		gi.forget(b.QueuedQueries[b.lead:])
		if _, err := gi.conn(ctx); err != nil {
			return nil, err
		}
		own := &requestBatch{}
		for _, q := range b.QueuedQueries[b.lead:] {
			if !b.commit || q.SQL != "COMMIT" {
				own.Queue(q.SQL, q.Arguments...)
			}
		}
		*b = *own
		return gi.Conn.SendBatch(ctx, &b.Batch), nil
	}
	if b.tx {
		gi.vars.applied = true
	}
	return br, nil
}

// end reads what is left of the batch (the COMMIT of its own transaction) and
// closes it; err is the error of the request's statements, if any. The
// statements of a batch that ran are prepared on the connection from now on.
func (gi *SmoothContext) end(ctx context.Context, br pgx.BatchResults, b *requestBatch, err error) error {
	if err == nil && b.commit {
		_, err = br.Exec()
	}
	if cerr := br.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		gi.rollback(ctx, b)
		return err
	}
	gi.remember(b.QueuedQueries[b.lead:])
	return nil
}

// rollback ends the batch's own transaction left open by an error.
func (gi *SmoothContext) rollback(ctx context.Context, b *requestBatch) {
	if b.commit && gi.Conn.PgConn().TxStatus() != 'I' {
		rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		gi.Conn.Exec(rctx, "ROLLBACK")
	}
}

// exec runs a statement without rows, behind the pending variables if any.
func (gi *SmoothContext) exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	b, err := gi.newBatch(ctx, sql)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	if b.lead == 0 {
		tag, err := gi.Conn.Exec(ctx, sql, args...)
		if err == nil {
			gi.remember1(sql)
		}
		return tag, err
	}
	b.Queue(sql, args...)
	br, err := gi.send(ctx, b)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	tag, err := br.Exec()
	return tag, gi.end(ctx, br, b, err)
}

// How the connections of a pool prepare the statements of a batch, after its
// query exec mode: in a Sync of their own before the batch when they cache
// prepared statements (or their descriptions), up to the capacity of the
// cache; never (exec, simple protocol); or always (describe exec).
const (
	alwaysOutOfBatch = -1
	neverOutOfBatch  = 0
)

func statementsInBatch(config *pgx.ConnConfig) int {
	switch config.DefaultQueryExecMode {
	case pgx.QueryExecModeCacheStatement:
		return config.StatementCacheCapacity
	case pgx.QueryExecModeCacheDescribe:
		return config.DescriptionCacheCapacity
	case pgx.QueryExecModeExec, pgx.QueryExecModeSimpleProtocol:
		return neverOutOfBatch
	}
	return alwaysOutOfBatch
}

// statementsKey names, in the custom data of a connection, its statementCache.
const statementsKey = "smoothdb.statements"

// statementCache is the statements a connection has prepared, as far as the
// requests know: the ones that ran on it, most recent first, up to half the
// capacity of pgx's own cache (whose recency follows the same uses, plus a
// few of statements outside the requests). A statement pgx dropped all the
// same fails the batch before anything runs, and send starts over.
type statementCache struct {
	order *list.List
	byKey map[string]*list.Element
	size  int
}

func (gi *SmoothContext) statements() *statementCache {
	if gi.Db == nil || gi.Db.statements <= 0 || gi.vars == nil || gi.vars.keep != "" {
		// no batch of this request carries variables: a request without
		// them, or of a session that keeps its connection prepared
		return nil
	}
	data := gi.Conn.PgConn().CustomData()
	sc, _ := data[statementsKey].(*statementCache)
	if sc == nil {
		sc = &statementCache{order: list.New(), byKey: map[string]*list.Element{}, size: max(gi.Db.statements/2, 1)}
		data[statementsKey] = sc
	}
	return sc
}

// prepared says whether every statement of sqls is prepared on the
// connection.
func (gi *SmoothContext) prepared(sqls []string) bool {
	if gi.Db != nil && gi.Db.statements == neverOutOfBatch {
		return true
	}
	sc := gi.statements()
	if sc == nil {
		return false
	}
	for _, s := range sqls {
		if s == "" {
			continue
		}
		e, ok := sc.byKey[s]
		if !ok {
			return false
		}
		sc.order.MoveToFront(e)
	}
	return true
}

func (gi *SmoothContext) remember(qs []*pgx.QueuedQuery) {
	for _, q := range qs {
		gi.remember1(q.SQL)
	}
}

func (gi *SmoothContext) remember1(sql string) {
	sc := gi.statements()
	if sc == nil || sql == "COMMIT" {
		return
	}
	if e, ok := sc.byKey[sql]; ok {
		sc.order.MoveToFront(e)
		return
	}
	sc.byKey[sql] = sc.order.PushFront(sql)
	if sc.order.Len() > sc.size {
		last := sc.order.Back()
		sc.order.Remove(last)
		delete(sc.byKey, last.Value.(string))
	}
}

func (gi *SmoothContext) forget(qs []*pgx.QueuedQuery) {
	sc := gi.statements()
	if sc == nil {
		return
	}
	for _, q := range qs {
		if e, ok := sc.byKey[q.SQL]; ok {
			sc.order.Remove(e)
			delete(sc.byKey, q.SQL)
		}
	}
}
