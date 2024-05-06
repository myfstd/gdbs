package gdbs

import (
	sysSql "database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/myfstd/gdbs/sqlx"
	"github.com/myfstd/gdbs/types"
	"github.com/myfstd/gdbs/util"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

type Handle struct {
}

func InitMapper(data []byte) error {
	var node types.Node
	if err := xml.Unmarshal(data, &node); err != nil {
		return err
	}
	var item types.SqlItem
	//Select
	mapS := make(map[string]types.SqlVal)
	for _, itemT := range node.Select {
		sv := types.SqlVal{
			Val:     itemT.Val,
			RstTyp:  itemT.RstTyp,
			ParmTyp: itemT.ParmTyp,
			IfItems: itemT.IfItems,
		}
		mapS[itemT.Id] = sv
	}
	item.Select = mapS

	//Insert
	mapI := make(map[string]types.SqlVal)
	for _, itemT := range node.Insert {
		sv := types.SqlVal{
			Val:     itemT.Val,
			RstTyp:  itemT.RstTyp,
			ParmTyp: itemT.ParmTyp,
			IfItems: itemT.IfItems,
		}
		mapI[itemT.Id] = sv
	}
	item.Insert = mapI

	//Update
	mapU := make(map[string]types.SqlVal)
	for _, itemT := range node.Update {
		sv := types.SqlVal{
			Val:     itemT.Val,
			RstTyp:  itemT.RstTyp,
			ParmTyp: itemT.ParmTyp,
			IfItems: itemT.IfItems,
		}
		mapU[itemT.Id] = sv
	}
	item.Update = mapU

	//Delete
	mapD := make(map[string]types.SqlVal)
	for _, itemT := range node.Delete {
		sv := types.SqlVal{
			Val:     itemT.Val,
			RstTyp:  itemT.RstTyp,
			ParmTyp: itemT.ParmTyp,
			IfItems: itemT.IfItems,
		}
		mapD[itemT.Id] = sv
	}
	item.Delete = mapD
	types.SqlCache[node.Names] = item
	return nil
}

//func InitMapper1(xmlUrl string) {
//	var h Handle
//	log.Println("dbs init start.....")
//	filepath.Walk(xmlUrl, func(path string, info fs.FileInfo, err error) error {
//		if !info.IsDir() && strings.EqualFold(filepath.Ext(path), ".xml") {
//			//mapperUrl = strings.TrimSpace(mapperUrl)
//			//if mapperUrl[len(mapperUrl)-1:] != "/" {
//			//	mapperUrl += mapperUrl + "/"
//			//}
//			//f := strings.ReplaceAll(info.Name(), ".xml", ".go")
//			//if _, err := os.Stat(mapperUrl + f); err != nil {
//			//	log.Println(info.Name(), "没有对应的", f+"！")
//			//	panic(err)
//			//}
//			node, err := h.mapperHandleSql(path)
//			if err != nil {
//				panic(err)
//			}
//			var item types.SqlItem
//
//			//Select
//			mapS := make(map[string]types.SqlVal)
//			for _, itemT := range node.Select {
//				sv := types.SqlVal{
//					Val:     itemT.Val,
//					RstTyp:  itemT.RstTyp,
//					ParmTyp: itemT.ParmTyp,
//					IfItems: itemT.IfItems,
//				}
//				mapS[itemT.Id] = sv
//			}
//			item.Select = mapS
//
//			//Insert
//			mapI := make(map[string]types.SqlVal)
//			for _, itemT := range node.Insert {
//				sv := types.SqlVal{
//					Val:     itemT.Val,
//					RstTyp:  itemT.RstTyp,
//					ParmTyp: itemT.ParmTyp,
//					IfItems: itemT.IfItems,
//				}
//				mapI[itemT.Id] = sv
//			}
//			item.Insert = mapI
//
//			//Update
//			mapU := make(map[string]types.SqlVal)
//			for _, itemT := range node.Update {
//				sv := types.SqlVal{
//					Val:     itemT.Val,
//					RstTyp:  itemT.RstTyp,
//					ParmTyp: itemT.ParmTyp,
//					IfItems: itemT.IfItems,
//				}
//				mapU[itemT.Id] = sv
//			}
//			item.Update = mapU
//
//			//Delete
//			mapD := make(map[string]types.SqlVal)
//			for _, itemT := range node.Delete {
//				sv := types.SqlVal{
//					Val:     itemT.Val,
//					RstTyp:  itemT.RstTyp,
//					ParmTyp: itemT.ParmTyp,
//					IfItems: itemT.IfItems,
//				}
//				mapD[itemT.Id] = sv
//			}
//			item.Delete = mapD
//
//			types.SqlCache[node.Names] = item
//		}
//		return nil
//	})
//
//	log.Println("dbs init end.....")
//}
//func (h *Handle) mapperHandleSql(path string) (types.Node, error) {
//
//	data, _ := os.ReadFile(path)
//	var node types.Node
//	if err := xml.Unmarshal(data, &node); err != nil {
//		return types.Node{}, err
//	}
//	fileName := strings.Replace(filepath.Base(path), ".xml", "", 1)
//	if node.Names != fileName {
//		return types.Node{}, errors.New(fmt.Sprintf("文件名[%s]与namespace名[%s]不一致！[", path, node.Names))
//	}
//	return node, nil
//}

func GetOneForMapper(db sqlx.DB, typ interface{}, param ...interface{}) (res interface{}, err error) {
	var h Handle
	//获取调用的函数名称
	if pc, file, _, ok := runtime.Caller(1); ok {
		funName := runtime.FuncForPC(pc).Name()
		funName = filepath.Ext(funName)[1:]
		nameSpace := strings.Replace(filepath.Base(file), filepath.Ext(file), "", 1)
		if item, ok := types.SqlCache[nameSpace]; ok {
			sqlItemSearch, search := item.Select[funName]
			switch true {
			case search:
				sqlItem := util.SqlItemCopy(&sqlItemSearch)
				err = h.mapperSearch(db, 0, typ, *sqlItem, param...)
				return typ, err
			}
		}
	}
	return nil, errors.New("没找到可执行的Sql语句！")
}

func ExecForMapper(db sqlx.DB, typ interface{}, param ...interface{}) (res interface{}, err error) {
	var h Handle
	//获取调用的函数名称
	if pc, file, _, ok := runtime.Caller(1); ok {
		funName := runtime.FuncForPC(pc).Name()
		funName = filepath.Ext(funName)[1:]
		nameSpace := strings.Replace(filepath.Base(file), filepath.Ext(file), "", 1)
		if item, ok := types.SqlCache[nameSpace]; ok {
			sqlItemSearch, search := item.Select[funName]
			sqlItemModify, modify := item.Update[funName]
			sqlItemAdd, add := item.Insert[funName]
			sqlItemRemove, remove := item.Delete[funName]
			switch true {
			case search:
				//sql := sqlItemSearch.Val
				sqlItem := util.SqlItemCopy(&sqlItemSearch)
				err = h.mapperSearch(db, 1, typ, *sqlItem, param...)
				return typ, err
			case modify:
				//sql := sqlItemModify.Val
				sqlItem := util.SqlItemCopy(&sqlItemModify)
				return h.mapperUpdate(db, *sqlItem, param...)
			case add:
				//sql := sqlItemAdd.Val
				sqlItem := util.SqlItemCopy(&sqlItemAdd)
				return h.mapperInsert(db, *sqlItem, param...)
			case remove:
				//sql := sqlItemRemove.Val
				sqlItem := util.SqlItemCopy(&sqlItemRemove)
				return h.mapperDelete(db, *sqlItem, param...)
			}
		}
	}
	return nil, errors.New("没找到可执行的Sql语句！")
}
func (h *Handle) mapperSearch(db sqlx.DB, typ int, res interface{}, sqlItem types.SqlVal, param ...interface{}) error {
	sql := sqlItem.Val
	tempSql := sqlItem.Val
	for _, item := range sqlItem.IfItems {
		tempSql += item.IfVal
	}
	if key := util.FindWordsByKey(tempSql, "@"); key != nil {
		var h Handle
		for _, e := range param {
			ty := reflect.TypeOf(e)
			if ty.Elem().Kind() == reflect.String {
				return errors.New("参数类型错误，直接收struct或map！")
			}
			sql = h.mapperGetRealSql(sqlItem, key, e)
		}
	}
	fmt.Printf("sql=>%s\n", strings.TrimSpace(sql))
	if typ == 0 {
		return db.Get(res, sql)
	}
	return db.Select(res, sql)

}
func (h *Handle) mapperUpdate(db sqlx.DB, sqlItem types.SqlVal, param ...interface{}) (res interface{}, err error) {
	sql := sqlItem.Val
	tempSql := sqlItem.Val
	for _, item := range sqlItem.IfItems {
		tempSql += item.IfVal
	}
	if key := util.FindWordsByKey(tempSql, "@"); key != nil {
		var h Handle
		for _, e := range param {
			ty := reflect.TypeOf(e)
			if ty.Kind() == reflect.String || ty.Elem().Kind() == reflect.String {
				return nil, errors.New("参数类型错误，直接收struct或map！")
			}
			sql = h.mapperGetRealSql(sqlItem, key, e)
		}
	}
	fmt.Printf("sql=>%s\n", strings.TrimSpace(sql))
	r, err := db.Exec(sql)
	if err != nil {
		return nil, err
	}
	id, _ := r.LastInsertId()
	return id, err
}
func (h *Handle) mapperInsert(db sqlx.DB, sqlItem types.SqlVal, param ...interface{}) (res interface{}, err error) {
	sql := sqlItem.Val
	tempSql := sqlItem.Val
	for _, item := range sqlItem.IfItems {
		tempSql += item.IfVal
	}
	//获取调用的函数名称
	if key := util.FindWordsByKey(tempSql, "@"); key != nil {
		var h Handle
		for _, e := range param {
			ty := reflect.TypeOf(e)
			if ty.Kind() == reflect.String || ty.Elem().Kind() == reflect.String {
				return nil, errors.New("参数类型错误，直接收struct或map！")
			}
			sql = h.mapperGetRealSql(sqlItem, key, e)
		}
	}
	sql = util.RepAtWords(sql, "@", "null")
	fmt.Printf("sql=>%s\n", strings.TrimSpace(sql))

	r, err := db.Exec(sql)
	if err != nil {
		return nil, err
	}
	id, _ := r.LastInsertId()
	return id, err
	//return db.Exec(sql)
}
func (h *Handle) mapperDelete(db sqlx.DB, sqlItem types.SqlVal, param ...interface{}) (res interface{}, err error) {
	sql := sqlItem.Val
	tempSql := sqlItem.Val
	for _, item := range sqlItem.IfItems {
		tempSql += item.IfVal
	}
	if key := util.FindWordsByKey(tempSql, "@"); key != nil {
		var h Handle
		for _, e := range param {
			ty := reflect.TypeOf(e)
			if ty.Kind() == reflect.String || ty.Elem().Kind() == reflect.String {
				return nil, errors.New("参数类型错误，直接收struct或map！")
			}
			sql = h.mapperGetRealSql(sqlItem, key, e)
		}
	}
	fmt.Printf("sql=>%s\n", strings.TrimSpace(sql))
	r, err := db.Exec(sql)
	id, _ := r.LastInsertId()
	return id, err
}

func (h *Handle) mapperGetRealSql(sqlItem types.SqlVal, key []string, a interface{}) string {
	t := reflect.TypeOf(a)
	v := reflect.ValueOf(a)
	kind := t.Kind()
	expr := t.Kind()
	if expr == reflect.Pointer {
		expr = t.Elem().Kind()
	}
	switch expr {
	case reflect.Struct:
		for _, s := range key {
			var v1 reflect.Value
			if kind == reflect.Pointer {
				for i := 0; i < t.Elem().NumField(); i++ {
					f := t.Elem().Field(i)
					if strings.EqualFold(f.Name, s[1:]) {
						v1 = v.Elem().FieldByName(f.Name)
						break
					}
				}
			} else {
				for i := 0; i < t.NumField(); i++ {
					f := t.Field(i)
					if strings.EqualFold(f.Name, s[1:]) {
						v1 = v.FieldByName(f.Name)
						break
					}
				}
			}
			if v1.IsValid() {
				//re := regexp.MustCompile("(?i)" + s)
				val := ""
				if v1.Kind() == reflect.Ptr {
					switch reflect.TypeOf(v1.Interface()).Elem().Kind() {
					case reflect.String:
						if v1.Interface().(*string) != nil {
							val = *v1.Interface().(*string)
							sqlItem = *util.ReplaceAll(s, val, &sqlItem)
							//sql = re.ReplaceAllString(sql, "'"+val+"'")
						}
					case reflect.Int:
						if v1.Interface().(*int) != nil {
							val = strconv.Itoa(*v1.Interface().(*int))
							sqlItem = *util.ReplaceAll(s, val, &sqlItem)
							//sql = re.ReplaceAllString(sql, val)
						}
					}
				} else {
					switch reflect.TypeOf(v1.Interface()).Kind() {
					case reflect.String:
						val = v1.Interface().(string)
						sqlItem = *util.ReplaceAll(s, val, &sqlItem)
					case reflect.Int:
						sqlItem = *util.ReplaceAll(s, v1.Interface().(int), &sqlItem)
					case reflect.Slice:
						sqlItem = *util.ReplaceAll(s, v1.Interface(), &sqlItem)
					}
				}

			}
		}
	case reflect.Map:
		for _, s := range key {
			m := v.Interface().(map[string]interface{})
			var v2 any
			for k, n := range m {
				if strings.EqualFold(k, s[1:]) {
					v2 = n
					break
				}
			}
			if v2 != "" {
				sqlItem = *util.ReplaceAll(s, v2, &sqlItem)
			}
		}
	}

	return util.Item2Sql(&sqlItem)
}

func ExecForEntity(db sqlx.DB, doTyp interface{}, entity interface{}) (res interface{}, err error) {
	var h Handle
	//fmt.Println(db, doTyp)
	t := reflect.TypeOf(entity)
	v := reflect.ValueOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	switch t.Kind() {
	case reflect.Slice:
		var rest []interface{}
		for i := 0; i < v.Len(); i++ {
			f := v.Index(i).Addr().Interface()
			if res, err = h.execEntity(db, doTyp, f); err == nil {
				rest = append(rest, res)
			}
		}
		return rest, err
	default:
		return h.execEntity(db, doTyp, entity)
	}
	return nil, errors.New("sql错误")
}
func (h *Handle) execEntity(db sqlx.DB, doTyp interface{}, entity interface{}) (interface{}, error) {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	tnf, bTableName := t.FieldByName("TableName")
	tableName := ""
	if !bTableName {
		return "", errors.New("Entity中没有TableName字段！")
	}
	tableName = tnf.Tag.Get("db")
	if tableName == "" {
		return "", errors.New("Entity的TableName字段没有指定表名称！")
	}
	switch doTyp {
	case KeyIns:
		if sql, err := h.entityInsertSql(tableName, t, entity); err == nil {
			r, err := db.Exec(sql)
			id, err := r.LastInsertId()
			return id, err
		}
	case KeyUp:
		if sql, err := h.entityUpdateSql(tableName, t, entity); err == nil {
			//return db.Exec(sql)
			r, err := db.Exec(sql)
			id, err := r.LastInsertId()
			return id, err
		}
	case KeyDel:
		if sql, err := h.entityDeleteSql(tableName, t, entity); err == nil {
			//return db.Exec(sql)
			r, err := db.Exec(sql)
			id, err := r.LastInsertId()
			return id, err
		}
	case KeySel: //TODO:无法反射类型 未完
		if sql, err := h.entitySearchSql(tableName, t, entity); err == nil {
			res := reflect.MakeSlice(reflect.SliceOf(t), 0, 10)
			err1 := db.Select(&res, sql)
			if err1 != nil {
				return nil, err1
			} else {
				return res, nil
			}
		}
	}
	return nil, errors.New("entity或内容错误！")
}
func (h *Handle) entityInsertSql(table string, t reflect.Type, entity interface{}) (string, error) {
	sql := "insert into " + table
	var sqlItem []string
	var sqlVal []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := reflect.ValueOf(entity).Elem().FieldByName(f.Name)
		if !v.IsValid() || v.Kind() == reflect.Ptr && v.IsNil() {
			continue
		}
		if !v.IsValid() || v.Kind() == reflect.String && len(v.String()) == 0 {
			continue
		}
		if !v.IsValid() || v.Kind() == reflect.Int && v.Int() == 0 {
			continue
		}
		var kind reflect.Kind
		var st string
		if f.Type.Kind() == reflect.Ptr {
			//st = v.Elem().tString()
			st = fmt.Sprintf("%v", v.Elem())
			kind = f.Type.Elem().Kind()
		} else {
			kind = f.Type.Kind()
			st = fmt.Sprintf("%v", v)
		}
		st = strings.ReplaceAll(st, "'", "''")

		switch kind {
		case reflect.Int:
			sqlVal = append(sqlVal, st)
		default:
			sqlVal = append(sqlVal, "'"+st+"'")
		}
		sqlItem = append(sqlItem, f.Tag.Get("db"))
	}
	if sqlItem == nil || sqlVal == nil {
		return "", errors.New("entity项目错误！")
	}
	strItem := strings.Replace(strings.Trim(fmt.Sprint(sqlItem), "[]"), " ", ",", -1)
	strVal := strings.Replace(strings.Trim(fmt.Sprint(sqlVal), "[]"), " ", ",", -1)
	sql = sql + "(" + strItem + ") values(" + strVal + ")"
	fmt.Println(sql)
	return sql, nil
}
func (h *Handle) entityUpdateSql(table string, t reflect.Type, entity interface{}) (string, error) {
	sql := "update  " + table
	sqlSet := " set "
	sqlWhere := " where "
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := reflect.ValueOf(entity).Elem().FieldByName(f.Name)
		if !v.IsValid() || v.Kind() == reflect.Ptr && v.IsNil() {
			continue
		}
		if !v.IsValid() || v.Kind() == reflect.String && len(v.String()) == 0 {
			continue
		}
		if !v.IsValid() || v.Kind() == reflect.Int && v.Int() == 0 {
			continue
		}
		var kind reflect.Kind
		var st string
		if f.Type.Kind() == reflect.Ptr {
			st = fmt.Sprintf("%v", v.Elem())
			kind = f.Type.Elem().Kind()
		} else {
			kind = f.Type.Kind()
			st = fmt.Sprintf("%v", v)
		}
		st = strings.ReplaceAll(st, "'", "''")
		//switch f.Type.tString() {
		//case "int", "*int":
		//if f.Tag.tGet("updateKey") != "" {
		//	sqlWhere = sqlWhere + f.Tag.tGet("db") + "=" + st + " and "
		//} else {
		//	sqlSet = sqlSet + f.Tag.tGet("db") + "=" + st + ","
		//}
		//case "string", "*string":
		//	if f.Tag.tGet("updateKey") != "" {
		//		sqlWhere = sqlWhere + f.Tag.tGet("db") + "='" + st + "' and "
		//	} else {
		//		sqlSet = sqlSet + f.Tag.tGet("db") + "='" + st + "',"
		//	}
		//case "time.Time", "*time.Time":
		//	//stime, _ := time.Parse("2006-01-02 15:04:05", st)
		//	stime := time.Now().Format("2006-01-02 15:04:05")
		//	if f.Tag.tGet("updateKey") != "" {
		//		sqlWhere = sqlWhere + f.Tag.tGet("db") + "='" + stime + "' and "
		//	} else {
		//		sqlSet = sqlSet + f.Tag.tGet("db") + "='" + stime + "',"
		//	}
		switch kind {
		case reflect.Int:
			if st != "0" {
				if f.Tag.Get("updateKey") != "" {
					sqlWhere = sqlWhere + f.Tag.Get("db") + "=" + st + " and "
				} else {
					sqlSet = sqlSet + f.Tag.Get("db") + "=" + st + ","
				}
			}
		default:
			if st != "" {
				if f.Tag.Get("updateKey") != "" {
					sqlWhere = sqlWhere + f.Tag.Get("db") + "='" + st + "' and "
				} else {
					sqlSet = sqlSet + f.Tag.Get("db") + "='" + st + "',"
				}
			}
		}
	}

	if sqlSet == " set " || sqlWhere == " where " {
		return "", errors.New("entity值错误！")
	}
	sqlSet = sqlSet[:len(sqlSet)-1]
	sqlWhere = sqlWhere[:len(sqlWhere)-4]
	return sql + sqlSet + sqlWhere, nil
}
func (h *Handle) entityDeleteSql(table string, t reflect.Type, entity interface{}) (string, error) {
	sql := "delete from " + table
	sqlWhere := " where "
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := reflect.ValueOf(entity).Elem().FieldByName(f.Name)
		if !v.IsValid() || v.Kind() == reflect.Ptr && v.IsNil() {
			continue
		}
		if !v.IsValid() || v.Kind() == reflect.String && len(v.String()) == 0 {
			continue
		}
		if !v.IsValid() || v.Kind() == reflect.Int && v.Int() == 0 {
			continue
		}
		var kind reflect.Kind
		var st string
		if f.Type.Kind() == reflect.Ptr {
			st = fmt.Sprintf("%v", v.Elem())
			kind = f.Type.Elem().Kind()
		} else {
			kind = f.Type.Kind()
			st = fmt.Sprintf("%v", v)
		}
		st = strings.ReplaceAll(st, "'", "''")
		switch kind {
		case reflect.Int:
			sqlWhere = sqlWhere + f.Tag.Get("db") + "=" + st + " and "
		default:
			sqlWhere = sqlWhere + f.Tag.Get("db") + "='" + st + "' and "
		}
	}

	if sqlWhere == " where " {
		return "", errors.New("entity值错误！")
	}
	sqlWhere = sqlWhere[:len(sqlWhere)-4]
	return sql + sqlWhere, nil
}
func (h *Handle) entitySearchSql(table string, t reflect.Type, entity interface{}) (string, error) {
	sql := "select * from " + table
	sqlWhere := ""
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := reflect.ValueOf(entity).Elem().FieldByName(f.Name)
		if !v.IsValid() || v.Kind() == reflect.Ptr && v.IsNil() {
			continue
		}
		if !v.IsValid() || v.Kind() == reflect.String && len(v.String()) == 0 {
			continue
		}
		if !v.IsValid() || v.Kind() == reflect.Int && v.Int() == 0 {
			continue
		}
		var kind reflect.Kind
		var st string
		if f.Type.Kind() == reflect.Ptr {
			st = fmt.Sprintf("%v", v.Elem())
			kind = f.Type.Elem().Kind()
		} else {
			kind = f.Type.Kind()
			st = fmt.Sprintf("%v", v)
		}
		st = strings.ReplaceAll(st, "'", "''")
		switch kind {
		case reflect.Int:
			sqlWhere = sqlWhere + f.Tag.Get("db") + "=" + st + " and "
		default:
			sqlWhere = sqlWhere + f.Tag.Get("db") + "='" + st + "' and "
		}
	}

	if sqlWhere != "" {
		sqlWhere = " where " + sqlWhere[:len(sqlWhere)-4]
	}
	return sql + sqlWhere, nil
}

func ExecForSql(db sqlx.DB, sql string, resTyp ...interface{}) (res interface{}, err error) {
	if len(resTyp) > 1 {
		return nil, errors.New("res参数错误，最多支持1个！")
	}
	key := strings.Fields(sql)[0]
	switch strings.ToLower(key) {
	case KeyIns, KeyDel, KeyUp:
		r, err := db.Exec(sql)
		rv, _ := r.RowsAffected()
		return rv, err
	case KeySel:
		typ := reflect.TypeOf(resTyp[0])
		if typ.Kind() != reflect.Ptr {
			return nil, errors.New("接收参数不是地址！")
		}
		switch typ.Elem().Kind() {
		case reflect.Slice:
			err = db.Select(resTyp[0], sql)
			return resTyp[0], err
		case reflect.Struct:
			err := db.Get(resTyp[0], sql)
			if errors.Is(err, sysSql.ErrNoRows) {
				return nil, nil
			}
			return resTyp[0], err
		}

	}
	return nil, errors.New("sql错误")
}

//func Select(db sqlx.DB, res interface{}, param ...interface{}) error {
//	//获取调用的函数名称
//	if pc, file, _, ok := runtime.Caller(1); ok {
//		funName := runtime.FuncForPC(pc).Name()
//		funName = filepath.Ext(funName)[1:]
//		nameSpace := strings.Replace(filepath.Base(file), filepath.Ext(file), "", 1)
//		if item, ok := types.SqlCache[nameSpace]; ok {
//			sqlItem := item.Select[funName]
//			sql := sqlItem.Val
//			if key := gutil.FindWordsByKey(sql, "@"); key != nil {
//				var h Handle
//				for _, e := range param {
//					ty := reflect.TypeOf(e)
//					if ty.Elem().Kind() == reflect.tString {
//						return errors.New("参数类型错误，直接收struct或map！")
//					}
//					sql = h.mapperGetRealSql(sql, key, e)
//				}
//			}
//			fmt.Printf("sql=>%s\n", strings.TrimSpace(sql))
//			return db.Select(res, sql)
//		}
//	}
//	return errors.New("没找到可执行的Sql语句！")
//}
//func Update(db sqlx.DB, param ...interface{}) (res interface{}, err error) {
//	//获取调用的函数名称
//	if pc, file, _, ok := runtime.Caller(1); ok {
//		funName := runtime.FuncForPC(pc).Name()
//		funName = filepath.Ext(funName)[1:]
//		nameSpace := strings.Replace(filepath.Base(file), filepath.Ext(file), "", 1)
//		if item, ok := types.SqlCache[nameSpace]; ok {
//			sqlItem := item.Update[funName]
//			sql := sqlItem.Val
//			if key := gutil.FindWordsByKey(sql, "@"); key != nil {
//				var h Handle
//				for _, e := range param {
//					ty := reflect.TypeOf(e)
//					if ty.Kind() == reflect.tString || ty.Elem().Kind() == reflect.tString {
//						return nil, errors.New("参数类型错误，直接收struct或map！")
//					}
//					sql = h.mapperGetRealSql(sql, key, e)
//				}
//			}
//			fmt.Printf("sql=>%s\n", strings.TrimSpace(sql))
//			r, err := db.Exec(sql)
//			rows, _ := r.RowsAffected()
//			return rows, err
//		}
//	}
//	return nil, errors.New("没找到可执行的Sql语句！")
//}
//func Insert(db sqlx.DB, param ...interface{}) (res interface{}, err error) {
//	//获取调用的函数名称
//	if pc, file, _, ok := runtime.Caller(1); ok {
//		funName := runtime.FuncForPC(pc).Name()
//		funName = filepath.Ext(funName)[1:]
//		nameSpace := strings.Replace(filepath.Base(file), filepath.Ext(file), "", 1)
//		if item, ok := types.SqlCache[nameSpace]; ok {
//			sqlItem := item.Insert[funName]
//			sql := sqlItem.Val
//			if key := gutil.FindWordsByKey(sql, "@"); key != nil {
//				var h Handle
//				for _, e := range param {
//					ty := reflect.TypeOf(e)
//					if ty.Kind() == reflect.tString || ty.Elem().Kind() == reflect.tString {
//						return nil, errors.New("参数类型错误，直接收struct或map！")
//					}
//					sql = h.mapperGetRealSql(sql, key, e)
//				}
//			}
//			fmt.Printf("sql=>%s\n", strings.TrimSpace(sql))
//			return db.Exec(sql)
//
//		}
//	}
//	return nil, errors.New("没找到可执行的Sql语句！")
//}
//func Delete(db sqlx.DB, param ...interface{}) (res interface{}, err error) {
//	//获取调用的函数名称
//	if pc, file, _, ok := runtime.Caller(1); ok {
//		funName := runtime.FuncForPC(pc).Name()
//		funName = filepath.Ext(funName)[1:]
//		nameSpace := strings.Replace(filepath.Base(file), filepath.Ext(file), "", 1)
//		if item, ok := types.SqlCache[nameSpace]; ok {
//			sqlItem := item.Delete[funName]
//			sql := sqlItem.Val
//			if key := gutil.FindWordsByKey(sql, "@"); key != nil {
//				var h Handle
//				for _, e := range param {
//					ty := reflect.TypeOf(e)
//					if ty.Kind() == reflect.tString || ty.Elem().Kind() == reflect.tString {
//						return nil, errors.New("参数类型错误，直接收struct或map！")
//					}
//					sql = h.mapperGetRealSql(sql, key, e)
//				}
//			}
//			fmt.Printf("sql=>%s\n", strings.TrimSpace(sql))
//			r, err := db.Exec(sql)
//			rows, _ := r.RowsAffected()
//			return rows, err
//
//		}
//	}
//	return nil, errors.New("没找到可执行的Sql语句！")
//}
