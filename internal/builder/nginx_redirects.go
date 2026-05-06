package builder

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"unicode"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// BuildRedirectBlock turns the per-site redirects table into nginx location
// blocks ready to drop into a server{} context.
//
// Behaviour:
//   - Exact matches emit `location = /from { return <code> /to; }` plus a
//     mirror `location = /from/ { ... }` so trailing-slash variants resolve.
//   - Regex entries (from_path starts with `~`) emit `location ~ <pattern> { ... }`.
//   - Chains (A->B, B->C) collapse to A->C plus B->C, Google's hard one-hop rule.
//   - Loops (A->B, B->A) are dropped entirely so nginx never serves them.
//   - status_code 410 emits `return 410;` with no Location header (signals "gone"
//     to search engines so they drop the URL cleanly).
//   - Rows with hostile content (`{`, `}`, `;`, newlines, control chars) are
//     dropped with a slog warn. Defense in depth - Layer 2 validates on insert
//     but a compromised DB or pre-validation row must not produce a config that
//     escapes the location block.
//
// The output is deterministic (sorted by from_path) so two builds of the same
// site produce byte-identical nginx.conf.
func BuildRedirectBlock(redirects []store.Redirect) string {
	if len(redirects) == 0 {
		return ""
	}

	safe := make([]store.Redirect, 0, len(redirects))
	for _, r := range redirects {
		if !isSafeRedirect(r) {
			slog.Warn("dropping unsafe redirect from nginx emission",
				"site_id", r.SiteID, "id", r.ID, "from", r.FromPath, "to", r.ToPath)
			continue
		}
		safe = append(safe, r)
	}
	if len(safe) == 0 {
		return ""
	}

	collapsed := collapseRedirectChains(safe)
	if len(collapsed) == 0 {
		return ""
	}

	sort.Slice(collapsed, func(i, j int) bool {
		return collapsed[i].FromPath < collapsed[j].FromPath
	})

	var b strings.Builder
	b.WriteString("# Redirects (managed by atomicsite migration / slug-change auto-301)\n")

	for _, r := range collapsed {
		from := r.FromPath
		isRegex := strings.HasPrefix(from, "~")
		code := redirectCode(r.StatusCode)

		if isRegex {
			pattern := strings.TrimPrefix(from, "~")
			if r.StatusCode == 410 {
				b.WriteString(fmt.Sprintf("location ~ %s { return 410; }\n", pattern))
				continue
			}
			b.WriteString(fmt.Sprintf("location ~ %s { return %d %s; }\n",
				pattern, code, escapeNginxValue(r.ToPath)))
			continue
		}

		exact := normalizeExactPath(from)
		if r.StatusCode == 410 {
			b.WriteString(fmt.Sprintf("location = %s { return 410; }\n", exact))
			if !strings.HasSuffix(exact, "/") {
				b.WriteString(fmt.Sprintf("location = %s/ { return 410; }\n", exact))
			}
			continue
		}
		b.WriteString(fmt.Sprintf("location = %s { return %d %s; }\n",
			exact, code, escapeNginxValue(r.ToPath)))
		if !strings.HasSuffix(exact, "/") {
			b.WriteString(fmt.Sprintf("location = %s/ { return %d %s; }\n",
				exact, code, escapeNginxValue(r.ToPath)))
		}
	}
	b.WriteString("\n")
	return b.String()
}

// collapseRedirectChains walks the redirect graph for each entry, replacing
// the to_path with the terminal destination (max 10 hops). Cycles are dropped.
// Regex entries are passed through unchanged - chain analysis on patterns
// would require URL-rewriting logic we deliberately don't ship.
func collapseRedirectChains(in []store.Redirect) []store.Redirect {
	byFrom := make(map[string]store.Redirect, len(in))
	for _, r := range in {
		byFrom[r.FromPath] = r
	}

	out := make([]store.Redirect, 0, len(in))
	for _, r := range in {
		if strings.HasPrefix(r.FromPath, "~") || r.StatusCode == 410 {
			out = append(out, r)
			continue
		}

		seen := map[string]bool{r.FromPath: true}
		target := r.ToPath
		dropped := false
		for hop := 0; hop < 10; hop++ {
			next, ok := byFrom[target]
			if !ok {
				break
			}
			if strings.HasPrefix(next.FromPath, "~") || next.StatusCode == 410 {
				break
			}
			if seen[next.FromPath] {
				dropped = true
				break
			}
			seen[next.FromPath] = true
			target = next.ToPath
		}
		if dropped {
			continue
		}
		collapsed := r
		collapsed.ToPath = target
		out = append(out, collapsed)
	}
	return out
}

// redirectCode coerces a stored status_code into a value nginx accepts on a
// `return` directive. Anything outside the redirect range collapses to 301
// (the safe SEO default).
func redirectCode(code int64) int {
	switch code {
	case 301, 302, 307, 308:
		return int(code)
	default:
		return 301
	}
}

// normalizeExactPath strips a single trailing slash so we can emit the
// `/foo` and `/foo/` variants from one stored value without doubling slashes.
// Root path stays as `/`.
func normalizeExactPath(p string) string {
	if p == "/" {
		return p
	}
	return strings.TrimRight(p, "/")
}

// escapeNginxValue quotes a value if it contains characters nginx would
// otherwise misparse (semicolons, whitespace, quotes). Most paths come back
// clean, so we keep the bare form when safe.
func escapeNginxValue(v string) string {
	if v == "" {
		return `""`
	}
	if strings.ContainsAny(v, " \t\";\n") {
		return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	return v
}

// isSafeRedirect rejects rows that would let an attacker break out of the
// emitted `location { }` block or inject extra directives. The rules:
//
//   - from_path: must be non-empty; for exact match must start with `/`; for
//     regex must start with `~`. No `{`, `}`, `;`, control chars, quotes, or
//     backslashes anywhere.
//   - to_path: must start with `/` (off-site redirects need a separate flag
//     we don't ship in Layer 1). Same forbidden-character rules.
//   - status_code 410 still requires a structurally valid to_path because the
//     row could become a non-410 redirect later.
//
// This is the last line of defense - Layer 2 CRUD validates on insert, the
// sqlc UNIQUE constraint blocks dupes, but if either fails this stops broken
// or hostile rows from reaching nginx.
func isSafeRedirect(r store.Redirect) bool {
	from := r.FromPath
	if from == "" || !nginxTokenSafe(from) {
		return false
	}
	if strings.HasPrefix(from, "~") {
		// Regex form: must have a non-empty pattern after the ~ (and optional space).
		pat := strings.TrimSpace(strings.TrimPrefix(from, "~"))
		if pat == "" {
			return false
		}
	} else if !strings.HasPrefix(from, "/") {
		// Exact form must be a path.
		return false
	}
	to := r.ToPath
	if to == "" || !nginxTokenSafe(to) {
		return false
	}
	if !strings.HasPrefix(to, "/") {
		// to_path must be path-only in Layer 1; absolute URLs would need
		// extra parsing to ensure they cannot inject scheme:// trickery.
		return false
	}
	if strings.HasPrefix(to, "//") {
		// Protocol-relative URLs (`//evil.com/`) bypass the host check -
		// browsers resolve them against the current scheme. This is the
		// classic open-redirect vector. Drop unconditionally.
		return false
	}
	return true
}

// nginxTokenSafe returns false if the string contains any character that
// could close the surrounding `location { }` block or terminate a directive.
// Conservative on purpose - paths in real CMS exports are ASCII-printable
// minus the structural set; anything else is suspicious.
func nginxTokenSafe(s string) bool {
	for _, r := range s {
		if r == '{' || r == '}' || r == ';' || r == '"' || r == '\\' || r == '\'' {
			return false
		}
		if r < 0x20 || r == 0x7f {
			return false
		}
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
