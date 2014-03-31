package flagx_test

import (
	"path/filepath"
	"testing"

	"bazil.org/bazil/cliutil/flagx"
)

func TestEmpty(t *testing.T) {
	var s flagx.AbsPath
	err := s.Set("")
	if err != flagx.EmptyPathError {
		t.Fatalf("expected EmptyPathError, got %v", err)
	}
}

func set(t testing.TB, value string) string {
	var s flagx.AbsPath
	err := s.Set(value)
	if err != nil {
		t.Fatalf("AbsPath.Set failed: %v", err)
	}
	return s.String()
}

func TestAbsolute(t *testing.T) {
	if g, e := set(t, "/fake-path-name"), "/fake-path-name"; g != e {
		t.Errorf("unexpected AbsPath: %q != %q", g, e)
	}
}

func TestRelative(t *testing.T) {
	want, err := filepath.Abs("fake-path-name")
	if err != nil {
		t.Fatal(err)
	}
	if g, e := set(t, "fake-path-name"), want; g != e {
		t.Errorf("unexpected AbsPath: %q != %q", g, e)
	}
}
