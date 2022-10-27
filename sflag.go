package sflag

import (
	"flag"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// TagKey is the key used to retrieve informations about the flag in
// the struct field tag. The value associated with the tag key must be
// a comma separated list of three items:
// - the name of the flag
// - the default value for the flag
// - the help message for the flag
var TagKey = "flag"

func parseTag(v string) (name string, deflt string, help string) {
	parts := strings.SplitN(v, ",", 3)
	if len(parts) != 3 {
		panic(fmt.Sprintf("invalid tag value %q", v))
	}
	name, deflt, help = parts[0], parts[1], parts[2]
	return
}

// AddFlags adds flags to fs according to the tags of the struct
// contained in s.
func AddFlags(fs *flag.FlagSet, s any) {
	v := reflect.Indirect(reflect.ValueOf(s))
	if v.Kind() != reflect.Struct {
		panic("not a struct")
	}
	addFlags(fs, &v)
}

func addFlags(fs *flag.FlagSet, v *reflect.Value) {
	fields := reflect.VisibleFields(v.Type())
	for _, fi := range fields {
		kind := fi.Type.Kind()
		tag := fi.Tag.Get(TagKey)
		if tag == "" {
			if kind == reflect.Struct {
				fiv := v.FieldByIndex(fi.Index)
				addFlags(fs, &fiv)
			}
			continue
		}
		name, deflt, help := parseTag(tag)
		if fl := fs.Lookup(name); fl != nil {
			panic(fmt.Sprintf("flag %q already defined", name))
		}
		setDefault := true
		switch kind {
		case reflect.Bool:
			fs.Bool(name, false, help)
		case reflect.Int:
			fs.Int(name, 0, help)
		case reflect.Uint:
			fs.Uint(name, 0, help)
		case reflect.Int64:
			t := fi.Type
			var d time.Duration
			if t == reflect.TypeOf(d) {
				fs.Duration(name, d, help)
			} else {
				fs.Int64(name, 0, help)
			}
		case reflect.Uint64:
			fs.Uint64(name, 0, help)
		case reflect.Float64:
			fs.Float64(name, 0.0, help)
		case reflect.String:
			fs.String(name, "", help)
		default:
			i := reflect.TypeOf((*flag.Getter)(nil)).Elem()
			var v reflect.Value
			switch {
			case fi.Type.Implements(i):
				if fi.Type.Kind() == reflect.Pointer {
					v = reflect.New(fi.Type.Elem())
				} else {
					v = reflect.New(fi.Type).Elem()
				}
			case reflect.PointerTo(fi.Type).Implements(i):
				v = reflect.New(fi.Type)
			default:
				panic(fmt.Sprintf("invalid type %q for flag %q. It doesn't implements %q or it's not a type recognized by the flag package", fi.Type, name, i))
			}
			fs.Var(v.Interface().(flag.Value), name, help)
			setDefault = false
		}
		if setDefault {
			fl := fs.Lookup(name)
			if err := fl.Value.Set(deflt); err != nil {
				panic(fmt.Sprintf("invalid default value %q for flag %q", deflt, name))
			}
			fl.DefValue = fl.Value.String()
		}
	}
}

// SetFromFlags sets the value of the fields in the struct contained
// in s with the value of the flags defined in fs. It uses the tag of
// the struct fields to determine the fields whose value should be set
// and to determine the corresponding flag to use.
func SetFromFlags(s any, fs *flag.FlagSet) {
	if !fs.Parsed() {
		panic("flag not parsed")
	}
	v := reflect.Indirect(reflect.ValueOf(s))
	if v.Kind() != reflect.Struct {
		panic("not a struct")
	}
	indexes := make(map[string][]int)
	getFlagIndexes(indexes, &v, nil)
	fs.VisitAll(func(fl *flag.Flag) {
		index := indexes[fl.Name]
		if index == nil {
			return
		}
		getter, ok := fl.Value.(flag.Getter)
		if !ok {
			return
		}
		flv := reflect.ValueOf(getter.Get())
		fiv := v.FieldByIndex(index)
		if !fiv.IsZero() && fl.Value.String() == fl.DefValue {
			return
		}
		switch {
		case flv.Kind() == reflect.Pointer && flv.Type().Elem().AssignableTo(fiv.Type()):
			fiv.Set(flv.Elem())
		case fiv.Kind() == reflect.Pointer && fiv.Type().Elem().AssignableTo(flv.Type()):
			if fiv.IsNil() {
				fiv.Set(reflect.New(flv.Type()))
			}
			fiv.Elem().Set(flv)
		default:
			fiv.Set(flv)
		}
	})
}

func getFlagIndexes(indexes map[string][]int, v *reflect.Value, pindex []int) {
	fields := reflect.VisibleFields(v.Type())
	for _, fi := range fields {
		index := make([]int, len(pindex)+len(fi.Index))
		copy(index, pindex)
		copy(index[len(pindex):], fi.Index)
		if fi.Type.Kind() == reflect.Struct {
			fiv := v.FieldByIndex(fi.Index)
			getFlagIndexes(indexes, &fiv, index)
			continue
		}
		tag := fi.Tag.Get(TagKey)
		if tag == "" {
			continue
		}
		name, _, _ := parseTag(tag)
		if _, ok := indexes[name]; ok {
			panic(fmt.Sprintf("duplicate flag %q", name))
		}
		indexes[name] = index
	}
}
