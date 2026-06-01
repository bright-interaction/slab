package builder

import (
	"strings"
	"testing"
)

func TestUnicodeRangeFor_AllSubsetsReturnNonEmpty(t *testing.T) {
	subsets := []string{"latin", "latin-ext", "cyrillic", "greek", "vietnamese", "arabic", "hebrew"}
	for _, s := range subsets {
		got := UnicodeRangeFor(s)
		if got == "" {
			t.Errorf("subset %q must return a non-empty unicode-range; got empty", s)
		}
		if !strings.Contains(got, "U+") {
			t.Errorf("subset %q range should contain at least one Unicode codepoint marker; got %q", s, got)
		}
	}
}

func TestUnicodeRangeFor_AllAndUnknownReturnEmpty(t *testing.T) {
	if got := UnicodeRangeFor("all"); got != "" {
		t.Errorf(`"all" should return empty so @font-face emits no unicode-range; got %q`, got)
	}
	if got := UnicodeRangeFor(""); got != "" {
		t.Errorf("empty subset should return empty; got %q", got)
	}
	if got := UnicodeRangeFor("not-a-real-subset"); got != "" {
		t.Errorf("unknown subset should return empty (lets browser load on every page); got %q", got)
	}
}

func TestUnicodeRangeFor_LatinCoversBasicASCII(t *testing.T) {
	got := UnicodeRangeFor("latin")
	if !strings.Contains(got, "U+0000-00FF") {
		t.Errorf("Latin subset should cover the Basic Latin block U+0000-00FF; got %q", got)
	}
	if !strings.Contains(got, "U+20AC") {
		t.Errorf("Latin subset should cover the Euro sign U+20AC; got %q", got)
	}
}

func TestFontSubsetOrAll(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "all"},
		{"latin", "latin"},
		{"cyrillic", "cyrillic"},
		{"all", "all"},
	}
	for _, c := range cases {
		if got := fontSubsetOrAll(c.in); got != c.want {
			t.Errorf("fontSubsetOrAll(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
