package httplog

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
	"sync"
)

// nolint:gochecknoglobals
var (
	sqlDriverNamesByType = map[reflect.Type]string{}
	sqlDriverNamesOnce   = sync.Once{}
)

// LookupDriverName get driverName from the driver instance.
// The database/sql API doesn't provide a way to get the registry name for
// a driver from the driver type.
// from https://github.com/golang/go/issues/12600
func LookupDriverName(driver driver.Driver) string {
	sqlDriverNamesOnce.Do(func() {
		for _, d := range sql.Drivers() {
			// Tested empty string DSN with MySQL, PostgreSQL, and SQLite3 drivers.
			if db, _ := sql.Open(d, ""); db != nil {
				sqlDriverNamesByType[reflect.TypeOf(db.Driver())] = d
			}
		}
	})

	return sqlDriverNamesByType[reflect.TypeOf(driver)]
}
