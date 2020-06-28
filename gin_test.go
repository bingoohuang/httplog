package httplog_test

import (
	"testing"

	"github.com/bingoohuang/httplog"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGin(t *testing.T) {
	router := httplog.NewGin(gin.New(), httplog.NewLogrusStore())

	router.GET("/hello/:name", ctler.Hello, httplog.Name("你好"))
	router.GET("/bypass/:name", ctler.Bypass, httplog.Ignore(true))

	rr := httplog.PerformRequest("GET", "/hello/bingoo", router)
	assert.Equal(t, 200, rr.Code)
	assert.Equal(t, "welcome bingoo", rr.Body.String())

	rr = httplog.PerformRequest("GET", "/bypass/bingoo", router)
	assert.Equal(t, 200, rr.Code)
	assert.Equal(t, "welcome bingoo", rr.Body.String())
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
