package positional

import (
	"reflect"
	"strings"
)

// Get the named struct tag field.
// For example, from
//
//    Foo struct {
//        Bar string `positional:"something,metavar=BLEH,somethingelse"`
//    }
//
// calling
//
//    getTagField("something,metavar=BLEH,somethingelse", "metavar")
//
// would return
//
//    "BLEH"
//
// If the field is not found, empty string is returned. value may be
// an empty string. Field value cannot contain commas.
func getTagField(value string, field string) string {
	l := strings.FieldsFunc(value, func(r rune) bool { return r == ',' })
	prefix := field + "="
	for _, f := range l {
		if strings.HasPrefix(f, prefix) {
			return f[len(prefix):]
		}
	}
	return ""
}

func meta(field reflect.StructField) string {
	name := getTagField(field.Tag.Get("positional"), "metavar")
	if name == "" {
		name = strings.ToUpper(field.Name)
	}

	if field.Type.Kind() == reflect.Slice {
		name += ".."
	}
	return name
}

// Usage returns a string suitable for use in a command line synopsis.
//
// Struct tags with the key "positional" can be used to control the
// usage message. The following struct tags are supported:
//
//     - "metavar": meta variable name to use, defaults to field name
//       in upper case
func Usage(args interface{}) string {
	value := reflect.ValueOf(args)
	// let it be a pointer or not, we don't care here
	value = reflect.Indirect(value)

	metas := []string{}
	nest := 0

	i := 0

	for ; i < value.NumField(); i++ {
		if value.Type().Field(i).Type == reflect.TypeOf(Optional{}) {
			i++
			break
		}

		metas = append(metas, meta(value.Type().Field(i)))
	}

	for ; i < value.NumField(); i++ {
		metas = append(metas, "["+meta(value.Type().Field(i)))
		nest++
	}

	if nest > 0 {
		metas[len(metas)-1] += strings.Repeat("]", nest)
	}

	return strings.Join(metas, " ")
}
