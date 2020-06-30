package httplog

import (
	"database/sql"
	"reflect"
	"strings"

	"github.com/bingoohuang/strcase"
)

type selectItem interface {
	Type() reflect.Type
	SetField(val reflect.Value)
	SetRoot(root reflect.Value)
}

type structItem struct {
	field *reflect.StructField
	root  reflect.Value
}

func (s *structItem) Type() reflect.Type         { return s.field.Type }
func (s *structItem) SetRoot(root reflect.Value) { s.root = root }
func (s *structItem) SetField(val reflect.Value) {
	s.root.FieldByIndex(s.field.Index).Set(val.Convert(s.field.Type))
}

// Mapping defines the interface for SQL query processing.
type Mapping interface {
	Scan(rowNum int) error
	RowsData() interface{}
}

// MapMapping maps the query rows to maps.
type MapMapping struct {
	columnSize  int
	nullReplace string
	columnLobs  []bool
	columnTypes []*sql.ColumnType
	rows        *sql.Rows
	rowsData    [][]string
}

// RowsData returns the mapped rows data.
func (m *MapMapping) RowsData() interface{} { return m.rowsData }

// Scan scans the rows one by one.
func (m *MapMapping) Scan(rowNum int) error {
	holders := make([]sql.NullString, m.columnSize)
	pointers := make([]interface{}, m.columnSize)

	for i := 0; i < m.columnSize; i++ {
		pointers[i] = &holders[i]
	}

	err := m.rows.Scan(pointers...)
	if err != nil {
		return err
	}

	values := make([]string, m.columnSize)

	for i, h := range holders {
		values[i] = IfElse(h.Valid, h.String, m.nullReplace)

		if h.Valid && m.columnLobs[i] {
			values[i] = "(" + m.columnTypes[i].DatabaseTypeName() + ")"
		}
	}

	m.rowsData = append(m.rowsData, values)

	return nil
}

// Preparer prepares to scan query rows.
type Preparer interface {
	// Prepare prepares to scan query rows.
	Prepare(rows *sql.Rows, columns []string) Mapping
}

// MapPreparer prepares to scan query rows.
type MapPreparer struct {
	// NullReplace is the replacement of null values.
	NullReplace string
}

// Prepare prepares to scan query rows.
func (m *MapPreparer) Prepare(rows *sql.Rows, columns []string) Mapping {
	columnSize := len(columns)
	columnTypes, _ := rows.ColumnTypes()
	columnLobs := make([]bool, columnSize)

	for i := 0; i < columnSize; i++ {
		columnLobs[i] = ContainsFold(columnTypes[i].DatabaseTypeName(), "LOB")
	}

	return &MapMapping{
		columnSize:  columnSize,
		nullReplace: m.NullReplace,
		columnTypes: columnTypes,
		columnLobs:  columnLobs,
		rows:        rows,
		rowsData:    make([][]string, 0),
	}
}

// StructMapper is the the structure to create struct mapping.
type StructMapper struct {
	StructType reflect.Type
}

// StructMapping is the structure for mapping row to a structure.
type StructMapping struct {
	mapFields selectItemSlice
	*StructMapper
	rows     *sql.Rows
	rowsData reflect.Value
}

// Scan scans the query result to fetch the rows one by one.
func (s *StructMapping) Scan(rowNum int) error {
	pointers, structPtr := s.mapFields.ResetDestinations(s.StructMapper)

	err := s.rows.Scan(pointers...)
	if err != nil {
		return err
	}

	for i, field := range s.mapFields {
		if p, ok := pointers[i].(*NullAny); ok {
			field.SetField(p.getVal())
		} else {
			field.SetField(reflect.ValueOf(pointers[i]).Elem())
		}
	}

	s.rowsData = reflect.Append(s.rowsData, structPtr.Elem())

	return nil
}

// RowsData returns the mapped rows data.
func (s *StructMapping) RowsData() interface{} { return s.rowsData.Interface() }

// Prepare prepares to scan query rows.
func (m *StructMapper) Prepare(rows *sql.Rows, columns []string) Mapping {
	return &StructMapping{
		rows:         rows,
		mapFields:    m.newStructFields(columns),
		StructMapper: m,
		rowsData:     reflect.MakeSlice(reflect.SliceOf(m.StructType), 0, 0),
	}
}

func (mapFields selectItemSlice) ResetDestinations(mapper *StructMapper) ([]interface{}, reflect.Value) {
	pointers := make([]interface{}, len(mapFields))
	structPtr := reflect.New(mapper.StructType)

	for i, fv := range mapFields {
		fv.SetRoot(structPtr.Elem())

		if ImplSQLScanner(fv.Type()) {
			pointers[i] = reflect.New(fv.Type()).Interface()
		} else {
			pointers[i] = &NullAny{Type: fv.Type()}
		}
	}

	return pointers, structPtr
}

// ImplType tells src whether it implements target type.
func ImplType(src, target reflect.Type) bool {
	if src == target {
		return true
	}

	if src.Kind() == reflect.Ptr {
		return src.Implements(target)
	}

	if target.Kind() != reflect.Interface {
		return false
	}

	return reflect.PtrTo(src).Implements(target)
}

// 参考 https://github.com/uber-go/dig/blob/master/types.go
// nolint gochecknoglobals
var (
	_sqlScannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
)

// ImplSQLScanner tells t whether it implements sql.Scanner interface.
func ImplSQLScanner(t reflect.Type) bool { return ImplType(t, _sqlScannerType) }

type selectItemSlice []selectItem

// newStructFields creates new struct fields slice.
func (m *StructMapper) newStructFields(columns []string) selectItemSlice {
	mapFields := make(selectItemSlice, len(columns))
	for i, col := range columns {
		mapFields[i] = m.newStructField(col)
	}

	return mapFields
}

// newStructField creates a new struct field.
func (m StructMapper) newStructField(col string) selectItem {
	fv, ok := m.StructType.FieldByNameFunc(func(field string) bool {
		return m.matchesField2Col(field, col)
	})

	if ok {
		return &structItem{field: &fv}
	}

	return nil
}

func (m StructMapper) matchesField2Col(field, col string) bool {
	f, _ := m.StructType.FieldByName(field)
	if v := f.Tag.Get("name"); v != "" && v != "-" {
		return v == col
	}

	eq := strings.EqualFold

	return eq(field, col) || eq(field, strcase.ToCamel(col))
}

// IfElse if else ...
func IfElse(ifCondition bool, ifValue, elseValue string) string {
	if ifCondition {
		return ifValue
	}

	return elseValue
}

// ContainsFold tell if a contains b in case-insensitively.
func ContainsFold(a, b string) bool {
	return strings.Contains(strings.ToUpper(a), strings.ToUpper(b))
}
