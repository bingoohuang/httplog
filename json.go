package httplog

import (
	"net/http"

	"github.com/bingoohuang/strcase"
	jsoniter "github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
)

// nolint
var (
	jsonContentType = []string{"application/json; charset=utf-8"}

	JSONUnmarshal     = jsoniter.Unmarshal
	JSONMarshal       = jsoniter.Marshal
	JSONMarshalIndent = jsoniter.MarshalIndent
)

// JSON contains the given interface object.
type JSON struct {
	Data interface{}
}

// Render (JSON) writes rowsData with custom ContentType.
func (r JSON) Render(w http.ResponseWriter) error {
	return WriteJSON(w, r.Data)
}

// WriteContentType (JSON) writes JSON ContentType.
func (r JSON) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, jsonContentType)
}

// nolint:gochecknoinits
func init() {
	extra.SetNamingStrategy(strcase.ToCamelLower)
}

// WriteJSON marshals the given interface object and writes it with custom ContentType.
func WriteJSON(w http.ResponseWriter, obj interface{}) error {
	writeContentType(w, jsonContentType)
	return jsoniter.NewEncoder(w).Encode(&obj)
}

func writeContentType(w http.ResponseWriter, value []string) {
	header := w.Header()

	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}
