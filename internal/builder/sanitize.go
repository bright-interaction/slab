package builder

import (
	"regexp"

	"github.com/microcosm-cc/bluemonday"
)

// componentNameRE accepts a safe component identifier. comp.Name is used as a
// filename under src/components/, so it must never contain a path separator,
// "..", or anything that could traverse out of that directory.
var componentNameRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

// ValidComponentName reports whether name is safe to use as a component filename.
func ValidComponentName(name string) bool {
	return componentNameRE.MatchString(name)
}

// rawHTMLPolicy sanitizes agent-authored raw-HTML block content. The raw-HTML
// block is an intentional author feature, but agent-controlled markup must not
// be able to inject executable content into the built page or the editor
// preview: bluemonday's UGC policy keeps ordinary formatting/markup while
// stripping <script>, <style>, event handlers (on*), and javascript:/data:
// URLs. Sites that genuinely need a script use the allowed-scripts surface, not
// a raw-HTML block. The policy is safe for concurrent Sanitize calls.
var rawHTMLPolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowStyling()              // permit class attributes for layout
	p.AllowAttrs("id").Globally() // keep anchor targets (href="#...") working
	return p
}()

// sanitizeRawHTML strips any executable content from agent-authored HTML before
// it is emitted into the generated Astro page or the editor preview. This is the
// render-time choke point that neutralizes stored XSS regardless of which write
// path (REST, MCP) or pre-existing row produced the content.
func sanitizeRawHTML(s string) string {
	return rawHTMLPolicy.Sanitize(s)
}
