package eval

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// RunAccessibilityChecks evaluates ARIA landmarks, alt text, form labels,
// heading order, focus indicators. Skips contrast (needs computed styles).
func RunAccessibilityChecks(site *SiteContext) []CheckResult {
	var checks []CheckResult

	checks = append(checks, perPageCheck("Main Landmark", "landmarks", 2, site, func(p PageContext) (bool, string) {
		if hasLandmark(p.Doc, "main") {
			return true, ""
		}
		return false, "no <main> or role=main"
	}, "Wrap page content in <main>."))

	checks = append(checks, perPageCheck("Navigation Landmark", "landmarks", 2, site, func(p PageContext) (bool, string) {
		if hasLandmark(p.Doc, "nav") {
			return true, ""
		}
		return false, "no <nav> or role=navigation"
	}, "Wrap navigation in <nav>."))

	checks = append(checks, perPageCheck("Banner Landmark", "landmarks", 1, site, func(p PageContext) (bool, string) {
		if hasLandmark(p.Doc, "header") {
			return true, ""
		}
		return false, "no <header> or role=banner"
	}, "Wrap site header in <header>."))

	checks = append(checks, perPageCheck("Contentinfo Landmark", "landmarks", 1, site, func(p PageContext) (bool, string) {
		if hasLandmark(p.Doc, "footer") {
			return true, ""
		}
		return false, "no <footer> or role=contentinfo"
	}, "Wrap site footer in <footer>."))

	checks = append(checks, perPageCheck("Page Language", "structure", 2, site, func(p PageContext) (bool, string) {
		htmlNode := firstElementByTag(p.Doc, "html")
		if htmlNode == nil || attr(htmlNode, "lang") == "" {
			return false, "no lang attribute on <html>"
		}
		return true, ""
	}, "Set lang= on the <html> element."))

	checks = append(checks, perPageCheck("Page Has H1", "structure", 2, site, func(p PageContext) (bool, string) {
		if len(elementsByTag(p.Doc, "h1")) == 0 {
			return false, "no <h1>"
		}
		return true, ""
	}, "Add exactly one <h1>."))

	checks = append(checks, perPageCheck("Heading Order", "structure", 2, site, func(p PageContext) (bool, string) {
		ok, msg := validateHeadingOrder(p.Doc)
		if !ok {
			return false, msg
		}
		return true, ""
	}, "Don't skip heading levels."))

	checks = append(checks, perPageCheck("No Empty Headings", "structure", 1, site, func(p PageContext) (bool, string) {
		for _, level := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
			for _, h := range elementsByTag(p.Doc, level) {
				if textContent(h) == "" {
					return false, fmt.Sprintf("empty <%s>", level)
				}
			}
		}
		return true, ""
	}, "All headings must contain visible text."))

	checks = append(checks, perPageCheck("Images Have Alt Text", "media", 3, site, func(p PageContext) (bool, string) {
		for _, img := range elementsByTag(p.Doc, "img") {
			if !hasAttr(img, "alt") {
				return false, "image missing alt attribute"
			}
		}
		return true, ""
	}, "Every <img> needs alt= (use alt=\"\" for decorative)."))

	checks = append(checks, perPageCheck("Form Inputs Have Labels", "forms", 3, site, func(p PageContext) (bool, string) {
		labels := elementsByTag(p.Doc, "label")
		labelFor := map[string]bool{}
		for _, l := range labels {
			if v := attr(l, "for"); v != "" {
				labelFor[v] = true
			}
		}
		for _, in := range elementsByTag(p.Doc, "input") {
			t := strings.ToLower(attr(in, "type"))
			if t == "hidden" || t == "submit" || t == "button" || t == "reset" {
				continue
			}
			id := attr(in, "id")
			if attr(in, "aria-label") != "" || attr(in, "aria-labelledby") != "" {
				continue
			}
			if id != "" && labelFor[id] {
				continue
			}
			return false, "input without associated <label> or aria-label"
		}
		return true, ""
	}, "Associate every input with a <label for=id> or aria-label."))

	checks = append(checks, perPageCheck("Buttons Have Labels", "forms", 2, site, func(p PageContext) (bool, string) {
		for _, b := range elementsByTag(p.Doc, "button") {
			if textContent(b) != "" || attr(b, "aria-label") != "" || attr(b, "title") != "" {
				continue
			}
			return false, "button without text or aria-label"
		}
		return true, ""
	}, "Buttons need accessible text (visible content or aria-label)."))

	checks = append(checks, perPageCheck("Links Have Labels", "forms", 2, site, func(p PageContext) (bool, string) {
		for _, a := range elementsByTag(p.Doc, "a") {
			if textContent(a) != "" || attr(a, "aria-label") != "" || attr(a, "title") != "" {
				continue
			}
			// Allow links that wrap an <img alt="...">
			imgs := elementsByTag(a, "img")
			hasAltImg := false
			for _, im := range imgs {
				if attr(im, "alt") != "" {
					hasAltImg = true
					break
				}
			}
			if hasAltImg {
				continue
			}
			return false, "link without accessible name"
		}
		return true, ""
	}, "Links need visible text, aria-label, or alt-bearing images."))

	checks = append(checks, perPageCheck("No Positive Tabindex", "focus", 2, site, func(p PageContext) (bool, string) {
		for _, n := range findAll(p.Doc, func(n *html.Node) bool {
			return n.Type == html.ElementNode && hasAttr(n, "tabindex")
		}) {
			tv := strings.TrimSpace(attr(n, "tabindex"))
			if tv != "" && tv != "0" && tv != "-1" {
				return false, "element with tabindex=\"" + tv + "\""
			}
		}
		return true, ""
	}, "Avoid tabindex > 0; let DOM order define tab sequence."))

	checks = append(checks, perPageCheck("Tables Have Headers", "structure", 2, site, func(p PageContext) (bool, string) {
		for _, t := range elementsByTag(p.Doc, "table") {
			if len(elementsByTag(t, "th")) == 0 {
				return false, "<table> with no <th>"
			}
		}
		return true, ""
	}, "Use <th> in tables for accessible header cells."))

	return checks
}

func hasLandmark(doc *html.Node, tag string) bool {
	// For <header>/<footer>, only count those NOT nested inside <article>,
	// <section>, <aside>, or <nav>. Per HTML5 spec, only top-level header/footer
	// are banner/contentinfo landmarks.
	scopedTags := map[string]bool{"header": true, "footer": true}
	if scopedTags[tag] {
		var found bool
		var walk func(*html.Node, bool)
		walk = func(n *html.Node, insideSectioning bool) {
			if found || n == nil {
				return
			}
			if n.Type == html.ElementNode {
				switch n.Data {
				case "article", "section", "aside", "nav":
					insideSectioning = true
				}
				if n.Data == tag && !insideSectioning {
					found = true
					return
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c, insideSectioning)
			}
		}
		walk(doc, false)
		if found {
			return true
		}
	} else {
		if firstElementByTag(doc, tag) != nil {
			return true
		}
	}
	roleMap := map[string]string{
		"main":   "main",
		"nav":    "navigation",
		"header": "banner",
		"footer": "contentinfo",
	}
	role := roleMap[tag]
	if role == "" {
		return false
	}
	return findFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && strings.EqualFold(attr(n, "role"), role)
	}) != nil
}
