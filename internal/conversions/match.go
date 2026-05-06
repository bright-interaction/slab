// Package conversions: goal-match engine for Phase 31.1.
//
// A goal definition (match_type + match_value) is evaluated against an
// in-flight signal (page path, explicit event name, or form submit
// id) by the matcher. The matcher is pure Go with zero DB or HTTP
// deps so it can be unit-tested table-driven and re-used from the
// pageview, /t/event, and form-submission ingest paths without
// import cycles.
package conversions

import "strings"

// MatchType is the goal matching strategy. Stored as TEXT in
// conversion_goals.match_type so the values are stable strings.
type MatchType string

const (
	MatchTypeURLPattern MatchType = "url_pattern"
	MatchTypeEventName  MatchType = "event_name"
	MatchTypeFormSubmit MatchType = "form_submit"
)

// IsValidMatchType reports whether s is one of the supported types.
// Used by handler validation before persisting a goal definition.
func IsValidMatchType(s string) bool {
	switch MatchType(s) {
	case MatchTypeURLPattern, MatchTypeEventName, MatchTypeFormSubmit:
		return true
	}
	return false
}

// Goal is the matcher's view of a conversion_goal row. Only the three
// fields the matcher needs are surfaced so the calling code stays
// decoupled from the sqlc-generated struct.
type Goal struct {
	ID         string
	SiteID     string
	Slug       string
	MatchType  MatchType
	MatchValue string
}

// MatchURLPattern reports whether path matches pattern. Pattern
// supports `*` as a single-segment wildcard. A trailing `*` (or a
// pattern of just `*`) acts as "match the rest, including across
// `/`". A non-trailing `*` matches zero or more characters that do
// NOT cross a `/` boundary. Matching is anchored at both ends and
// case-sensitive.
//
// Examples:
//
//	pattern="/thank-you",      path="/thank-you"            -> true
//	pattern="/thank-you/*",    path="/thank-you/lead"       -> true
//	pattern="/thank-you/*",    path="/thank-you"            -> false
//	pattern="/blog/*/comments",path="/blog/x/comments"      -> true
//	pattern="/blog/*/comments",path="/blog/x/y/comments"    -> false
//	pattern="*",               path="/anything"             -> true
//	pattern="/foo*",           path="/foobar"               -> true
func MatchURLPattern(pattern, path string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	return globMatch(pattern, path)
}

// globMatch is the recursive matcher. The wildcard rules are:
//
//   - `*` at the end of the pattern matches the rest of the path
//     including '/' (trailing wildcard).
//   - `*` anywhere else matches zero or more characters that do NOT
//     include '/' (single-segment wildcard).
func globMatch(p, s string) bool {
	for {
		if len(p) == 0 {
			return len(s) == 0
		}
		if p[0] == '*' {
			rest := p[1:]
			if rest == "" {
				return true
			}
			// Try every split point in s, but stop at the first
			// '/' (the wildcard cannot cross a segment boundary
			// when followed by more pattern).
			for i := 0; i <= len(s); i++ {
				if globMatch(rest, s[i:]) {
					return true
				}
				if i < len(s) && s[i] == '/' {
					return false
				}
			}
			return false
		}
		if len(s) == 0 || p[0] != s[0] {
			return false
		}
		p, s = p[1:], s[1:]
	}
}

// MatchGoals filters goals down to those that match the given
// (match_type, value) tuple. value is the path for url_pattern, the
// event name for event_name, or the form_id for form_submit. The
// result preserves goals' input order.
func MatchGoals(goals []Goal, matchType MatchType, value string) []Goal {
	out := make([]Goal, 0, len(goals))
	for _, g := range goals {
		if g.MatchType != matchType {
			continue
		}
		switch matchType {
		case MatchTypeURLPattern:
			if MatchURLPattern(g.MatchValue, value) {
				out = append(out, g)
			}
		case MatchTypeEventName, MatchTypeFormSubmit:
			if g.MatchValue == value {
				out = append(out, g)
			}
		}
	}
	return out
}
