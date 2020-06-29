# httplog

[![Travis CI](https://img.shields.io/travis/bingoohuang/httplog/master.svg?style=flat-square)](https://travis-ci.com/bingoohuang/httplog)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/bingoohuang/httplog/blob/master/LICENSE.md)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/bingoohuang/httplog)
[![Coverage Status](http://codecov.io/github/bingoohuang/httplog/coverage.svg?branch=master)](http://codecov.io/github/bingoohuang/httplog?branch=master)
[![goreport](https://www.goreportcard.com/badge/github.com/bingoohuang/httplog)](https://www.goreportcard.com/report/github.com/bingoohuang/httplog)

httplog golang version. see [java version](https://github.com/gobars/httplog)

## Usage

### Import

`go get github.com/bingoohuang/httplog`

### Standard http wrapper

```go
mux := httplog.NewMux(http.NewServeMux(), httplog.NewLogrusStore())
mux.HandleFunc("/echo", handleIndex, httplog.Biz("回显处理"))
mux.HandleFunc("/json", handleJSON, httplog.Biz("JSON处理"))
mux.HandleFunc("/ignored", handleIgnore, httplog.Ignore(true))
mux.HandleFunc("/noname", handleNoname)

server := http.Server{Addr: ":8080", Handler: mux}
log.Fatal(server.ListenAndServe())
```

### Gin wrapper

```go
router := httplog.NewGin(gin.New(), httplog.NewLogrusStore())

router.GET("/hello/:name", ctler.Hello, httplog.Biz("你好"))
router.GET("/bypass/:name", ctler.Bypass, httplog.Ignore(true))

// 监听运行于 0.0.0.0:8080
router.Run(":8080")
```

### save log to SQL database

```go
DSN := `root:root@tcp(127.0.0.1:3306)/httplog?charset=utf8mb4&parseTime=true&loc=Local`
db, _ := sql.Open("mysql", DSN)
store := httplog.NewSQLStore(db, "")
router := httplog.NewGin(gin.New(), store)

router.GET("/hello/:name", ctler.Hello, httplog.Biz("你好"), httplog.Tables("biz_log"))
router.GET("/bypass/:name", ctler.Bypass, httplog.Ignore(true))

// 监听运行于 0.0.0.0:8080
router.Run(":8080")
```

### Prepare log tables

业务日志表定义，根据具体业务需要，必须字段为主键`id`（名字固定）, 示例: [mysql](testdata/mysql.sql)

<details>
  <summary>
    <p>日志表建表规范</p>
  </summary>

字段注释包含| 或者字段名 | 说明
---|---|---
内置类:||
`httplog:"id"`|id| 日志记录ID
`httplog:"created"`|created| 创建时间
`httplog:"ip"` |ip|当前机器IP
`httplog:"hostname"` |hostname|当前机器名称
`httplog:"pid"` |pid|应用程序PID
`httplog:"started"` |start|开始时间(yyyy-MM-dd HH:mm:ss.SSS)
`httplog:"end"` |end|结束时间(yyyy-MM-dd HH:mm:ss.SSS)
`httplog:"cost"` |cost|花费时间（ms)
`httplog:"biz"` |biz|业务名称，对应到HttpLog注解的biz
请求类:||
`httplog:"req_head_xxx"` |req_head_xxx|请求中的xxx头
`httplog:"req_heads"` |req_heads|请求中的所有头
`httplog:"req_method"` |req_method|请求method
`httplog:"req_url"` |req_url|请求URL
`httplog:"req_path_xxx"` |req_path_xxx|请求URL中的xxx路径参数
`httplog:"req_paths"` |req_paths|请求URL中的所有路径参数
`httplog:"req_query_xxx"` |req_query_xxx|请求URl中的xxx查询参数
`httplog:"req_queries"` |req_queries|请求URl中的所有查询参数
`httplog:"req_param_xxx"` |req_param_xxx|请求中query/form的xxx参数
`httplog:"req_params"` |req_params|请求中query/form的所有参数
`httplog:"req_body"` |req_body|请求体
`httplog:"req_json"` |req_json|请求体（当Content-Type为JSON时)
`httplog:"req_json_xxx"` |req_json_xxx|请求体JSON中的xxx属性
响应类:||
`httplog:"rsp_head_xxx"` |rsp_head_xxx|响应中的xxx头
`httplog:"rsp_heads"` |rsp_heads|响应中的所有头
`httplog:"rsp_body"` |rsp_body|响应体
`httplog:"rsp_json"` |rsp_json|响应体JSON（当Content-Type为JSON时)
`httplog:"rsp_json_xxx"`|rsp_json_xxx| 请求体JSON中的xxx属性
`httplog:"rsp_status"`|rsp_status| 响应编码
上下文:||
`httplog:"ctx_xxx"` |ctx_xxx|上下文对象xxx的值, 通过api设置: `httplog.PutAttr(r, "xxx", "yyy")`
</details>

### Ctrler examples

```go
type Ctrler struct {
	Hello  gin.HandlerFunc `route:"GET /hello/:name" name:"你好"`
	Bypass gin.HandlerFunc `route:"POST /bypass/:name" ignore:"true"`
}

func main) {
	ctler := Ctrler{
		Hello: func(context *gin.Context) {
			context.String(200, "welcome "+context.Param("name"))
		},
		Bypass: func(context *gin.Context) {
			context.String(200, "welcome "+context.Param("name"))
		},
	}

	router := httplog.NewGin(gin.New(), &httplog.LogrusStore{})
	router.RegisterCtler(ctler)
    // 监听运行于 0.0.0.0:8080
    router.Run(":8080")
}
```

## Scripts

```
curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"username":"xyz","password":"xyz"}' \
  http://localhost:8100
```

## Resources

1. [Logging HTTP requests in Go](https://presstige.io/p/Logging-HTTP-requests-in-Go-233de7fe59a747078b35b82a1b035d36)
1. [httpsnoop provides an easy way to capture http related metrics (i.e. response time, bytes written, and http status code) from your application's http.Handlers.](https://github.com/felixge/httpsnoop)
