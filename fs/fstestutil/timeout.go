package fstestutil

import (
	"flag"
	"testing"
	"time"
)

// SetDefaultTimeout sets the default value for the `go test
// -test.timeout` flag. Original default is no timeout.
func SetDefaultTimeout(d time.Duration) {
	testing.Init()
	f := flag.Lookup("test.timeout")
	if f == nil {
		panic("flag -test.timeout not found")
	}
	if f.Value.String() != "0" {
		// not at default value
		return
	}
	f.DefValue = d.String()
	err := f.Value.Set(f.DefValue)
	if err != nil {
		panic("ShortenTestTimeout cannot set Duration: " + err.Error())
	}
}
