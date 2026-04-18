package storage

import "testing"

func TestSafeLogicalPath(t *testing.T) {
	cases := []struct {
		in    string
		ok    bool
		out   string
	}{
		{"docs/readme.txt", true, "docs/readme.txt"},
		{"/docs/readme.txt", true, "docs/readme.txt"},
		{"../etc/passwd", false, ""},
		{"foo..bar/baz", true, "foo..bar/baz"},
		{"", false, ""},
		{"..", false, ""},
	}
	for _, c := range cases {
		got, err := SafeLogicalPath(c.in)
		if c.ok && err != nil {
			t.Fatalf("%q: unexpected err: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("%q: expected error", c.in)
		}
		if c.ok && got != c.out {
			t.Fatalf("%q: got %q want %q", c.in, got, c.out)
		}
	}
}
