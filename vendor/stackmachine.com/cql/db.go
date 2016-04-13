package cql

import (
	"database/sql"

	ot "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
)

type DB struct {
	*sql.DB
}

func Open(driverName, dataSourceName string) (*DB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	return &DB{db}, err
}

type Tx struct {
	*sql.Tx
}

type Stmt struct {
	*sql.Stmt
}

type Row struct {
	ctxerr error
	*sql.Row
}

func (r *Row) Scan(dest ...interface{}) error {
	if r.ctxerr != nil {
		return r.ctxerr
	}
	return r.Row.Scan(dest...)
}

func Wrap(db *sql.DB) *DB {
	return &DB{db}
}

func (db *DB) PingC(ctx context.Context) error {
	span, _ := ot.StartSpanFromContext(ctx, "Ping")
	defer span.Finish()

	result := make(chan error, 1)

	go func() {
		result <- db.Ping()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-result:
		return err
	}
}

func (db *DB) BeginC(ctx context.Context) (*Tx, error) {
	span, _ := ot.StartSpanFromContext(ctx, "BeginC")
	defer span.Finish()

	type txAndError struct {
		tx  *sql.Tx
		err error
	}
	result := make(chan txAndError, 1)

	go func() {
		tx, err := db.Begin()
		result <- txAndError{tx, err}
	}()

	select {
	case <-ctx.Done():
		go func() {
			if r := <-result; r.tx != nil {
				r.tx.Rollback()
			}
		}()
		return nil, ctx.Err()
	case r := <-result:
		return &Tx{r.tx}, r.err
	}
}

type preparer interface {
	Prepare(query string) (*sql.Stmt, error)
}

func prepare(p preparer, ctx context.Context, query string) (*Stmt, error) {
	type stmtAndError struct {
		stmt *sql.Stmt
		err  error
	}
	result := make(chan stmtAndError, 1)

	go func() {
		s, err := p.Prepare(query)
		result <- stmtAndError{s, err}
	}()

	select {
	case <-ctx.Done():
		go func() {
			if r := <-result; r.stmt != nil {
				r.stmt.Close()
			}
		}()
		return nil, ctx.Err()
	case r := <-result:
		return &Stmt{r.stmt}, r.err
	}
}

func (db *DB) PrepareC(ctx context.Context, query string) (*Stmt, error) {
	span, nctx := ot.StartSpanFromContext(ctx, "PrepareC")
	defer span.Finish()
	return prepare(db, nctx, query)
}

func (tx *Tx) PrepareC(ctx context.Context, query string) (*Stmt, error) {
	span, nctx := ot.StartSpanFromContext(ctx, "PrepareC")
	defer span.Finish()
	return prepare(tx, nctx, query)
}

func execute(ctx context.Context, exec func() (sql.Result, error)) (sql.Result, error) {
	type resultAndError struct {
		result sql.Result
		err    error
	}
	result := make(chan resultAndError, 1)

	go func() {
		r, err := exec()
		result <- resultAndError{r, err}
	}()

	select {
	case <-ctx.Done():
		var res sql.Result
		return res, ctx.Err()
	case r := <-result:
		return r.result, r.err
	}
}

func (db *DB) ExecC(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	span, nctx := ot.StartSpanFromContext(ctx, "ExecC")
	defer span.Finish()
	return execute(nctx, func() (sql.Result, error) { return db.Exec(query, args...) })
}

func (s *Stmt) ExecC(ctx context.Context, args ...interface{}) (sql.Result, error) {
	span, nctx := ot.StartSpanFromContext(ctx, "ExecC")
	defer span.Finish()
	return execute(nctx, func() (sql.Result, error) { return s.Exec(args...) })
}

func (tx *Tx) ExecC(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	span, nctx := ot.StartSpanFromContext(ctx, "ExecC")
	defer span.Finish()
	return execute(nctx, func() (sql.Result, error) { return tx.Exec(query, args...) })
}

func query(ctx context.Context, query func() (*sql.Rows, error)) (*sql.Rows, error) {
	type rowsAndError struct {
		rows *sql.Rows
		err  error
	}
	result := make(chan rowsAndError, 1)

	go func() {
		s, err := query()
		result <- rowsAndError{s, err}
	}()

	select {
	case <-ctx.Done():
		go func() {
			if r := <-result; r.rows != nil {
				r.rows.Close()
			}
		}()
		return nil, ctx.Err()
	case r := <-result:
		return r.rows, r.err
	}
}

func (db *DB) QueryC(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
	span, nctx := ot.StartSpanFromContext(ctx, "QueryC")
	defer span.Finish()
	return query(nctx, func() (*sql.Rows, error) { return db.Query(q, args...) })
}

func (s *Stmt) QueryC(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
	span, nctx := ot.StartSpanFromContext(ctx, "QueryC")
	defer span.Finish()
	return query(nctx, func() (*sql.Rows, error) { return s.Query(args...) })
}

func (tx *Tx) QueryC(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
	span, nctx := ot.StartSpanFromContext(ctx, "QueryC")
	defer span.Finish()
	return query(nctx, func() (*sql.Rows, error) { return tx.Query(q, args...) })
}

func queryrow(ctx context.Context, row func() *sql.Row) *Row {
	result := make(chan *sql.Row, 1)

	go func() {
		result <- row()
	}()

	select {
	case <-ctx.Done():
		go func() {
			r := <-result
			// We call Scan here to make sure that the underlying Rows struct is closed
			r.Scan()
		}()
		return &Row{Row: nil, ctxerr: ctx.Err()}
	case r := <-result:
		return &Row{Row: r, ctxerr: nil}
	}
}

func (db *DB) QueryRowC(ctx context.Context, query string, args ...interface{}) *Row {
	span, nctx := ot.StartSpanFromContext(ctx, "QueryRowC")
	defer span.Finish()
	return queryrow(nctx, func() *sql.Row { return db.QueryRow(query, args...) })
}

func (s *Stmt) QueryRowC(ctx context.Context, args ...interface{}) *Row {
	span, nctx := ot.StartSpanFromContext(ctx, "QueryRowC")
	defer span.Finish()
	return queryrow(nctx, func() *sql.Row { return s.QueryRow(args...) })
}

func (tx *Tx) QueryRowC(ctx context.Context, query string, args ...interface{}) *Row {
	span, nctx := ot.StartSpanFromContext(ctx, "QueryRowC")
	defer span.Finish()
	return queryrow(nctx, func() *sql.Row { return tx.QueryRow(query, args...) })
}
