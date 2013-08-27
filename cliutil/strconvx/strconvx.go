// Package strconvx implements conversions from strings to basic data
// types.
//
// It is similar to the strconv package, but with a calling convention
// that gives less control over details, but works better when the
// type of the variable is not necessarily known, such as in
// libraries.
package strconvx

import (
	"fmt"
	"strconv"
)

// Parse interprets a string as a value of any of the recognized
// types. Value should be a pointer.
func Parse(value interface{}, s string) error {
	switch x := value.(type) {
	case *string:
		*x = s
		return nil

	case *int:
		res, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		// check for overflow in 32-bit environments
		if int64(int(res)) != res {
			return &strconv.NumError{
				Func: "ParseInt",
				Num:  s,
				Err:  strconv.ErrRange,
			}
		}
		*x = int(res)
		return nil

	case *int8:
		res, err := strconv.ParseInt(s, 10, 8)
		if err != nil {
			return err
		}
		*x = int8(res)
		return nil

	case *int16:
		res, err := strconv.ParseInt(s, 10, 16)
		if err != nil {
			return err
		}
		*x = int16(res)
		return nil

	case *int32:
		res, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return err
		}
		*x = int32(res)
		return nil

	case *int64:
		res, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*x = int64(res)
		return nil

	case *uint:
		res, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		// check for overflow in 32-bit environments
		if uint64(uint(res)) != res {
			return &strconv.NumError{
				Func: "ParseInt",
				Num:  s,
				Err:  strconv.ErrRange,
			}
		}
		*x = uint(res)
		return nil

	case *uint8:
		res, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			return err
		}
		*x = uint8(res)
		return nil

	case *uint16:
		res, err := strconv.ParseUint(s, 10, 16)
		if err != nil {
			return err
		}
		*x = uint16(res)
		return nil

	case *uint32:
		res, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return err
		}
		*x = uint32(res)
		return nil

	case *uint64:
		res, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		*x = uint64(res)
		return nil

	// TODO maybe float32, float64, float
	default:
		return fmt.Errorf("cannot parse into %T", value)
	}
}
