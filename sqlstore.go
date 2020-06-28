package httplog

import "database/sql"

type SQLStore struct {
	DB         *sql.DB
	DriverName string
	LogTable   string
}

func NewSQLStore(db *sql.DB, defaultLogTable string) *SQLStore {
	s := &SQLStore{DB: db}
	s.DriverName = LookupDriverName(db.Driver())
	s.LogTable = defaultLogTable

	return s
}

func (s *SQLStore) loadTableSchema(tableName string) {

}

// TableSchema defines the schema of a table
type TableSchema struct {
	ColumnName    string `name:"column_name"`
	ColumnComment string `name:"column_comment"`
	DataType      string `name:"data_type"`
	MaxLength     int    `name:"max_length"`
	ColumnSeq     int    `name:"column_seq"`
}

const mysqlSchemaSQL = `select column_name, column_comment, data_type,
 character_maximum_length max_length, ordinal_position column_seq
 from information_schema.columns
 where table_schema = database()
 and table_name = ?`

// Store stores the log in database like MySQL, InfluxDB, and etc.
func (s *SQLStore) Store(log *Log) {

}
