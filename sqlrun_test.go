package httplog_test

import (
	"database/sql"
	"testing"

	"github.com/bingoohuang/httplog"
	"github.com/stretchr/testify/assert"
)

func TestSQLQuery(t *testing.T) {
	db, err := sql.Open("mysql", DSN)
	assert.Nil(t, err)

	run := httplog.NewSQLRun(db, &httplog.MapPreparer{})
	result := run.DoQuery("select 1")
	assert.Nil(t, result.Error)
	assert.Equal(t, [][]string{{"1"}}, result.Rows)
	assert.Equal(t, []string{"1"}, result.Headers)
	assert.True(t, result.IsQuery)
}
