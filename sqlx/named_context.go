package sqlx

import (
	"context"
	"database/sql"
)

type namedPreparerContext interface {
	PreparerContext
	binder
}

func prepareNamedContext(ctx context.Context, p namedPreparerContext, query string) (*NamedStmt, error) {
	bindType := BindType(p.DriverName())
	q, args, err := compileNamedQuery([]byte(query), bindType)
	if err != nil {
		return nil, err
	}
	stmt, err := PreparexContext(ctx, p, q)
	if err != nil {
		return nil, err
	}
	return &NamedStmt{
		QueryString: q,
		Params:      args,
		Stmt:        stmt,
	}, nil
}
func (n *NamedStmt) ExecContext(ctx context.Context, arg interface{}) (sql.Result, error) {
	args, err := bindAnyArgs(n.Params, arg, n.Stmt.Mapper)
	if err != nil {
		return *new(sql.Result), err
	}
	return n.Stmt.ExecContext(ctx, args...)
}
func (n *NamedStmt) QueryContext(ctx context.Context, arg interface{}) (*sql.Rows, error) {
	args, err := bindAnyArgs(n.Params, arg, n.Stmt.Mapper)
	if err != nil {
		return nil, err
	}
	return n.Stmt.QueryContext(ctx, args...)
}
func (n *NamedStmt) QueryRowContext(ctx context.Context, arg interface{}) *Row {
	args, err := bindAnyArgs(n.Params, arg, n.Stmt.Mapper)
	if err != nil {
		return &Row{err: err}
	}
	return n.Stmt.QueryRowxContext(ctx, args...)
}
func (n *NamedStmt) MustExecContext(ctx context.Context, arg interface{}) sql.Result {
	res, err := n.ExecContext(ctx, arg)
	if err != nil {
		panic(err)
	}
	return res
}
func (n *NamedStmt) QueryxContext(ctx context.Context, arg interface{}) (*Rows, error) {
	r, err := n.QueryContext(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &Rows{Rows: r, Mapper: n.Stmt.Mapper, unsafe: isUnsafe(n)}, err
}
func (n *NamedStmt) QueryRowxContext(ctx context.Context, arg interface{}) *Row {
	return n.QueryRowContext(ctx, arg)
}
func (n *NamedStmt) SelectContext(ctx context.Context, dest interface{}, arg interface{}) error {
	rows, err := n.QueryxContext(ctx, arg)
	if err != nil {
		return err
	}
	// if something happens here, we want to make sure the rows are Closed
	defer rows.Close()
	return scanAll(rows, dest, false)
}
func (n *NamedStmt) GetContext(ctx context.Context, dest interface{}, arg interface{}) error {
	r := n.QueryRowxContext(ctx, arg)
	return r.scanAny(dest, false)
}
func NamedQueryContext(ctx context.Context, e ExtContext, query string, arg interface{}) (*Rows, error) {
	q, args, err := bindNamedMapper(BindType(e.DriverName()), query, arg, mapperFor(e))
	if err != nil {
		return nil, err
	}
	return e.QueryxContext(ctx, q, args...)
}
func NamedExecContext(ctx context.Context, e ExtContext, query string, arg interface{}) (sql.Result, error) {
	q, args, err := bindNamedMapper(BindType(e.DriverName()), query, arg, mapperFor(e))
	if err != nil {
		return nil, err
	}
	return e.ExecContext(ctx, q, args...)
}
