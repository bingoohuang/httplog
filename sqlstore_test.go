package httplog_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/bingoohuang/httplog"
	"github.com/stretchr/testify/assert"
)

func TestNewSQLStore(t *testing.T) {
	DSN := `root:root@tcp(127.0.0.1:3306)/httplog?charset=utf8mb4&parseTime=true&loc=Local`
	db, err := sql.Open("mysql", DSN)
	assert.Nil(t, err)

	store := httplog.NewSQLStore(db, "")

	mux := httplog.NewMux(http.NewServeMux(), store)
	mux.HandleFunc("/echo", handleIndex, httplog.Biz("回显处理"), httplog.Tables("biz_log"))

	r, _ := http.NewRequest("GET", "/echo", nil)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
}
