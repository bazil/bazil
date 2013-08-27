package positional

import (
	"errors"
	"reflect"

	"bazil.org/bazil/cliutil/strconvx"
)

// ErrTooManyArgs indicates that there were too many arguments.
type ErrTooManyArgs struct{}

func (ErrTooManyArgs) Error() string {
	return "too many arguments"
}

// ErrMissingMandatoryArg indicates that a mandatory argument is
// missing.
type ErrMissingMandatoryArg struct {
	Name string
}

func (e ErrMissingMandatoryArg) Error() string {
	return "missing mandatory argument: " + e.Name
}

// Setter is an interface that can be implemented by fields that need
// to control their conversion from string to the data type.
type Setter interface {
	Set(string) error
}

// Optional is a marker for the point in the arguments struct where
// the rest of the fields are optional.
type Optional struct{}

// Parse fills the fields of the struct pointed to by args with input
// from the given list.
//
// Fields are handled based on their data type, for example strings
// are converted into integers when necessary.
//
// In addition to built-in conversion rules, if a field implements
// Setter, the Set method is called to do the conversion. This
// interface is compatible with flag.Value.
//
// To mark fields optional, insert an Optional marker in the struct
// after the last mandatory field.
//
// To consume any number of arguments, use a slice as the last field.
//
// Parse returns an error of type ErrMissingMandatoryArg if a mandatory
// field was not filled.
//
// Parse returns an error of type ErrTooManyArgs if there are more
// arguments than (non-slice) fields.
func Parse(args interface{}, list []string) error {
	pointer := reflect.ValueOf(args).Elem()
	if !pointer.CanSet() {
		return errors.New("must pass a pointer to positional.Parse")
	}
	value := reflect.Indirect(pointer)
	mandatory := true

	idx := 0

	for {

		if idx >= value.NumField() {
			break
		}

		if value.Type().Field(idx).Type == reflect.TypeOf(Optional{}) {
			mandatory = false
			idx++
			continue
		}

		if len(list) == 0 {
			if mandatory {
				name := meta(value.Type().Field(idx))
				return ErrMissingMandatoryArg{Name: name}
			}
			// only optional fields left and we ran out of args; ok
			break
		}

		field := value.Field(idx)
		switch field.Kind() {
		case reflect.Slice:
			if idx != value.NumField()-1 {
				return errors.New("cannot have items in argument struct after a slice element")
			}
			// consider mandatory requirement fulfilled
			mandatory = false

			v := reflect.Append(field, reflect.ValueOf(list[0]))
			field.Set(v)
			// do NOT advance idx; slice takes rest of args

		default:
			v, ok := field.Addr().Interface().(Setter)
			if ok {
				err := v.Set(list[0])
				if err != nil {
					return err
				}
			} else {
				if field.Kind() == reflect.Ptr {
					// instantiate a new value, parse into it
					val := reflect.New(field.Type().Elem())
					field.Set(val)
					field = val.Elem()
				}
				err := strconvx.Parse(field.Addr().Interface(), list[0])
				if err != nil {
					return err
				}
			}
			idx++
		}
		list = list[1:]
	}

	if len(list) > 0 {
		return ErrTooManyArgs{}
	}
	return nil
}
