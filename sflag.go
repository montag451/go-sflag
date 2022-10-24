package sflag

import (
	"flag"
	"fmt"
	"reflect"
)

// TagKey is the key used to retrieve the name of the flag in the
// struct field tag.
var TagKey = "flag"

// Set sets the value of the fields in the struct contained in i with
// the value of the flags defined in fs. It used the tag of the struct
// fields to determine the struct fields whose value should be set and
// to determine the corresponding flag to use.
func Set(i any, fs *flag.FlagSet) {
	if !fs.Parsed() {
		panic("flag not parsed")
	}
	v := reflect.Indirect(reflect.ValueOf(i))
	if v.Kind() != reflect.Struct {
		panic("not a struct")
	}
	tags := make(map[string][]int)
	var getTags func(*reflect.Value, []int)
	getTags = func(v *reflect.Value, pindex []int) {
		fields := reflect.VisibleFields(v.Type())
		for _, fi := range fields {
			index := make([]int, len(pindex)+len(fi.Index))
			copy(index, pindex)
			copy(index[len(pindex):], fi.Index)
			if fi.Type.Kind() == reflect.Struct {
				fiv := v.FieldByIndex(fi.Index)
				getTags(&fiv, index)
			} else {
				tag := fi.Tag.Get(TagKey)
				if tag != "" {
					if _, ok := tags[tag]; ok {
						panic(fmt.Sprintf("duplicate %q tag %q", TagKey, tag))
					}
					tags[tag] = index
				}
			}
		}
	}
	getTags(&v, nil)
	fs.VisitAll(func(fl *flag.Flag) {
		index := tags[fl.Name]
		if index == nil {
			return
		}
		getter, ok := fl.Value.(flag.Getter)
		if !ok {
			return
		}
		flv := reflect.ValueOf(getter.Get())
		fiv := v.FieldByIndex(index)
		if fiv.IsZero() || fl.Value.String() != fl.DefValue {
			fiv.Set(flv)
		}
	})
}
