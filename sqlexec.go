package httplog

// refer https://yougg.github.io/2017/08/24/用go语言写一个简单的mysql客户端/
import (
	"database/sql"
	"strings"
	"time"
)

// ExecResult defines the result structure of sql execution.
type ExecResult struct {
	Error        error
	CostTime     time.Duration
	Headers      []string
	Rows         interface{} // [][]string
	RowsAffected int64
	LastInsertID int64
}

// SQLExec wraps Exec method.
type SQLExec interface {
	// Exec executes update.
	Exec(query string, args ...interface{}) (sql.Result, error)
	// Query performs query.
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

type SQLExecImpl struct {
	SQLExec
	Mapper Mapper
}

// DoExec executes a SQL.
func (s *SQLExecImpl) DoExec(sqlStr string, maxRows int) ExecResult {
	_, isQuerySQL := IsQuerySQL(sqlStr)

	if isQuerySQL {
		return s.DoQuery(sqlStr, maxRows)
	}

	return s.DoUpdate(sqlStr)
}

// DoQuery does the query.
func (s *SQLExecImpl) DoQuery(sqlStr string, maxRows int) (result ExecResult) {
	start := time.Now()

	defer func() {
		result.CostTime = time.Since(start)
	}()

	rows, err := s.Query(sqlStr)
	if err != nil || rows != nil && rows.Err() != nil {
		if err == nil {
			err = rows.Err()
		}

		result.Error = err

		return result
	}

	columns, err := rows.Columns()
	if err != nil {
		result.Error = err

		return result
	}

	mapping := s.Mapper.Prepare(rows, columns)

	for row := 0; rows.Next() && (maxRows == 0 || row < maxRows); row++ {
		if err := mapping.Scan(); err != nil {
			result.Error = err

			return result
		}
	}

	result.Error = err
	result.Headers = columns
	result.Rows = mapping.RowsData()

	return result
}

// DoUpdate does the update.
func (s *SQLExecImpl) DoUpdate(sqlStr string) ExecResult {
	start := time.Now()
	r, err := s.Exec(sqlStr)

	var (
		affected     int64
		lastInsertID int64
	)

	if r != nil {
		affected, _ = r.RowsAffected()
		lastInsertID, _ = r.LastInsertId()
	}

	return ExecResult{Error: err,
		CostTime:     time.Since(start),
		RowsAffected: affected,
		LastInsertID: lastInsertID,
	}
}

// IsQuerySQL tests a sql is a query or not
func IsQuerySQL(sql string) (string, bool) {
	key := FirstWord(sql)

	switch strings.ToUpper(key) {
	case "INSERT", "DELETE", "UPDATE", "SET", "REPLACE":
		return key, false
	case "SELECT", "SHOW", "DESC", "DESCRIBE", "EXPLAIN":
		return key, true
	default:
		return key, false
	}
}

// FirstWord returns the first word of the SQL statement s.
func FirstWord(s string) string {
	if fields := strings.Fields(strings.TrimSpace(s)); len(fields) > 0 {
		return fields[0]
	}

	return ""
}
