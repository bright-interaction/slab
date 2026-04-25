package eval

import (
	"strings"
)

// RunSecurityChecks evaluates HTTP headers + HTML for security best practices.
// Headers are sourced from BuildSecurityHeaders (what the site WOULD emit when
// deployed via _headers or nginx.conf).
func RunSecurityChecks(site *SiteContext) []CheckResult {
	var checks []CheckResult
	h := site.Headers

	add := func(name, section string, present bool, weight int, recIfFail string) {
		if present {
			checks = append(checks, Pass(name, section, weight, "Header present"))
		} else {
			checks = append(checks, Fail(name, section, weight, SeverityWarning, "Header missing", recIfFail))
		}
	}

	// 1-9. Core security headers
	add("Strict-Transport-Security", "headers", h["Strict-Transport-Security"] != "", 2,
		"Enable HSTS in Settings -> Security to enforce HTTPS for 1 year.")
	add("Content-Security-Policy", "headers", h["Content-Security-Policy"] != "", 3,
		"Enable CSP. Atomicsite auto-builds it from your allowed scripts.")
	add("X-Content-Type-Options: nosniff", "headers",
		strings.EqualFold(h["X-Content-Type-Options"], "nosniff"), 1,
		"Set X-Content-Type-Options to 'nosniff'.")
	add("X-Frame-Options", "headers", h["X-Frame-Options"] != "", 2,
		"Set X-Frame-Options to DENY or SAMEORIGIN to prevent clickjacking.")
	add("Referrer-Policy", "headers", h["Referrer-Policy"] != "", 1,
		"Set Referrer-Policy to strict-origin-when-cross-origin.")
	add("Permissions-Policy", "headers", h["Permissions-Policy"] != "", 1,
		"Set Permissions-Policy to disable unused browser features.")
	add("Cross-Origin-Opener-Policy", "headers", h["Cross-Origin-Opener-Policy"] != "", 1,
		"Set COOP to same-origin to isolate browsing context.")
	add("Cross-Origin-Resource-Policy", "headers", h["Cross-Origin-Resource-Policy"] != "", 1,
		"Set CORP to same-origin to prevent cross-origin reads.")

	// 10. CSP quality
	if csp := h["Content-Security-Policy"]; csp != "" {
		quality, issues := analyzeCSP(csp)
		switch quality {
		case "strong":
			checks = append(checks, Pass("CSP Quality", "csp", 3, "Strong CSP, no issues"))
		case "moderate":
			checks = append(checks, Fail("CSP Quality", "csp", 2, SeverityWarning,
				"Moderate CSP: "+strings.Join(issues, "; "),
				"Tighten CSP by removing 'unsafe-inline'/'unsafe-eval' and adding missing directives."))
		default:
			checks = append(checks, Fail("CSP Quality", "csp", 3, SeverityError,
				"Weak CSP: "+strings.Join(issues, "; "),
				"Rebuild CSP from scratch -- multiple critical issues."))
		}
	}

	// 11. SRI for cross-origin scripts/links across ALL pages
	if len(site.Pages) > 0 {
		var thirdPartyTotal, thirdPartyWithSRI int
		for _, p := range site.Pages {
			t, s := analyzeSRI(p)
			thirdPartyTotal += t
			thirdPartyWithSRI += s
		}
		if thirdPartyTotal == 0 {
			checks = append(checks, Pass("Subresource Integrity", "sri", 2, "No third-party scripts to hash"))
		} else {
			ratio := float64(thirdPartyWithSRI) / float64(thirdPartyTotal)
			if ratio >= 1 {
				checks = append(checks, Pass("Subresource Integrity", "sri", 2,
					"All third-party resources have integrity hashes"))
			} else if ratio >= 0.5 {
				checks = append(checks, Fail("Subresource Integrity", "sri", 1, SeverityWarning,
					"Some third-party resources missing SRI",
					"Add integrity= and crossorigin=anonymous to all third-party <script>/<link> tags."))
			} else {
				checks = append(checks, Fail("Subresource Integrity", "sri", 2, SeverityError,
					"Most third-party resources lack SRI",
					"Add integrity hashes to prevent supply-chain compromise."))
			}
		}
	}

	// 12. No mixed content
	mixedFound := false
	for _, p := range site.Pages {
		if hasMixedContent(p.HTML) {
			mixedFound = true
			break
		}
	}
	if mixedFound {
		checks = append(checks, Fail("No Mixed Content", "mixed", 2, SeverityError,
			"http:// URLs found in HTML", "Replace all http:// links/sources with https:// or relative URLs."))
	} else {
		checks = append(checks, Pass("No Mixed Content", "mixed", 2, "All resources HTTPS or relative"))
	}

	// 13. security.txt
	if site.SecurityTxt != "" {
		checks = append(checks, Pass("security.txt", "well-known", 1, "RFC 9116 contact published"))
	} else {
		checks = append(checks, Fail("security.txt", "well-known", 1, SeverityWarning,
			"No /.well-known/security.txt", "Set security_email in your site profile."))
	}

	// 14. CSP on home page (verify the meta-stripped form is present in HTML)
	if len(site.Pages) > 0 {
		hasMetaCSP := false
		for _, p := range site.Pages {
			if findMetaContent(p.Doc, "Content-Security-Policy") != "" {
				hasMetaCSP = true
				break
			}
		}
		if hasMetaCSP {
			checks = append(checks, Pass("CSP meta tag", "csp", 1, "Meta CSP emitted on at least one page"))
		} else if h["Content-Security-Policy"] != "" {
			checks = append(checks, Fail("CSP meta tag", "csp", 1, SeverityInfo,
				"CSP set in headers but no meta fallback",
				"Static hosts without header support won't apply CSP. Layouts.go should emit a meta tag."))
		}
	}

	return checks
}

// analyzeCSP returns "strong", "moderate", or "weak" plus list of issues.
// Parses directives so unsafe-eval/unsafe-inline are only flagged when in
// security-sensitive directives (script-src/default-src), not style-src.
func analyzeCSP(csp string) (string, []string) {
	var issues []string
	directives := map[string]string{}
	for _, dir := range splitCSP(csp) {
		parts := strings.SplitN(strings.TrimSpace(dir), " ", 2)
		if len(parts) < 1 {
			continue
		}
		name := strings.ToLower(parts[0])
		val := ""
		if len(parts) == 2 {
			val = strings.ToLower(parts[1])
		}
		directives[name] = val
	}

	scriptSrc := directives["script-src"]
	defaultSrc := directives["default-src"]
	scriptish := scriptSrc
	if scriptish == "" {
		scriptish = defaultSrc
	}

	if strings.Contains(scriptish, "'unsafe-eval'") {
		issues = append(issues, "script-src has unsafe-eval")
	}
	if strings.Contains(scriptish, "'unsafe-inline'") && !strings.Contains(scriptish, "nonce-") {
		issues = append(issues, "script-src has unsafe-inline (no nonce)")
	}
	// Wildcard host in any sensitive directive
	for _, name := range []string{"script-src", "default-src", "connect-src"} {
		v := directives[name]
		if v == "*" || strings.Contains(v, " * ") || strings.HasSuffix(v, " *") || strings.HasPrefix(v, "* ") {
			issues = append(issues, "wildcard host in "+name)
		}
	}

	required := []string{"default-src", "script-src", "frame-ancestors", "base-uri", "object-src", "form-action"}
	for _, r := range required {
		if _, ok := directives[r]; !ok {
			issues = append(issues, "missing "+r)
		}
	}

	switch {
	case len(issues) == 0:
		return "strong", nil
	case len(issues) <= 2:
		return "moderate", issues
	default:
		return "weak", issues
	}
}

func splitCSP(csp string) []string {
	var out []string
	for _, p := range strings.Split(csp, ";") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// analyzeSRI counts third-party (cross-origin) script/link tags and how many have integrity attrs.
func analyzeSRI(page PageContext) (total, withSRI int) {
	for _, s := range elementsByTag(page.Doc, "script") {
		src := attr(s, "src")
		if isThirdPartyURL(src) {
			total++
			if attr(s, "integrity") != "" {
				withSRI++
			}
		}
	}
	for _, l := range elementsByTag(page.Doc, "link") {
		rel := strings.ToLower(attr(l, "rel"))
		if rel != "stylesheet" {
			continue
		}
		href := attr(l, "href")
		if isThirdPartyURL(href) {
			total++
			if attr(l, "integrity") != "" {
				withSRI++
			}
		}
	}
	return total, withSRI
}

func isThirdPartyURL(u string) bool {
	if u == "" {
		return false
	}
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "//")
}

func hasMixedContent(htmlStr string) bool {
	// Heuristic: any "http://" URL inside src= or href= that isn't localhost
	lower := strings.ToLower(htmlStr)
	for _, prefix := range []string{`src="http://`, `src='http://`, `href="http://`, `href='http://`} {
		idx := strings.Index(lower, prefix)
		for idx >= 0 {
			// ignore http://localhost, http://127.0.0.1
			rest := lower[idx+len(prefix):]
			if !strings.HasPrefix(rest, "localhost") && !strings.HasPrefix(rest, "127.") {
				return true
			}
			next := strings.Index(lower[idx+1:], prefix)
			if next < 0 {
				break
			}
			idx = idx + 1 + next
		}
	}
	return false
}
