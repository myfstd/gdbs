package util

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/myfstd/gdbs/types"
	"github.com/myfstd/geval"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func FindWordsByKey(str string, sta string) []string {
	str = strings.ReplaceAll(str, ",", " , ")
	str = strings.ReplaceAll(str, ")", " ) ")
	str = strings.ReplaceAll(str, "(", " ( ")
	str = strings.ReplaceAll(str, "=", " = ")
	// 构建正则表达式模式
	pattern := fmt.Sprintf("%s\\S+", sta) // ^表示行首，\w+表示任意数量的字母、数字或下划线
	re := regexp.MustCompile(pattern)     // 编译正则表达式
	// 提取所有匹配到的单词
	words := re.FindAllString(str, -1)
	return words
}

func RepAtWords(str string, sta string, target string) string {
	str = strings.ReplaceAll(str, ",", " , ")
	str = strings.ReplaceAll(str, ")", " ) ")
	str = strings.ReplaceAll(str, "(", " ( ")
	str = strings.ReplaceAll(str, "=", " = ")
	re := regexp.MustCompile(`\S+`)
	result := re.ReplaceAllStringFunc(str, func(word string) string {
		if word[0] == sta[0] {
			return target
		} else {
			return word
		}
	})
	result = strings.ReplaceAll(result, " , ", ",")
	result = strings.ReplaceAll(result, " ) ", ")")
	result = strings.ReplaceAll(result, " ( ", "(")
	result = strings.ReplaceAll(result, " = ", "=")
	return result
}

func RemoveNull(slice []string) []string {
	var output []string
	for _, element := range slice {
		if element != "" {
			output = append(output, element)
		}
	}
	return output
}

func ReplaceAll(key string, val interface{}, sqlItem *types.SqlVal) *types.SqlVal {
	switch reflect.TypeOf(val).Kind() {
	case reflect.String:
		val1 := strings.ReplaceAll(val.(string), "'", "")
		sqlItem.Val = strings.ReplaceAll(sqlItem.Val, key, "'"+val1+"'")
	case reflect.Int:
		sqlItem.Val = strings.ReplaceAll(sqlItem.Val, key, fmt.Sprintf("%v", val))
	case reflect.Slice:
		if vs, b := val.([]int); b {
			strNums := make([]string, len(vs))
			for i, num := range vs {
				strNums[i] = strconv.Itoa(num)
			}
			rp := strings.Join(strNums, ",")
			sqlItem.Val = strings.ReplaceAll(sqlItem.Val, key, rp)
		}
		if vs, b := val.([]string); b {
			rp := strings.Join(vs, ",")
			sqlItem.Val = strings.ReplaceAll(sqlItem.Val, key, rp)
		}

	}
	for idx, it := range sqlItem.IfItems {
		if strings.Contains(strings.ToLower(it.Test), strings.ToLower(key[1:])) {
			if !geval.Eval(strings.ReplaceAll(strings.ToLower(it.Test),
				strings.ToLower(key[1:]), fmt.Sprintf("'%v'", val))).(bool) {
				sqlItem.IfItems[idx].IfVal = ""
			} else {
				switch reflect.TypeOf(val).Kind() {
				case reflect.String:
					val1 := strings.ReplaceAll(val.(string), "'", "''")
					sqlItem.IfItems[idx].IfVal = strings.ReplaceAll(it.IfVal, key, "'"+val1+"'")
				case reflect.Int:
					sqlItem.IfItems[idx].IfVal = strings.ReplaceAll(it.IfVal, key, fmt.Sprintf("%v", val))
				case reflect.Slice:
					if vs, b := val.([]int); b {
						strNums := make([]string, len(vs))
						for i, num := range vs {
							strNums[i] = strconv.Itoa(num)
						}
						rp := strings.Join(strNums, ",")
						sqlItem.IfItems[idx].IfVal = strings.ReplaceAll(it.IfVal, key, rp)
					}
					if vs, b := val.([]string); b {
						rp := strings.Join(vs, ",")
						sqlItem.IfItems[idx].IfVal = strings.ReplaceAll(it.IfVal, key, rp)
					}
				}
			}
		}
	}
	return sqlItem
}

func Item2Sql(sqlItem *types.SqlVal) string {
	if sqlItem.IfItems == nil {
		return sqlItem.Val
	}
	sql := ""
	sqlAry := strings.Split(sqlItem.Val, "\n")
	sqlAry = RemoveNull(sqlAry)
	idx := 0
	for _, s := range sqlAry {
		if strings.TrimSpace(s) != "" {
			sql += s
		} else {
			if len(sqlItem.IfItems) > idx {
				sql += " " + sqlItem.IfItems[idx].IfVal
				idx++
			}
		}
	}
	if idx != len(sqlItem.IfItems) {
		log.Println("sql错误！")
	}
	sql = strings.ReplaceAll(sql, "&gt;", ">")
	sql = strings.ReplaceAll(sql, "&lt;", "<")
	return sql
}

func SqlItemCopy(src *types.SqlVal) *types.SqlVal {
	var buf bytes.Buffer
	dst := new(types.SqlVal)
	gob.NewEncoder(&buf).Encode(src)
	gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
	return dst
}
