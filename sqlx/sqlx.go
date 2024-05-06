package sqlx

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/myfstd/gdbs/sqlx/reflectx"
)

var NameMapper = strings.ToLower
var origMapper = reflect.ValueOf(NameMapper)
var mpr *reflectx.Mapper
var mprMu sync.Mutex

func mapper() *reflectx.Mapper {
	mprMu.Lock()
	defer mprMu.Unlock()

	if mpr == nil {
		mpr = reflectx.NewMapperFunc("db", NameMapper)
	} else if origMapper != reflect.ValueOf(NameMapper) {
		mpr = reflectx.NewMapperFunc("db", NameMapper)
		origMapper = reflect.ValueOf(NameMapper)
	}
	return mpr
}
func isScannable(t reflect.Type) bool {
	if reflect.PtrTo(t).Implements(_scannerInterface) {
		return true
	}
	if t.Kind() != reflect.Struct {
		return true
	}
	return len(mapper().TypeMap(t).Index) == 0
}

type ColScanner interface {
	Columns() ([]string, error)
	Scan(dest ...interface{}) error
	Err() error
}
type Queryer interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Queryx(query string, args ...interface{}) (*Rows, error)
	QueryRowx(query string, args ...interface{}) *Row
}
type Execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}
type binder interface {
	DriverName() string
	Rebind(string) string
	BindNamed(string, interface{}) (string, []interface{}, error)
}
type Ext interface {
	binder
	Queryer
	Execer
}
type Preparer interface {
	Prepare(query string) (*sql.Stmt, error)
}

func isUnsafe(i interface{}) bool {
	switch v := i.(type) {
	case Row:
		return v.unsafe
	case *Row:
		return v.unsafe
	case Rows:
		return v.unsafe
	case *Rows:
		return v.unsafe
	case NamedStmt:
		return v.Stmt.unsafe
	case *NamedStmt:
		return v.Stmt.unsafe
	case Stmt:
		return v.unsafe
	case *Stmt:
		return v.unsafe
	case qStmt:
		return v.unsafe
	case *qStmt:
		return v.unsafe
	case DB:
		return v.unsafe
	case *DB:
		return v.unsafe
	case Tx:
		return v.unsafe
	case *Tx:
		return v.unsafe
	case sql.Rows, *sql.Rows:
		return false
	default:
		return false
	}
}

func mapperFor(i interface{}) *reflectx.Mapper {
	switch i := i.(type) {
	case DB:
		return i.Mapper
	case *DB:
		return i.Mapper
	case Tx:
		return i.Mapper
	case *Tx:
		return i.Mapper
	default:
		return mapper()
	}
}

var _scannerInterface = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
var _valuerInterface = reflect.TypeOf((*driver.Valuer)(nil)).Elem()

type Row struct {
	err    error
	unsafe bool
	rows   *sql.Rows
	Mapper *reflectx.Mapper
}

func (r *Row) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	defer r.rows.Close()
	for _, dp := range dest {
		if _, ok := dp.(*sql.RawBytes); ok {
			return errors.New("sql: RawBytes isn't allowed on Row.Scan")
		}
	}
	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	err := r.rows.Scan(dest...)
	if err != nil {
		return err
	}
	if err := r.rows.Close(); err != nil {
		return err
	}
	return nil
}
func (r *Row) Columns() ([]string, error) {
	if r.err != nil {
		return []string{}, r.err
	}
	return r.rows.Columns()
}
func (r *Row) ColumnTypes() ([]*sql.ColumnType, error) {
	if r.err != nil {
		return []*sql.ColumnType{}, r.err
	}
	return r.rows.ColumnTypes()
}
func (r *Row) Err() error {
	return r.err
}

type DB struct {
	*sql.DB
	driverName string
	unsafe     bool
	Mapper     *reflectx.Mapper
}

func NewDb(db *sql.DB, driverName string) *DB {
	return &DB{DB: db, driverName: driverName, Mapper: mapper()}
}
func (db *DB) DriverName() string {
	return db.driverName
}
func Open(driverName, dataSourceName string) (*DB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &DB{DB: db, driverName: driverName, Mapper: mapper()}, err
}
func MustOpen(driverName, dataSourceName string) *DB {
	db, err := Open(driverName, dataSourceName)
	if err != nil {
		panic(err)
	}
	return db
}
func (db *DB) MapperFunc(mf func(string) string) {
	db.Mapper = reflectx.NewMapperFunc("db", mf)
}
func (db *DB) Rebind(query string) string {
	return Rebind(BindType(db.driverName), query)
}
func (db *DB) Unsafe() *DB {
	return &DB{DB: db.DB, driverName: db.driverName, unsafe: true, Mapper: db.Mapper}
}
func (db *DB) BindNamed(query string, arg interface{}) (string, []interface{}, error) {
	return bindNamedMapper(BindType(db.driverName), query, arg, db.Mapper)
}
func (db *DB) NamedQuery(query string, arg interface{}) (*Rows, error) {
	return NamedQuery(db, query, arg)
}
func (db *DB) NamedExec(query string, arg interface{}) (sql.Result, error) {
	return NamedExec(db, query, arg)
}
func (db *DB) Select(dest interface{}, query string, args ...interface{}) error {
	return Select(db, dest, query, args...)
}
func (db *DB) Get(dest interface{}, query string, args ...interface{}) error {
	return Get(db, dest, query, args...)
}
func (db *DB) MustBegin() *Tx {
	tx, err := db.Beginx()
	if err != nil {
		panic(err)
	}
	return tx
}
func (db *DB) Beginx() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx, driverName: db.driverName, unsafe: db.unsafe, Mapper: db.Mapper}, err
}
func (db *DB) Queryx(query string, args ...interface{}) (*Rows, error) {
	r, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{Rows: r, unsafe: db.unsafe, Mapper: db.Mapper}, err
}
func (db *DB) QueryRowx(query string, args ...interface{}) *Row {
	rows, err := db.DB.Query(query, args...)
	return &Row{rows: rows, err: err, unsafe: db.unsafe, Mapper: db.Mapper}
}
func (db *DB) MustExec(query string, args ...interface{}) sql.Result {
	return MustExec(db, query, args...)
}
func (db *DB) Preparex(query string) (*Stmt, error) {
	return Preparex(db, query)
}
func (db *DB) PrepareNamed(query string) (*NamedStmt, error) {
	return prepareNamed(db, query)
}

type Conn struct {
	*sql.Conn
	driverName string
	unsafe     bool
	Mapper     *reflectx.Mapper
}
type Tx struct {
	*sql.Tx
	driverName string
	unsafe     bool
	Mapper     *reflectx.Mapper
}

func (tx *Tx) DriverName() string {
	return tx.driverName
}
func (tx *Tx) Rebind(query string) string {
	return Rebind(BindType(tx.driverName), query)
}
func (tx *Tx) Unsafe() *Tx {
	return &Tx{Tx: tx.Tx, driverName: tx.driverName, unsafe: true, Mapper: tx.Mapper}
}
func (tx *Tx) BindNamed(query string, arg interface{}) (string, []interface{}, error) {
	return bindNamedMapper(BindType(tx.driverName), query, arg, tx.Mapper)
}
func (tx *Tx) NamedQuery(query string, arg interface{}) (*Rows, error) {
	return NamedQuery(tx, query, arg)
}
func (tx *Tx) NamedExec(query string, arg interface{}) (sql.Result, error) {
	return NamedExec(tx, query, arg)
}
func (tx *Tx) Select(dest interface{}, query string, args ...interface{}) error {
	return Select(tx, dest, query, args...)
}
func (tx *Tx) Queryx(query string, args ...interface{}) (*Rows, error) {
	r, err := tx.Tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{Rows: r, unsafe: tx.unsafe, Mapper: tx.Mapper}, err
}
func (tx *Tx) QueryRowx(query string, args ...interface{}) *Row {
	rows, err := tx.Tx.Query(query, args...)
	return &Row{rows: rows, err: err, unsafe: tx.unsafe, Mapper: tx.Mapper}
}
func (tx *Tx) Get(dest interface{}, query string, args ...interface{}) error {
	return Get(tx, dest, query, args...)
}
func (tx *Tx) MustExec(query string, args ...interface{}) sql.Result {
	return MustExec(tx, query, args...)
}
func (tx *Tx) Preparex(query string) (*Stmt, error) {
	return Preparex(tx, query)
}
func (tx *Tx) Stmtx(stmt interface{}) *Stmt {
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
	return &Stmt{Stmt: tx.Stmt(s), Mapper: tx.Mapper}
}
func (tx *Tx) NamedStmt(stmt *NamedStmt) *NamedStmt {
	return &NamedStmt{
		QueryString: stmt.QueryString,
		Params:      stmt.Params,
		Stmt:        tx.Stmtx(stmt.Stmt),
	}
}
func (tx *Tx) PrepareNamed(query string) (*NamedStmt, error) {
	return prepareNamed(tx, query)
}

type Stmt struct {
	*sql.Stmt
	unsafe bool
	Mapper *reflectx.Mapper
}

func (s *Stmt) Unsafe() *Stmt {
	return &Stmt{Stmt: s.Stmt, unsafe: true, Mapper: s.Mapper}
}
func (s *Stmt) Select(dest interface{}, args ...interface{}) error {
	return Select(&qStmt{s}, dest, "", args...)
}
func (s *Stmt) Get(dest interface{}, args ...interface{}) error {
	return Get(&qStmt{s}, dest, "", args...)
}
func (s *Stmt) MustExec(args ...interface{}) sql.Result {
	return MustExec(&qStmt{s}, "", args...)
}
func (s *Stmt) QueryRowx(args ...interface{}) *Row {
	qs := &qStmt{s}
	return qs.QueryRowx("", args...)
}
func (s *Stmt) Queryx(args ...interface{}) (*Rows, error) {
	qs := &qStmt{s}
	return qs.Queryx("", args...)
}

type qStmt struct{ *Stmt }

func (q *qStmt) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return q.Stmt.Query(args...)
}
func (q *qStmt) Queryx(query string, args ...interface{}) (*Rows, error) {
	r, err := q.Stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	return &Rows{Rows: r, unsafe: q.Stmt.unsafe, Mapper: q.Stmt.Mapper}, err
}
func (q *qStmt) QueryRowx(query string, args ...interface{}) *Row {
	rows, err := q.Stmt.Query(args...)
	return &Row{rows: rows, err: err, unsafe: q.Stmt.unsafe, Mapper: q.Stmt.Mapper}
}
func (q *qStmt) Exec(query string, args ...interface{}) (sql.Result, error) {
	return q.Stmt.Exec(args...)
}

type Rows struct {
	*sql.Rows
	unsafe  bool
	Mapper  *reflectx.Mapper
	started bool
	fields  [][]int
	values  []interface{}
}

func (r *Rows) SliceScan() ([]interface{}, error) {
	return SliceScan(r)
}
func (r *Rows) MapScan(dest map[string]interface{}) error {
	return MapScan(r, dest)
}
func (r *Rows) StructScan(dest interface{}) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	v = v.Elem()
	if !r.started {
		columns, err := r.Columns()
		if err != nil {
			return err
		}
		m := r.Mapper
		r.fields = m.TraversalsByName(v.Type(), columns)
		if f, err := missingFields(r.fields); err != nil && !r.unsafe {
			return fmt.Errorf("missing destination name %s in %T", columns[f], dest)
		}
		r.values = make([]interface{}, len(columns))
		r.started = true
	}
	err := fieldsByTraversal(v, r.fields, r.values, true)
	if err != nil {
		return err
	}
	err = r.Scan(r.values...)
	if err != nil {
		return err
	}
	return r.Err()
}
func Connect(driverName, dataSourceName string) (*DB, error) {
	db, err := Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func MustConnect(driverName, dataSourceName string) *DB {
	db, err := Connect(driverName, dataSourceName)
	if err != nil {
		panic(err)
	}
	return db
}
func Preparex(p Preparer, query string) (*Stmt, error) {
	s, err := p.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &Stmt{Stmt: s, unsafe: isUnsafe(p), Mapper: mapperFor(p)}, err
}
func Select(q Queryer, dest interface{}, query string, args ...interface{}) error {
	rows, err := q.Queryx(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanAll(rows, dest, false)
}
func Get(q Queryer, dest interface{}, query string, args ...interface{}) error {
	r := q.QueryRowx(query, args...)
	return r.scanAny(dest, false)
}
func LoadFile(e Execer, path string) (*sql.Result, error) {
	realpath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	contents, err := ioutil.ReadFile(realpath)
	if err != nil {
		return nil, err
	}
	res, err := e.Exec(string(contents))
	return &res, err
}
func MustExec(e Execer, query string, args ...interface{}) sql.Result {
	res, err := e.Exec(query, args...)
	if err != nil {
		panic(err)
	}
	return res
}
func (r *Row) SliceScan() ([]interface{}, error) {
	return SliceScan(r)
}
func (r *Row) MapScan(dest map[string]interface{}) error {
	return MapScan(r, dest)
}
func (r *Row) scanAny(dest interface{}, structOnly bool) error {
	if r.err != nil {
		return r.err
	}
	if r.rows == nil {
		r.err = sql.ErrNoRows
		//r.err = nil
		return r.err
	}
	defer r.rows.Close()

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if v.IsNil() {
		return errors.New("nil pointer passed to StructScan destination")
	}
	base := reflectx.Deref(v.Type())
	scannable := isScannable(base)
	if structOnly && scannable {
		return structOnlyError(base)
	}
	columns, err := r.Columns()
	if err != nil {
		return err
	}
	if scannable && len(columns) > 1 {
		return fmt.Errorf("scannable dest type %s with >1 columns (%d) in result", base.Kind(), len(columns))
	}
	if scannable {
		return r.Scan(dest)
	}
	m := r.Mapper
	fields := m.TraversalsByName(v.Type(), columns)
	if f, err := missingFields(fields); err != nil && !r.unsafe {
		return fmt.Errorf("missing destination name %s in %T", columns[f], dest)
	}
	values := make([]interface{}, len(columns))
	err = fieldsByTraversal(v, fields, values, true)
	if err != nil {
		return err
	}
	return r.Scan(values...)
}
func (r *Row) StructScan(dest interface{}) error {
	return r.scanAny(dest, true)
}
func SliceScan(r ColScanner) ([]interface{}, error) {
	// ignore r.started, since we needn't use reflect for anything.
	columns, err := r.Columns()
	if err != nil {
		return []interface{}{}, err
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(interface{})
	}

	err = r.Scan(values...)

	if err != nil {
		return values, err
	}

	for i := range columns {
		values[i] = *(values[i].(*interface{}))
	}

	return values, r.Err()
}
func MapScan(r ColScanner, dest map[string]interface{}) error {
	columns, err := r.Columns()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(interface{})
	}

	err = r.Scan(values...)
	if err != nil {
		return err
	}

	for i, column := range columns {
		dest[column] = *(values[i].(*interface{}))
	}

	return r.Err()
}

type rowsi interface {
	Close() error
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(...interface{}) error
}

func structOnlyError(t reflect.Type) error {
	isStruct := t.Kind() == reflect.Struct
	isScanner := reflect.PtrTo(t).Implements(_scannerInterface)
	if !isStruct {
		return fmt.Errorf("expected %s but got %s", reflect.Struct, t.Kind())
	}
	if isScanner {
		return fmt.Errorf("structscan expects a struct dest but the provided struct type %s implements scanner", t.Name())
	}
	return fmt.Errorf("expected a struct, but struct %s has no exported fields", t.Name())
}
func scanAll(rows rowsi, dest interface{}, structOnly bool) error {
	var v, vp reflect.Value

	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if value.IsNil() {
		return errors.New("nil pointer passed to StructScan destination")
	}
	direct := reflect.Indirect(value)

	slice, err := baseType(value.Type(), reflect.Slice)
	if err != nil {
		return err
	}
	direct.SetLen(0)

	isPtr := slice.Elem().Kind() == reflect.Ptr
	base := reflectx.Deref(slice.Elem())
	scannable := isScannable(base)

	if structOnly && scannable {
		return structOnlyError(base)
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if scannable && len(columns) > 1 {
		return fmt.Errorf("non-struct dest type %s with >1 columns (%d)", base.Kind(), len(columns))
	}
	if !scannable {
		var values []interface{}
		var m *reflectx.Mapper

		switch rows.(type) {
		case *Rows:
			m = rows.(*Rows).Mapper
		default:
			m = mapper()
		}

		fields := m.TraversalsByName(base, columns)
		if f, err := missingFields(fields); err != nil && !isUnsafe(rows) {
			return fmt.Errorf("missing destination name %s in %T", columns[f], dest)
		}
		values = make([]interface{}, len(columns))
		for rows.Next() {
			vp = reflect.New(base)
			v = reflect.Indirect(vp)
			err = fieldsByTraversal(v, fields, values, true)
			if err != nil {
				return err
			}
			err = rows.Scan(values...)
			if err != nil {
				return err
			}
			if isPtr {
				direct.Set(reflect.Append(direct, vp))
			} else {
				direct.Set(reflect.Append(direct, v))
			}
		}
	} else {
		for rows.Next() {
			vp = reflect.New(base)
			err = rows.Scan(vp.Interface())
			if err != nil {
				return err
			}
			if isPtr {
				direct.Set(reflect.Append(direct, vp))
			} else {
				direct.Set(reflect.Append(direct, reflect.Indirect(vp)))
			}
		}
	}
	return rows.Err()
}
func StructScan(rows rowsi, dest interface{}) error {
	return scanAll(rows, dest, true)
}
func baseType(t reflect.Type, expected reflect.Kind) (reflect.Type, error) {
	t = reflectx.Deref(t)
	if t.Kind() != expected {
		return nil, fmt.Errorf("expected %s but got %s", expected, t.Kind())
	}
	return t, nil
}
func fieldsByTraversal(v reflect.Value, traversals [][]int, values []interface{}, ptrs bool) error {
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return errors.New("argument not a struct")
	}
	for i, traversal := range traversals {
		if len(traversal) == 0 {
			values[i] = new(interface{})
			continue
		}
		f := reflectx.FieldByIndexes(v, traversal)
		if ptrs {
			values[i] = f.Addr().Interface()
		} else {
			values[i] = f.Interface()
		}
	}
	return nil
}
func missingFields(transversals [][]int) (field int, err error) {
	for i, t := range transversals {
		if len(t) == 0 {
			return i, errors.New("missing field")
		}
	}
	return 0, nil
}
