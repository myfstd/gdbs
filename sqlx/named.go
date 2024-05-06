package sqlx

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"unicode"

	"github.com/myfstd/gdbs/sqlx/reflectx"
)

type NamedStmt struct {
	Params      []string
	QueryString string
	Stmt        *Stmt
}

func (n *NamedStmt) Close() error {
	return n.Stmt.Close()
}
func (n *NamedStmt) Exec(arg interface{}) (sql.Result, error) {
	args, err := bindAnyArgs(n.Params, arg, n.Stmt.Mapper)
	if err != nil {
		return *new(sql.Result), err
	}
	return n.Stmt.Exec(args...)
}
func (n *NamedStmt) Query(arg interface{}) (*sql.Rows, error) {
	args, err := bindAnyArgs(n.Params, arg, n.Stmt.Mapper)
	if err != nil {
		return nil, err
	}
	return n.Stmt.Query(args...)
}
func (n *NamedStmt) QueryRow(arg interface{}) *Row {
	args, err := bindAnyArgs(n.Params, arg, n.Stmt.Mapper)
	if err != nil {
		return &Row{err: err}
	}
	return n.Stmt.QueryRowx(args...)
}
func (n *NamedStmt) MustExec(arg interface{}) sql.Result {
	res, err := n.Exec(arg)
	if err != nil {
		panic(err)
	}
	return res
}
func (n *NamedStmt) Queryx(arg interface{}) (*Rows, error) {
	r, err := n.Query(arg)
	if err != nil {
		return nil, err
	}
	return &Rows{Rows: r, Mapper: n.Stmt.Mapper, unsafe: isUnsafe(n)}, err
}
func (n *NamedStmt) QueryRowx(arg interface{}) *Row {
	return n.QueryRow(arg)
}
func (n *NamedStmt) Select(dest interface{}, arg interface{}) error {
	rows, err := n.Queryx(arg)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanAll(rows, dest, false)
}
func (n *NamedStmt) Get(dest interface{}, arg interface{}) error {
	r := n.QueryRowx(arg)
	return r.scanAny(dest, false)
}
func (n *NamedStmt) Unsafe() *NamedStmt {
	r := &NamedStmt{Params: n.Params, Stmt: n.Stmt, QueryString: n.QueryString}
	r.Stmt.unsafe = true
	return r
}

type namedPreparer interface {
	Preparer
	binder
}

func prepareNamed(p namedPreparer, query string) (*NamedStmt, error) {
	bindType := BindType(p.DriverName())
	q, args, err := compileNamedQuery([]byte(query), bindType)
	if err != nil {
		return nil, err
	}
	stmt, err := Preparex(p, q)
	if err != nil {
		return nil, err
	}
	return &NamedStmt{
		QueryString: q,
		Params:      args,
		Stmt:        stmt,
	}, nil
}
func convertMapStringInterface(v interface{}) (map[string]interface{}, bool) {
	var m map[string]interface{}
	mtype := reflect.TypeOf(m)
	t := reflect.TypeOf(v)
	if !t.ConvertibleTo(mtype) {
		return nil, false
	}
	return reflect.ValueOf(v).Convert(mtype).Interface().(map[string]interface{}), true
}
func bindAnyArgs(names []string, arg interface{}, m *reflectx.Mapper) ([]interface{}, error) {
	if maparg, ok := convertMapStringInterface(arg); ok {
		return bindMapArgs(names, maparg)
	}
	return bindArgs(names, arg, m)
}
func bindArgs(names []string, arg interface{}, m *reflectx.Mapper) ([]interface{}, error) {
	arglist := make([]interface{}, 0, len(names))
	v := reflect.ValueOf(arg)
	for v = reflect.ValueOf(arg); v.Kind() == reflect.Ptr; {
		v = v.Elem()
	}
	err := m.TraversalsByNameFunc(v.Type(), names, func(i int, t []int) error {
		if len(t) == 0 {
			return fmt.Errorf("could not find name %s in %#v", names[i], arg)
		}
		val := reflectx.FieldByIndexesReadOnly(v, t)
		arglist = append(arglist, val.Interface())
		return nil
	})
	return arglist, err
}
func bindMapArgs(names []string, arg map[string]interface{}) ([]interface{}, error) {
	arglist := make([]interface{}, 0, len(names))
	for _, name := range names {
		val, ok := arg[name]
		if !ok {
			return arglist, fmt.Errorf("could not find name %s in %#v", name, arg)
		}
		arglist = append(arglist, val)
	}
	return arglist, nil
}
func bindStruct(bindType int, query string, arg interface{}, m *reflectx.Mapper) (string, []interface{}, error) {
	bound, names, err := compileNamedQuery([]byte(query), bindType)
	if err != nil {
		return "", []interface{}{}, err
	}
	arglist, err := bindAnyArgs(names, arg, m)
	if err != nil {
		return "", []interface{}{}, err
	}
	return bound, arglist, nil
}

var valuesReg = regexp.MustCompile(`\)\s*(?i)VALUES\s*\(`)

func findMatchingClosingBracketIndex(s string) int {
	count := 0
	for i, ch := range s {
		if ch == '(' {
			count++
		}
		if ch == ')' {
			count--
			if count == 0 {
				return i
			}
		}
	}
	return 0
}

func fixBound(bound string, loop int) string {
	loc := valuesReg.FindStringIndex(bound)
	if len(loc) < 2 {
		return bound
	}
	openingBracketIndex := loc[1] - 1
	index := findMatchingClosingBracketIndex(bound[openingBracketIndex:])
	if index == 0 {
		return bound
	}
	closingBracketIndex := openingBracketIndex + index + 1
	var buffer bytes.Buffer
	buffer.WriteString(bound[0:closingBracketIndex])
	for i := 0; i < loop-1; i++ {
		buffer.WriteString(",")
		buffer.WriteString(bound[openingBracketIndex:closingBracketIndex])
	}
	buffer.WriteString(bound[closingBracketIndex:])
	return buffer.String()
}
func bindArray(bindType int, query string, arg interface{}, m *reflectx.Mapper) (string, []interface{}, error) {
	bound, names, err := compileNamedQuery([]byte(query), QUESTION)
	if err != nil {
		return "", []interface{}{}, err
	}
	arrayValue := reflect.ValueOf(arg)
	arrayLen := arrayValue.Len()
	if arrayLen == 0 {
		return "", []interface{}{}, fmt.Errorf("length of array is 0: %#v", arg)
	}
	var arglist = make([]interface{}, 0, len(names)*arrayLen)
	for i := 0; i < arrayLen; i++ {
		elemArglist, err := bindAnyArgs(names, arrayValue.Index(i).Interface(), m)
		if err != nil {
			return "", []interface{}{}, err
		}
		arglist = append(arglist, elemArglist...)
	}
	if arrayLen > 1 {
		bound = fixBound(bound, arrayLen)
	}
	if bindType != QUESTION {
		bound = Rebind(bindType, bound)
	}
	return bound, arglist, nil
}
func bindMap(bindType int, query string, args map[string]interface{}) (string, []interface{}, error) {
	bound, names, err := compileNamedQuery([]byte(query), bindType)
	if err != nil {
		return "", []interface{}{}, err
	}
	arglist, err := bindMapArgs(names, args)
	return bound, arglist, err
}

var allowedBindRunes = []*unicode.RangeTable{unicode.Letter, unicode.Digit}

func compileNamedQuery(qs []byte, bindType int) (query string, names []string, err error) {
	names = make([]string, 0, 10)
	rebound := make([]byte, 0, len(qs))
	inName := false
	last := len(qs) - 1
	currentVar := 1
	name := make([]byte, 0, 10)
	for i, b := range qs {
		if b == ':' {
			if inName && i > 0 && qs[i-1] == ':' {
				rebound = append(rebound, ':')
				inName = false
				continue
			} else if inName {
				err = errors.New("unexpected `:` while reading named param at " + strconv.Itoa(i))
				return query, names, err
			}
			inName = true
			name = []byte{}
		} else if inName && i > 0 && b == '=' && len(name) == 0 {
			rebound = append(rebound, ':', '=')
			inName = false
			continue
		} else if inName && (unicode.IsOneOf(allowedBindRunes, rune(b)) || b == '_' || b == '.') && i != last {
			name = append(name, b)
		} else if inName {
			inName = false
			if i == last && unicode.IsOneOf(allowedBindRunes, rune(b)) {
				name = append(name, b)
			}
			names = append(names, string(name))
			switch bindType {
			case NAMED:
				rebound = append(rebound, ':')
				rebound = append(rebound, name...)
			case QUESTION, UNKNOWN:
				rebound = append(rebound, '?')
			case DOLLAR:
				rebound = append(rebound, '$')
				for _, b := range strconv.Itoa(currentVar) {
					rebound = append(rebound, byte(b))
				}
				currentVar++
			case AT:
				rebound = append(rebound, '@', 'p')
				for _, b := range strconv.Itoa(currentVar) {
					rebound = append(rebound, byte(b))
				}
				currentVar++
			}
			if i != last {
				rebound = append(rebound, b)
			} else if !unicode.IsOneOf(allowedBindRunes, rune(b)) {
				rebound = append(rebound, b)
			}
		} else {
			rebound = append(rebound, b)
		}
	}
	return string(rebound), names, err
}
func BindNamed(bindType int, query string, arg interface{}) (string, []interface{}, error) {
	return bindNamedMapper(bindType, query, arg, mapper())
}
func Named(query string, arg interface{}) (string, []interface{}, error) {
	return bindNamedMapper(QUESTION, query, arg, mapper())
}
func bindNamedMapper(bindType int, query string, arg interface{}, m *reflectx.Mapper) (string, []interface{}, error) {
	t := reflect.TypeOf(arg)
	k := t.Kind()
	switch {
	case k == reflect.Map && t.Key().Kind() == reflect.String:
		m, ok := convertMapStringInterface(arg)
		if !ok {
			return "", nil, fmt.Errorf("sqlx.bindNamedMapper: unsupported map type: %T", arg)
		}
		return bindMap(bindType, query, m)
	case k == reflect.Array || k == reflect.Slice:
		return bindArray(bindType, query, arg, m)
	default:
		return bindStruct(bindType, query, arg, m)
	}
}
func NamedQuery(e Ext, query string, arg interface{}) (*Rows, error) {
	q, args, err := bindNamedMapper(BindType(e.DriverName()), query, arg, mapperFor(e))
	if err != nil {
		return nil, err
	}
	return e.Queryx(q, args...)
}
func NamedExec(e Ext, query string, arg interface{}) (sql.Result, error) {
	q, args, err := bindNamedMapper(BindType(e.DriverName()), query, arg, mapperFor(e))
	if err != nil {
		return nil, err
	}
	return e.Exec(q, args...)
}
