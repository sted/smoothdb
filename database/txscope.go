package database

import (
	"context"
	"time"
)

// ddlSavepoint names the savepoint of withTx. The helpers that use withTx do
// not nest, so one name is enough.
const ddlSavepoint = "smoothdb_ddl"

// withTx runs fn as one unit on conn: every statement of fn applies, or none.
// On an idle connection it begins and ends a transaction of its own. Inside
// a transaction (the request's, when TransactionMode is not "none") it uses a
// savepoint instead: the request's transaction stays open, with its
// transaction-local role and claims, and the request still decides whether to
// commit or roll back. A failure of fn rolls back the statements of fn only.
// The end runs even when ctx is cancelled, with a bounded timeout.
func withTx(ctx context.Context, conn *DbConn, fn func() error) (err error) {
	begin, commit, rollback := "BEGIN", "COMMIT", "ROLLBACK"
	if conn.PgConn().TxStatus() != 'I' {
		begin = "SAVEPOINT " + ddlSavepoint
		commit = "RELEASE SAVEPOINT " + ddlSavepoint
		rollback = "ROLLBACK TO SAVEPOINT " + ddlSavepoint + "; RELEASE SAVEPOINT " + ddlSavepoint
	}
	if _, err = conn.Exec(ctx, begin); err != nil {
		return err
	}
	ok := false
	defer func() {
		// a panic in fn leaves ok false: roll back
		end := rollback
		if ok {
			end = commit
		}
		endCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if _, endErr := conn.Exec(endCtx, end); err == nil {
			err = endErr
		}
	}()
	err = fn()
	ok = err == nil
	return err
}
