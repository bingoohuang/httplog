package httplog

import (
	"database/sql"
	"reflect"
	"strings"

	"github.com/bingoohuang/strcase"
)

type SelectItem interface {
	Type() reflect.Type
	Set(val reflect.Value)
	ResetParent(parent reflect.Value)
}

type StructItem struct {
	*reflect.StructField
	parent reflect.Value
}

func (s *StructItem) Type() reflect.Type               { return s.StructField.Type }
func (s *StructItem) ResetParent(parent reflect.Value) { s.parent = parent }
func (s *StructItem) Set(val reflect.Value) {
	f := s.parent.FieldByName(s.StructField.Name)
	f.Set(val.Convert(f.Type()))
}

type Mapping interface {
	Scan() error
	RowsData() interface{}
}

type MapMapping struct {
	columnSize  int
	nullReplace string
	columnLobs  []bool
	columnTypes []*sql.ColumnType
	rows        *sql.Rows
	rowsData    [][]string
}

func (m *MapMapping) RowsData() interface{} {
	return m.rowsData
}

func (m *MapMapping) Scan() error {
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

	for i, v := range holders {
		values[i] = IfElse(v.Valid, v.String, m.nullReplace)

		if m.columnLobs[i] && v.Valid {
			values[i] = "(" + m.columnTypes[i].DatabaseTypeName() + ")"
		}
	}

	m.rowsData = append(m.rowsData, values)

	return nil
}

type Mapper interface {
	Prepare(rows *sql.Rows, columns []string) Mapping
}

type MapMapper struct {
	nullReplace string
}

func (m *MapMapper) Prepare(rows *sql.Rows, columns []string) Mapping {
	columnSize := len(columns)
	columnTypes, _ := rows.ColumnTypes()
	columnLobs := make([]bool, columnSize)

	for i := 0; i < len(columnTypes); i++ {
		columnLobs[i] = ContainsIgnoreCase(columnTypes[i].DatabaseTypeName(), "LOB")
	}

	return &MapMapping{
		columnSize:  columnSize,
		nullReplace: m.nullReplace,
		columnTypes: columnTypes,
		columnLobs:  columnLobs,
		rows:        rows,
		rowsData:    make([][]string, 0),
	}
}

type StructMapper struct {
	StructType reflect.Type
}

type StructMapping struct {
	mapFields SelectItemSlice
	*StructMapper
	rows     *sql.Rows
	rowsData []interface{}
}

func (s *StructMapping) Scan() error {
	pointers, structPtr := s.mapFields.ResetDestinations(s.StructMapper)

	err := s.rows.Scan(pointers...)
	if err != nil {
		return err
	}

	for i, field := range s.mapFields {
		if p, ok := pointers[i].(*NullAny); ok {
			field.Set(p.getVal())
		} else {
			field.Set(reflect.ValueOf(pointers[i]).Elem())
		}
	}

	s.rowsData = append(s.rowsData, structPtr.Elem().Interface())

	return nil
}

func (s *StructMapping) RowsData() interface{} {
	return s.rowsData
}

func (m *StructMapper) Prepare(rows *sql.Rows, columns []string) Mapping {
	mapping := &StructMapping{
		rows:         rows,
		mapFields:    m.NewStructFields(columns),
		StructMapper: m,
		rowsData:     make([]interface{}, 0),
	}

	return mapping
}

func (mapFields SelectItemSlice) ResetDestinations(mapper *StructMapper) ([]interface{}, reflect.Value) {
	pointers := make([]interface{}, len(mapFields))
	structPtr := reflect.New(mapper.StructType)

	for i, fv := range mapFields {
		fv.ResetParent(structPtr)

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

type SelectItemSlice []SelectItem

// NewStructFields creates new struct fields slice.
func (m *StructMapper) NewStructFields(columns []string) SelectItemSlice {
	mapFields := make(SelectItemSlice, len(columns))
	for i, col := range columns {
		mapFields[i] = m.NewStructField(col)
	}

	return mapFields
}

// NewStructField creates a new struct field.
func (m StructMapper) NewStructField(col string) SelectItem {
	fv, ok := m.StructType.FieldByNameFunc(func(field string) bool {
		return m.matchesField2Col(field, col)
	})

	if ok {
		return &StructItem{StructField: &fv}
	}

	return nil
}

func (m StructMapper) matchesField2Col(field, col string) bool {
	f, _ := m.StructType.FieldByName(field)
	if tagName := f.Tag.Get("name"); tagName != "" {
		return tagName == col
	}

	return strings.EqualFold(field, col) || strings.EqualFold(field, strcase.ToCamel(col))
}

// IfElse if else ...
func IfElse(ifCondition bool, ifValue, elseValue string) string {
	if ifCondition {
		return ifValue
	}

	return elseValue
}

// ContainsIgnoreCase tell if a contains b in case-insensitively
func ContainsIgnoreCase(a, b string) bool {
	return strings.Contains(strings.ToUpper(a), strings.ToUpper(b))
}
