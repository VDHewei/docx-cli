package docxlib

import (
	"reflect"
	"unsafe"

	"github.com/gomutex/godocx/docx"
	"github.com/gomutex/godocx/wml/ctypes"
)

// unsafeGetTableCT returns a pointer to the underlying ctypes.Table of a docx.Table.
// It uses reflect+unsafe to access the unexported 'ct' field.
func unsafeGetTableCT(tbl *docx.Table) *ctypes.Table {
	v := reflect.ValueOf(tbl).Elem()
	f := v.FieldByName("ct")
	ptr := unsafe.Pointer(f.UnsafeAddr())
	return (*ctypes.Table)(ptr)
}
