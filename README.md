##### gdbs说明：

###### 底层采用sqlx处理，在此基础上增加了xml的sql处理逻辑。
###### 提供了如下几种调用方法
###### 1.GetOneForMapper 根据xml的sql获取/更新一条数据。
###### 2.ExecForMapper 根据xml的sql获取/更新多条数据。
###### 3.ExecForEntity 根据Entity的定义获取/更新多条数据。
###### 4.ExecForSql 通过写固定sql获取/更新数据。




##### 使用示例：

###### 1.首先将xml读入内存(代码片段)

```
    names := sqlxml.AssetNames()
	for _, name := range names {
		data, _ := sqlxml.Asset(name)
		dbs.InitMapper(data)
	}
```


###### 2.执行sql，获取数据

```
func (u *userMapper) GetUserPage(param param.UserParam) (any, error) {
	var users []entities.UserExtRole
	return dbs.ExecForMapper(gdb.MysqlDb, &users, &param)
}
```

