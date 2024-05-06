package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
)

func ConnectContext(ctx context.Context, driverName, dataSourceName string) (*DB, error) {
	db, err := Open(driverName, dataSourceName)
	if err != nil {
		return db, err
	}
	err = db.PingContext(ctx)
	return db, err
}

type QueryerContext interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*Rows, error)
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *Row
}
type PreparerContext interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
type ExecerContext interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}
type ExtContext interface {
	binder
	QueryerContext
	ExecerContext
}

func SelectContext(ctx context.Context, q QueryerContext, dest interface{}, query string, args ...interface{}) error {
	rows, err := q.QueryxContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanAll(rows, dest, false)
}
func PreparexContext(ctx context.Context, p PreparerContext, query string) (*Stmt, error) {
	s, err := p.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &Stmt{Stmt: s, unsafe: isUnsafe(p), Mapper: mapperFor(p)}, err
}
func GetContext(ctx context.Context, q QueryerContext, dest interface{}, query string, args ...interface{}) error {
	r := q.QueryRowxContext(ctx, query, args...)
	return r.scanAny(dest, false)
}
func LoadFileContext(ctx context.Context, e ExecerContext, path string) (*sql.Result, error) {
	realpath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	contents, err := ioutil.ReadFile(realpath)
	if err != nil {
		return nil, err
	}
	res, err := e.ExecContext(ctx, string(contents))
	return &res, err
}
func MustExecContext(ctx context.Context, e ExecerContext, query string, args ...interface{}) sql.Result {
	res, err := e.ExecContext(ctx, query, args...)
	if err != nil {
		panic(err)
	}
	return res
}
func (db *DB) PrepareNamedContext(ctx context.Context, query string) (*NamedStmt, error) {
	return prepareNamedContext(ctx, db, query)
}
func (db *DB) NamedQueryContext(ctx context.Context, query string, arg interface{}) (*Rows, error) {
	return NamedQueryContext(ctx, db, query, arg)
}
func (db *DB) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return NamedExecContext(ctx, db, query, arg)
}
func (db *DB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return SelectContext(ctx, db, dest, query, args...)
}
func (db *DB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return GetContext(ctx, db, dest, query, args...)
}
func (db *DB) PreparexContext(ctx context.Context, query string) (*Stmt, error) {
	return PreparexContext(ctx, db, query)
}
func (db *DB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
	r, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{Rows: r, unsafe: db.unsafe, Mapper: db.Mapper}, err
}
func (db *DB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *Row {
	rows, err := db.DB.QueryContext(ctx, query, args...)
	return &Row{rows: rows, err: err, unsafe: db.unsafe, Mapper: db.Mapper}
}
func (db *DB) MustBeginTx(ctx context.Context, opts *sql.TxOptions) *Tx {
	tx, err := db.BeginTxx(ctx, opts)
	if err != nil {
		panic(err)
	}
	return tx
}
func (db *DB) MustExecContext(ctx context.Context, query string, args ...interface{}) sql.Result {
	return MustExecContext(ctx, db, query, args...)
}
func (db *DB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, driverName: db.driverName, unsafe: db.unsafe, Mapper: db.Mapper}, err
}
func (db *DB) Connx(ctx context.Context) (*Conn, error) {
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}

	return &Conn{Conn: conn, driverName: db.driverName, unsafe: db.unsafe, Mapper: db.Mapper}, nil
}
func (c *Conn) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := c.Conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, driverName: c.driverName, unsafe: c.unsafe, Mapper: c.Mapper}, err
}
func (c *Conn) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return SelectContext(ctx, c, dest, query, args...)
}
func (c *Conn) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return GetContext(ctx, c, dest, query, args...)
}
func (c *Conn) PreparexContext(ctx context.Context, query string) (*Stmt, error) {
	return PreparexContext(ctx, c, query)
}
func (c *Conn) QueryxContext(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
	r, err := c.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{Rows: r, unsafe: c.unsafe, Mapper: c.Mapper}, err
}
func (c *Conn) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *Row {
	rows, err := c.Conn.QueryContext(ctx, query, args...)
	return &Row{rows: rows, err: err, unsafe: c.unsafe, Mapper: c.Mapper}
}
func (c *Conn) Rebind(query string) string {
	return Rebind(BindType(c.driverName), query)
}
func (tx *Tx) StmtxContext(ctx context.Context, stmt interface{}) *Stmt {
	var s *sql.Stmt
	switch v := stmt.(type) {
	case Stmt:
		s = v.Stmt
	case *Stmt:
		s = v.Stmt
	case *sql.Stmt:
		s = v
	default:
		panic(fmt.Sprintf("non-statement type %v passed to Stmtx", reflect.ValueOf(stmt).Type()))
	}
	return &Stmt{Stmt: tx.StmtContext(ctx, s), Mapper: tx.Mapper}
}
func (tx *Tx) NamedStmtContext(ctx context.Context, stmt *NamedStmt) *NamedStmt {
	return &NamedStmt{
		QueryString: stmt.QueryString,
		Params:      stmt.Params,
		Stmt:        tx.StmtxContext(ctx, stmt.Stmt),
	}
}
func (tx *Tx) PreparexContext(ctx context.Context, query string) (*Stmt, error) {
	return PreparexContext(ctx, tx, query)
}
func (tx *Tx) PrepareNamedContext(ctx context.Context, query string) (*NamedStmt, error) {
	return prepareNamedContext(ctx, tx, query)
}
func (tx *Tx) MustExecContext(ctx context.Context, query string, args ...interface{}) sql.Result {
	return MustExecContext(ctx, tx, query, args...)
}
func (tx *Tx) QueryxContext(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
	r, err := tx.Tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{Rows: r, unsafe: tx.unsafe, Mapper: tx.Mapper}, err
}
func (tx *Tx) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return SelectContext(ctx, tx, dest, query, args...)
}
func (tx *Tx) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return GetContext(ctx, tx, dest, query, args...)
}
func (tx *Tx) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *Row {
	rows, err := tx.Tx.QueryContext(ctx, query, args...)
	return &Row{rows: rows, err: err, unsafe: tx.unsafe, Mapper: tx.Mapper}
}
func (tx *Tx) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return NamedExecContext(ctx, tx, query, arg)
}
func (s *Stmt) SelectContext(ctx context.Context, dest interface{}, args ...interface{}) error {
	return SelectContext(ctx, &qStmt{s}, dest, "", args...)
}
func (s *Stmt) GetContext(ctx context.Context, dest interface{}, args ...interface{}) error {
	return GetContext(ctx, &qStmt{s}, dest, "", args...)
}
func (s *Stmt) MustExecContext(ctx context.Context, args ...interface{}) sql.Result {
	return MustExecContext(ctx, &qStmt{s}, "", args...)
}
func (s *Stmt) QueryRowxContext(ctx context.Context, args ...interface{}) *Row {
	qs := &qStmt{s}
	return qs.QueryRowxContext(ctx, "", args...)
}
func (s *Stmt) QueryxContext(ctx context.Context, args ...interface{}) (*Rows, error) {
	qs := &qStmt{s}
	return qs.QueryxContext(ctx, "", args...)
}
func (q *qStmt) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return q.Stmt.QueryContext(ctx, args...)
}
func (q *qStmt) QueryxContext(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
	r, err := q.Stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{Rows: r, unsafe: q.Stmt.unsafe, Mapper: q.Stmt.Mapper}, err
}
func (q *qStmt) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *Row {
	rows, err := q.Stmt.QueryContext(ctx, args...)
	return &Row{rows: rows, err: err, unsafe: q.Stmt.unsafe, Mapper: q.Stmt.Mapper}
}
func (q *qStmt) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return q.Stmt.ExecContext(ctx, args...)
}
