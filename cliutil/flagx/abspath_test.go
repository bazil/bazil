package flagx_test

import (
	"path/filepath"
	"testing"

	"bazil.org/bazil/cliutil/flagx"
)

func TestAbsPathEmpty(t *testing.T) {
	var s flagx.AbsPath
	err := s.Set("")
	if err != flagx.ErrEmptyPath {
		t.Fatalf("expected ErrEmptyPath, got %v", err)
	}
}

func setAbsPath(t testing.TB, value string) string {
	var s flagx.AbsPath
	err := s.Set(value)
	if err != nil {
		t.Fatalf("AbsPath.Set failed: %v", err)
	}
	return s.String()
}

func TestAbsPathAbsolute(t *testing.T) {
	if g, e := setAbsPath(t, "/fake-path-name"), "/fake-path-name"; g != e {
		t.Errorf("unexpected AbsPath: %q != %q", g, e)
	}
}

func TestAbsPathRelative(t *testing.T) {
	want, err := filepath.Abs("fake-path-name")
	if err != nil {
		t.Fatal(err)
	}
	if g, e := setAbsPath(t, "fake-path-name"), want; g != e {
		t.Errorf("unexpected AbsPath: %q != %q", g, e)
	}
}
