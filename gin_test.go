package httplog_test

import (
	"testing"

	"github.com/bingoohuang/httplog"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func loginFilter(c *gin.Context) {
	httplog.PutAttr(c.Request, "username", "bingoohuang")
	c.Next()
}

func TestGin(t *testing.T) {
	router := httplog.NewGin(gin.New(), httplog.NewLogrusStore())

	router.Use(loginFilter)

	group := router.Group("/group")

	group.GET("/hello/:name", ctler.Hello, httplog.Biz("你好"))
	group.GET("/bypass/:name", ctler.Bypass, httplog.Ignore(true))
	group.GET("/bare", ctler.Bypass)

	// 监听运行于 0.0.0.0:8080
	//router.Run(":8080")

	//server := &http.Server{Addr: ":8080", Handler: router}
	//server.ListenAndServe()

	rr := httplog.PerformRequest("GET", "/group/hello/bingoo", router)
	assert.Equal(t, 200, rr.Code)
	assert.Equal(t, "welcome bingoo", rr.Body.String())

	rr = httplog.PerformRequest("GET", "/group/bypass/bingoo", router)
	assert.Equal(t, 200, rr.Code)
	assert.Equal(t, "welcome bingoo", rr.Body.String())

	httplog.PerformRequest("GET", "/bare", router)
}

type Ctrler struct {
	Hello  gin.HandlerFunc `route:"GET /hello/:name" name:"你好"`
	Bypass gin.HandlerFunc `route:"POST /bypass/:name" ignore:"true"`
}

// nolint:gochecknoglobals
var (
	ctler = Ctrler{
		Hello: func(context *gin.Context) {
			context.String(200, "welcome "+context.Param("name"))
		},
		Bypass: func(context *gin.Context) {
			context.String(200, "welcome "+context.Param("name"))
		},
	}
)

func TestCtrler(t *testing.T) {
	router := httplog.NewGin(gin.New(), &httplog.LogrusStore{})
	router.RegisterCtler(ctler)

	rr := httplog.PerformRequest("GET", "/hello/bingoo", router)
	assert.Equal(t, 200, rr.Code)
	assert.Equal(t, "welcome bingoo", rr.Body.String())

	rr = httplog.PerformRequest("POST", "/bypass/bingoo", router)
	assert.Equal(t, 200, rr.Code)
	assert.Equal(t, "welcome bingoo", rr.Body.String())
}
