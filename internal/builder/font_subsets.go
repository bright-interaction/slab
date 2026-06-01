package builder

// Unicode-range subsets keyed by the value stored in site_fonts.subset.
// Ranges are the canonical Google Fonts subsets so an operator who pre-
// subset their woff2 elsewhere can match exactly. "all" maps to an empty
// string so the renderer omits the unicode-range descriptor entirely;
// "latin" is the default for new uploads (EU/UK target market).
var unicodeRanges = map[string]string{
	"latin":      "U+0000-00FF, U+0131, U+0152-0153, U+02BB-02BC, U+02C6, U+02DA, U+02DC, U+0304, U+0308, U+0329, U+2000-206F, U+2074, U+20AC, U+2122, U+2191, U+2193, U+2212, U+2215, U+FEFF, U+FFFD",
	"latin-ext":  "U+0100-02AF, U+0304, U+0308, U+0329, U+1E00-1E9F, U+1EF2-1EFF, U+2020, U+20A0-20AB, U+20AD-20C0, U+2113, U+2C60-2C7F, U+A720-A7FF",
	"cyrillic":   "U+0400-045F, U+0490-0491, U+04B0-04B1, U+2116",
	"greek":      "U+0370-03FF",
	"vietnamese": "U+0102-0103, U+0110-0111, U+0128-0129, U+0168-0169, U+01A0-01A1, U+01AF-01B0, U+1EA0-1EF9, U+20AB",
	"arabic":     "U+0600-06FF, U+200C-200E, U+2010-2011, U+204F, U+2E41, U+FB50-FDFF, U+FE80-FEFC",
	"hebrew":     "U+0590-05FF, U+200C-2010, U+20AA, U+25CC, U+FB1D-FB4F",
	"all":        "",
}

// UnicodeRangeFor returns the unicode-range descriptor string for a given
// subset name. Empty / unknown / "all" return "" so the renderer omits the
// descriptor entirely and the browser loads the font on every page.
func UnicodeRangeFor(subset string) string {
	if r, ok := unicodeRanges[subset]; ok {
		return r
	}
	return ""
}

// fontSubsetOrAll is the display-friendly companion to UnicodeRangeFor:
// empty subset names render as "all" in the source comment so an operator
// reading the HTML sees a meaningful label rather than nothing.
func fontSubsetOrAll(subset string) string {
	if subset == "" {
		return "all"
	}
	return subset
}
