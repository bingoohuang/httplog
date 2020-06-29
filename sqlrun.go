package httplog

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

// MiniDB wraps Exec method.
type MiniDB interface {
	// Exec executes update.
	Exec(query string, args ...interface{}) (sql.Result, error)
	// Query performs query.
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// SQLExec is used to execute only updates.
type SQLExec struct {
	MiniDB
}

// SQLRun is used to execute queries and updates.
type SQLRun struct {
	*SQLExec
	Preparer // required only for query
}

// NewSQLExec creates a new SQLExec for only updates.
func NewSQLExec(db MiniDB) *SQLExec {
	return &SQLExec{MiniDB: db}
}

// NewSQLRun creates a new SQLRun for queries and updates.
func NewSQLRun(db MiniDB, mapper Preparer) *SQLRun {
	return &SQLRun{Preparer: mapper, SQLExec: NewSQLExec(db)}
}

// DoExec executes a SQL.
func (s *SQLRun) DoExec(sqlStr string, maxRows int, vars ...interface{}) ExecResult {
	_, isQuerySQL := IsQuerySQL(sqlStr)

	if isQuerySQL {
		return s.DoQuery(sqlStr, maxRows, vars...)
	}

	return s.DoUpdate(sqlStr, vars...)
}

// DoQuery does the query.
func (s *SQLRun) DoQuery(sqlStr string, maxRows int, vars ...interface{}) (result ExecResult) {
	start := time.Now()

	defer func() {
		result.CostTime = time.Since(start)
	}()

	rows, err := s.Query(sqlStr, vars...)
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

	mapping := s.Preparer.Prepare(rows, columns)

	for row := 0; rows.Next() && (maxRows <= 0 || row < maxRows); row++ {
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
func (s *SQLExec) DoUpdate(sqlStr string, vars ...interface{}) (result ExecResult) {
	start := time.Now()
	r, err := s.Exec(sqlStr, vars...)

	if r != nil {
		result.RowsAffected, _ = r.RowsAffected()
		result.LastInsertID, _ = r.LastInsertId()
	}

	result.Error = err
	result.CostTime = time.Since(start)

	return result
}

// IsQuerySQL tests a sql is a query or not.
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
