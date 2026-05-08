package shield

import (
	"context"
	"regexp"
	"strings"
)

// redactPattern is one regex + the kind it produces. Order in
// redactPatterns matters: more specific patterns must come before more
// general ones (e.g. personnummer before generic digit runs).
type redactPattern struct {
	kind Kind
	re   *regexp.Regexp
	hint []string // hint keys to extract on match
}

var redactPatterns = []redactPattern{
	{
		kind: KindPersonnummer,
		// Swedish personnummer: YYYYMMDD-XXXX or YYMMDD-XXXX, optional
		// dash. 10 or 12 digits with optional dash separator.
		re:   regexp.MustCompile(`\b(?:19|20)?\d{6}[-\s]?\d{4}\b`),
		hint: []string{"century"},
	},
	{
		kind: KindIBAN,
		// IBAN: 2 letters + 2 digits + 11..30 alnum. Loosely matched;
		// detail validation happens at the application layer.
		re:   regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{10,30}\b`),
		hint: []string{"country"},
	},
	{
		kind: KindEmail,
		re:   regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`),
		hint: []string{"domain", "len"},
	},
	{
		kind: KindPhone,
		// Loose international + Swedish formats. Matches +CC followed
		// by 7..14 digits with optional spaces/dashes/parentheses.
		re:   regexp.MustCompile(`\+\d{1,3}[\s\-().]*\d{1,4}[\s\-().]*\d{1,4}[\s\-().]*\d{1,9}`),
		hint: []string{"country", "len"},
	},
}

// RedactString runs the regex bank over in and replaces every match
// with a marker. Used for free-form text fields (note bodies, message
// content, comments) where struct-tag-based shielding does not apply
// because the PII is embedded inside narrative text.
func (s *Session) RedactString(ctx context.Context, in string) (string, error) {
	if in == "" {
		return in, nil
	}
	out := in
	for _, p := range redactPatterns {
		var firstErr error
		out = p.re.ReplaceAllStringFunc(out, func(match string) string {
			// Skip if the match is already inside an existing marker.
			if strings.HasPrefix(match, "[shield:") {
				return match
			}
			hint := buildHint(p.kind, match, p.hint)
			marker, err := s.Tokenize(ctx, p.kind, match, hint)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return match
			}
			return marker
		})
		if firstErr != nil {
			return out, firstErr
		}
	}
	return out, nil
}
