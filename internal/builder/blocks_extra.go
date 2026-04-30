package builder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brightinteraction/atomicsite/internal/store"
)

// renderSplitHeroBlock renders a side-by-side hero (image right + text left
// on desktop, stacked on mobile). Use when the page wants a SaaS-marketing
// shape rather than the centered hero. Layout flips to centered when the
// site setting general.hero_layout=centered or the block's data.layout=centered.
//
// Optional bg modes via data.bg:
//   - "circuit": ships an animated PCB-style canvas behind the content
//     (auto-tints to --color-primary, prefers-reduced-motion respected).
//   - "circuit-static": same SVG pattern but static (no canvas / no JS).
func renderSplitHeroBlock(data map[string]any, mediaByID map[string]store.Medium) string {
	var b strings.Builder
	bg := dataString(data, "bg")
	cls := "block block--split_hero"
	if bg == "circuit" || bg == "circuit-static" {
		cls += " has-circuit-bg"
	}
	b.WriteString(fmt.Sprintf("  <section class=\"%s\"", cls))
	if bg == "circuit-static" {
		b.WriteString(` data-bg="circuit"`)
	}
	b.WriteString(">\n")
	if bg == "circuit" {
		b.WriteString("    <canvas data-circuit-canvas class=\"block-circuit-canvas\" aria-hidden=\"true\"></canvas>\n")
		b.WriteString("    <script>")
		b.Write(CircuitBgScript)
		b.WriteString("</script>\n")
	}
	b.WriteString("    <div class=\"split-hero-content\">\n")
	if eyebrow := dataString(data, "eyebrow"); eyebrow != "" {
		b.WriteString(fmt.Sprintf("      <p class=\"eyebrow\">%s</p>\n", escapeHTML(eyebrow)))
	}
	if headline := dataString(data, "headline"); headline != "" {
		b.WriteString(fmt.Sprintf("      <h1>%s</h1>\n", escapeHTML(headline)))
	}
	if sub := dataString(data, "subheading"); sub != "" {
		b.WriteString(fmt.Sprintf("      <p class=\"subheading\">%s</p>\n", escapeHTML(sub)))
	}
	if ctaText := dataString(data, "cta_text"); ctaText != "" {
		ctaURL := dataString(data, "cta_url")
		if ctaURL == "" {
			ctaURL = "#"
		}
		b.WriteString(fmt.Sprintf("      <a href=\"%s\" class=\"btn-primary\">%s</a>\n",
			escapeURL(ctaURL), escapeHTML(ctaText)))
	}
	if secLabel := dataString(data, "secondary_label"); secLabel != "" {
		secURL := dataString(data, "secondary_url")
		if secURL == "" {
			secURL = "#"
		}
		b.WriteString(fmt.Sprintf("      <a href=\"%s\" class=\"btn-secondary\">%s</a>\n",
			escapeURL(secURL), escapeHTML(secLabel)))
	}
	b.WriteString("    </div>\n")
	b.WriteString("    <div class=\"split-hero-image\">\n")
	if imageID := dataString(data, "image_id"); imageID != "" {
		b.WriteString("      " + renderMediaImg(imageID, dataString(data, "image_alt"), dataString(data, "headline"), "split-hero-img", mediaByID) + "\n")
	}
	b.WriteString("    </div>\n")
	b.WriteString("  </section>\n")
	return b.String()
}

// renderStatGridBlock renders a horizontal grid of large numbers + labels.
// Used for trust signals, social proof, hero stats. Each item: {value, label,
// description?}. Distinct from feature_grid (no icons, number-first hierarchy).
func renderStatGridBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--stat_grid\">\n")
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if sub := dataString(data, "subheading"); sub != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"subheading\">%s</p>\n", escapeHTML(sub)))
	}
	if itemsRaw, ok := data["items"].([]any); ok && len(itemsRaw) > 0 {
		b.WriteString("    <ul class=\"stat-grid\">\n")
		for _, it := range itemsRaw {
			item, ok := it.(map[string]any)
			if !ok {
				continue
			}
			b.WriteString("      <li class=\"stat-grid-item\">\n")
			if v := dataString(item, "value"); v != "" {
				b.WriteString(fmt.Sprintf("        <div class=\"stat-value\">%s</div>\n", escapeHTML(v)))
			}
			if l := dataString(item, "label"); l != "" {
				b.WriteString(fmt.Sprintf("        <div class=\"stat-label\">%s</div>\n", escapeHTML(l)))
			}
			b.WriteString("      </li>\n")
		}
		b.WriteString("    </ul>\n")
	}
	b.WriteString("  </section>\n")
	return b.String()
}

// renderAccordionFAQBlock renders a Q&A accordion via native <details>/<summary>
// (no JS needed). Each item: {question, answer}.
func renderAccordionFAQBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--accordion_faq\">\n")
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if sub := dataString(data, "subheading"); sub != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"subheading\">%s</p>\n", escapeHTML(sub)))
	}
	if itemsRaw, ok := data["items"].([]any); ok && len(itemsRaw) > 0 {
		// Emit FAQPage JSON-LD so the eval picks up structured FAQ.
		var faqLDItems []map[string]any
		for _, it := range itemsRaw {
			item, ok := it.(map[string]any)
			if !ok {
				continue
			}
			q := dataString(item, "question")
			a := dataString(item, "answer")
			if q == "" {
				continue
			}
			faqLDItems = append(faqLDItems, map[string]any{
				"@type": "Question",
				"name":  q,
				"acceptedAnswer": map[string]any{
					"@type": "Answer",
					"text":  a,
				},
			})
		}
		if len(faqLDItems) > 0 {
			ld := map[string]any{
				"@context":   "https://schema.org",
				"@type":      "FAQPage",
				"mainEntity": faqLDItems,
			}
			ldBytes, _ := json.Marshal(ld)
			b.WriteString("    <script type=\"application/ld+json\">")
			b.Write(ldBytes)
			b.WriteString("</script>\n")
		}
		b.WriteString("    <div class=\"faq-accordion\">\n")
		for _, it := range itemsRaw {
			item, ok := it.(map[string]any)
			if !ok {
				continue
			}
			q := dataString(item, "question")
			a := dataString(item, "answer")
			if q == "" {
				continue
			}
			b.WriteString("      <details class=\"faq-item\">\n")
			b.WriteString(fmt.Sprintf("        <summary>%s</summary>\n", escapeHTML(q)))
			if a != "" {
				b.WriteString("        <div class=\"faq-body\">\n")
				renderTextParagraphs(&b, a)
				b.WriteString("        </div>\n")
			}
			b.WriteString("      </details>\n")
		}
		b.WriteString("    </div>\n")
	}
	b.WriteString("  </section>\n")
	return b.String()
}

// renderPricingBlock renders a 3-up tier card grid. Each tier:
// {name, price, price_period, description, features[], cta_text, cta_url, featured?}.
func renderPricingBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--pricing\">\n")
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if sub := dataString(data, "subheading"); sub != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"subheading\">%s</p>\n", escapeHTML(sub)))
	}
	if tiersRaw, ok := data["tiers"].([]any); ok && len(tiersRaw) > 0 {
		b.WriteString("    <ul class=\"pricing-grid\">\n")
		for _, t := range tiersRaw {
			tier, ok := t.(map[string]any)
			if !ok {
				continue
			}
			cls := "pricing-tier"
			if featured, _ := tier["featured"].(bool); featured {
				cls += " is-featured"
			}
			b.WriteString(fmt.Sprintf("      <li class=\"%s\">\n", cls))
			if n := dataString(tier, "name"); n != "" {
				b.WriteString(fmt.Sprintf("        <div class=\"tier-name\">%s</div>\n", escapeHTML(n)))
			}
			if p := dataString(tier, "price"); p != "" {
				b.WriteString(fmt.Sprintf("        <div class=\"tier-price\">%s</div>\n", escapeHTML(p)))
			}
			if pp := dataString(tier, "price_period"); pp != "" {
				b.WriteString(fmt.Sprintf("        <div class=\"tier-price-period\">%s</div>\n", escapeHTML(pp)))
			}
			if d := dataString(tier, "description"); d != "" {
				b.WriteString(fmt.Sprintf("        <p class=\"tier-description\">%s</p>\n", escapeHTML(d)))
			}
			if featuresRaw, ok := tier["features"].([]any); ok && len(featuresRaw) > 0 {
				b.WriteString("        <ul class=\"tier-features\">\n")
				for _, f := range featuresRaw {
					if fs, ok := f.(string); ok && fs != "" {
						b.WriteString(fmt.Sprintf("          <li>%s</li>\n", escapeHTML(fs)))
					}
				}
				b.WriteString("        </ul>\n")
			}
			if ctaText := dataString(tier, "cta_text"); ctaText != "" {
				ctaURL := dataString(tier, "cta_url")
				if ctaURL == "" {
					ctaURL = "#"
				}
				cls := "btn-primary"
				if featured, _ := tier["featured"].(bool); !featured {
					cls = "btn-secondary"
				}
				b.WriteString(fmt.Sprintf("        <div class=\"tier-cta\"><a href=\"%s\" class=\"%s\">%s</a></div>\n",
					escapeURL(ctaURL), cls, escapeHTML(ctaText)))
			}
			b.WriteString("      </li>\n")
		}
		b.WriteString("    </ul>\n")
	}
	b.WriteString("  </section>\n")
	return b.String()
}

// renderLogoStripBlock renders a row of customer/partner logos. Each item:
// {image_id, alt, href?}.
func renderLogoStripBlock(data map[string]any, mediaByID map[string]store.Medium) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--logo_strip\">\n")
	if label := dataString(data, "label"); label != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"logo-strip-label\">%s</p>\n", escapeHTML(label)))
	}
	if itemsRaw, ok := data["items"].([]any); ok && len(itemsRaw) > 0 {
		b.WriteString("    <ul class=\"logo-strip\">\n")
		for _, it := range itemsRaw {
			item, ok := it.(map[string]any)
			if !ok {
				continue
			}
			imageID := dataString(item, "image_id")
			if imageID == "" {
				continue
			}
			alt := dataString(item, "alt")
			img := renderMediaImg(imageID, alt, alt, "", mediaByID)
			if href := dataString(item, "href"); href != "" {
				b.WriteString(fmt.Sprintf("      <li><a href=\"%s\">%s</a></li>\n", escapeURL(href), img))
			} else {
				b.WriteString(fmt.Sprintf("      <li>%s</li>\n", img))
			}
		}
		b.WriteString("    </ul>\n")
	}
	b.WriteString("  </section>\n")
	return b.String()
}

// renderCodeBlock renders a monospace code presentation with optional
// language label. Data: {language?, code, label?}. The eval-friendly
// shape so technical-blog-style pages can ship code without raw HTML.
func renderCodeBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--code_block\">\n")
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if lang := dataString(data, "language"); lang != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"code-label\">%s</p>\n", escapeHTML(lang)))
	}
	if code := dataString(data, "code"); code != "" {
		b.WriteString(fmt.Sprintf("    <pre><code>%s</code></pre>\n", escapeHTML(code)))
	}
	b.WriteString("  </section>\n")
	return b.String()
}

// renderFormBlock renders a basic HTML form. Data shape:
// {heading?, subheading?, action, method?, fields: [{name, type, label, placeholder?, required?, options?}], submit_label}.
// type can be: text, email, tel, url, textarea, select, checkbox, radio.
// Browser submits to action; the operator wires the receiving endpoint.
func renderFormBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--form\">\n")
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if sub := dataString(data, "subheading"); sub != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"subheading\">%s</p>\n", escapeHTML(sub)))
	}
	action := dataString(data, "action")
	if action == "" {
		action = "#"
	}
	method := strings.ToLower(dataString(data, "method"))
	if method != "get" {
		method = "post"
	}
	b.WriteString(fmt.Sprintf("    <form action=\"%s\" method=\"%s\">\n", escapeURL(action), method))
	if fieldsRaw, ok := data["fields"].([]any); ok {
		for _, f := range fieldsRaw {
			field, ok := f.(map[string]any)
			if !ok {
				continue
			}
			name := dataString(field, "name")
			ftype := dataString(field, "type")
			label := dataString(field, "label")
			placeholder := dataString(field, "placeholder")
			required, _ := field["required"].(bool)
			if name == "" || ftype == "" {
				continue
			}
			b.WriteString("      <div class=\"form-field\">\n")
			if label != "" {
				b.WriteString(fmt.Sprintf("        <label for=\"f-%s\">%s</label>\n", escapeAttr(name), escapeHTML(label)))
			}
			reqAttr := ""
			if required {
				reqAttr = " required"
			}
			switch ftype {
			case "textarea":
				b.WriteString(fmt.Sprintf("        <textarea id=\"f-%s\" name=\"%s\" placeholder=\"%s\"%s></textarea>\n",
					escapeAttr(name), escapeAttr(name), escapeAttr(placeholder), reqAttr))
			case "select":
				b.WriteString(fmt.Sprintf("        <select id=\"f-%s\" name=\"%s\"%s>\n",
					escapeAttr(name), escapeAttr(name), reqAttr))
				if optsRaw, ok := field["options"].([]any); ok {
					for _, o := range optsRaw {
						if os, ok := o.(string); ok {
							b.WriteString(fmt.Sprintf("          <option value=\"%s\">%s</option>\n",
								escapeAttr(os), escapeHTML(os)))
						}
					}
				}
				b.WriteString("        </select>\n")
			default:
				// text, email, tel, url, checkbox, radio (basic: no group support yet).
				b.WriteString(fmt.Sprintf("        <input type=\"%s\" id=\"f-%s\" name=\"%s\" placeholder=\"%s\"%s />\n",
					escapeAttr(ftype), escapeAttr(name), escapeAttr(name), escapeAttr(placeholder), reqAttr))
			}
			b.WriteString("      </div>\n")
		}
	}
	submitLabel := dataString(data, "submit_label")
	if submitLabel == "" {
		submitLabel = "Submit"
	}
	b.WriteString(fmt.Sprintf("      <button type=\"submit\">%s</button>\n", escapeHTML(submitLabel)))
	b.WriteString("    </form>\n")
	b.WriteString("  </section>\n")
	return b.String()
}

// renderLogoCarouselBlock renders a horizontal infinite-scroll marquee of
// customer / partner logos. Pure CSS animation (no JS). Items duplicate
// once in markup so the loop is seamless. Use under hero on a real
// production marketing site to anchor trust before the first feature
// section. Each item: {image_id, alt, href?} OR {label, href?} for
// text-mark logos when an image isn't available.
func renderLogoCarouselBlock(data map[string]any, mediaByID map[string]store.Medium) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--logo_carousel\">\n")
	if label := dataString(data, "label"); label != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"logo-carousel-label\">%s</p>\n", escapeHTML(label)))
	}
	itemsRaw, ok := data["items"].([]any)
	if !ok || len(itemsRaw) == 0 {
		b.WriteString("  </section>\n")
		return b.String()
	}
	// Render the inner ul twice so the CSS animation can loop seamlessly.
	b.WriteString("    <div class=\"logo-carousel\" aria-label=\"Trusted by\">\n")
	for pass := 0; pass < 2; pass++ {
		ariaHidden := ""
		if pass == 1 {
			ariaHidden = ` aria-hidden="true"`
		}
		b.WriteString(fmt.Sprintf("      <ul class=\"logo-carousel-track\"%s>\n", ariaHidden))
		for _, it := range itemsRaw {
			item, ok := it.(map[string]any)
			if !ok {
				continue
			}
			imageID := dataString(item, "image_id")
			label := dataString(item, "label")
			alt := dataString(item, "alt")
			if alt == "" {
				alt = label
			}
			if imageID == "" && label == "" {
				continue
			}
			href := dataString(item, "href")
			b.WriteString("        <li>")
			if href != "" {
				b.WriteString(fmt.Sprintf(`<a href="%s">`, escapeURL(href)))
			}
			if imageID != "" {
				b.WriteString(renderMediaImg(imageID, alt, label, "carousel-img", mediaByID))
			} else {
				b.WriteString(fmt.Sprintf(`<span class="carousel-mark">%s</span>`, escapeHTML(label)))
			}
			if href != "" {
				b.WriteString("</a>")
			}
			b.WriteString("</li>\n")
		}
		b.WriteString("      </ul>\n")
	}
	b.WriteString("    </div>\n")
	b.WriteString("  </section>\n")
	return b.String()
}

// renderReplacementGridBlock renders a bento-style grid of "old → new"
// replacement cards. Each card shows the SaaS being replaced with a
// strikethrough on the "from" name + arrow + bold replacement, plus a
// short description below. Some items can span 2 columns via
// items[i].span ("wide") to break up the grid rhythm. Matches the
// brightinteraction.com /#services bento pattern.
//
// Each item: {from, to, description, span?, icon?}.
// span values: "" (1 col, default), "wide" (2 cols on md+).
func renderReplacementGridBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--replacement_grid\">\n")
	if eyebrow := dataString(data, "eyebrow"); eyebrow != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"eyebrow\">%s</p>\n", escapeHTML(eyebrow)))
	}
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if sub := dataString(data, "subheading"); sub != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"subheading\">%s</p>\n", escapeHTML(sub)))
	}
	if itemsRaw, ok := data["items"].([]any); ok && len(itemsRaw) > 0 {
		b.WriteString("    <ul class=\"replacement-grid\">\n")
		for _, it := range itemsRaw {
			item, ok := it.(map[string]any)
			if !ok {
				continue
			}
			from := dataString(item, "from")
			to := dataString(item, "to")
			desc := dataString(item, "description")
			span := dataString(item, "span")
			cls := "replacement-card"
			if span == "wide" {
				cls += " is-wide"
			}
			b.WriteString(fmt.Sprintf("      <li class=\"%s\">\n", cls))
			b.WriteString("        <div class=\"replacement-pair\">\n")
			if from != "" {
				b.WriteString(fmt.Sprintf("          <span class=\"replacement-from\">%s</span>\n", escapeHTML(from)))
			}
			b.WriteString("          <svg class=\"replacement-arrow\" viewBox=\"0 0 24 24\" width=\"16\" height=\"16\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\" aria-hidden=\"true\"><path d=\"M5 12h14\"/><path d=\"m12 5 7 7-7 7\"/></svg>\n")
			if to != "" {
				b.WriteString(fmt.Sprintf("          <span class=\"replacement-to\">%s</span>\n", escapeHTML(to)))
			}
			b.WriteString("        </div>\n")
			if desc != "" {
				b.WriteString(fmt.Sprintf("        <p class=\"replacement-desc\">%s</p>\n", escapeHTML(desc)))
			}
			b.WriteString("      </li>\n")
		}
		b.WriteString("    </ul>\n")
	}
	if footer := dataString(data, "footer"); footer != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"replacement-footer\">%s</p>\n", escapeHTML(footer)))
	}
	b.WriteString("  </section>\n")
	return b.String()
}

// renderCalculatorBlock is intentionally NOT registered in renderDataBlock's
// switch. Calculators are bespoke per customer (the cost formula, the
// services list, the currency, the layout — all site-specific). Atomicsite
// provides the general primitives (form, feature_grid, replacement_grid,
// embed, raw) so an agent or developer can hand-build the calculator for
// the customer who needs one. This function is kept as a reference shape
// for that custom build, demonstrating the right pattern: data-config JSON
// + inline <script> hydrated by a small vanilla-JS reader. See the demo
// site's component catalog if you need the actual implementation.
func renderCalculatorBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--calculator\">\n")
	if eyebrow := dataString(data, "eyebrow"); eyebrow != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"eyebrow\">%s</p>\n", escapeHTML(eyebrow)))
	}
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if sub := dataString(data, "subheading"); sub != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"subheading\">%s</p>\n", escapeHTML(sub)))
	}

	// JSON config blob the embedded JS reads. We pass it via a JSON-encoded
	// data attribute (rather than a separate <script> tag) so the CSP path
	// stays clean.
	configBytes, _ := json.Marshal(map[string]any{
		"services":            data["services"],
		"currency_symbol":     dataString(data, "currency_symbol"),
		"currency_rate":       data["currency_rate"],
		"team_size_min":       data["team_size_min"],
		"team_size_max":       data["team_size_max"],
		"team_size_default":   data["team_size_default"],
		"hardware_monthly":    data["hardware_monthly"],
		"maintenance_yearly":  data["maintenance_yearly"],
		"labels":              data["labels"],
	})

	b.WriteString(fmt.Sprintf("    <div class=\"calculator-root\" data-config='%s'>\n",
		escapeAttr(string(configBytes))))

	// Service tile grid. Initial markup uses the operator-supplied selected
	// flags so the form looks right pre-hydration; the JS replays the same
	// state on load and then takes over.
	b.WriteString("      <div class=\"calculator-services\" role=\"group\" aria-label=\"Pick the tools you currently pay for\">\n")
	if servicesRaw, ok := data["services"].([]any); ok {
		for _, sv := range servicesRaw {
			svc, ok := sv.(map[string]any)
			if !ok {
				continue
			}
			key := dataString(svc, "key")
			label := dataString(svc, "label")
			brand := dataString(svc, "saas_brand")
			selected, _ := svc["selected"].(bool)
			selCls := ""
			if selected {
				selCls = " is-selected"
			}
			b.WriteString(fmt.Sprintf("        <button type=\"button\" class=\"calculator-service%s\" data-key=\"%s\" data-selected=\"%t\">\n",
				selCls, escapeAttr(key), selected))
			b.WriteString("          <span class=\"calculator-service-body\">\n")
			b.WriteString(fmt.Sprintf("            <span class=\"calculator-service-label\">%s</span>\n", escapeHTML(label)))
			if brand != "" {
				b.WriteString(fmt.Sprintf("            <span class=\"calculator-service-brand\">%s</span>\n", escapeHTML(brand)))
			}
			b.WriteString("          </span>\n")
			b.WriteString("          <span class=\"calculator-service-check\" aria-hidden=\"true\">✓</span>\n")
			b.WriteString("        </button>\n")
		}
	}
	b.WriteString("      </div>\n")

	// Team-size slider.
	teamMin := intOrDefault(data["team_size_min"], 1)
	teamMax := intOrDefault(data["team_size_max"], 100)
	teamDefault := intOrDefault(data["team_size_default"], 10)
	b.WriteString("      <div class=\"calculator-slider\">\n")
	b.WriteString("        <label for=\"calc-team-size\">Team size</label>\n")
	b.WriteString(fmt.Sprintf("        <input type=\"range\" id=\"calc-team-size\" min=\"%d\" max=\"%d\" value=\"%d\" />\n",
		teamMin, teamMax, teamDefault))
	b.WriteString(fmt.Sprintf("        <output for=\"calc-team-size\" id=\"calc-team-display\">%d</output>\n", teamDefault))
	b.WriteString("        <span class=\"calculator-slider-unit\">people</span>\n")
	b.WriteString("      </div>\n")

	// Three-column result. JS fills in the numbers; markup is the skeleton.
	b.WriteString("      <div class=\"calculator-results\">\n")
	for _, col := range []struct {
		key, title, kind string
	}{
		{"saas", "Your SaaS cost", "saas"},
		{"hardware", "Hardware only", "hardware"},
		{"selfhosted", "Self-hosted with us", "selfhosted"},
	} {
		b.WriteString(fmt.Sprintf("        <div class=\"calculator-col is-%s\">\n", col.kind))
		b.WriteString(fmt.Sprintf("          <p class=\"calculator-col-label\">%s</p>\n", col.title))
		b.WriteString(fmt.Sprintf("          <p class=\"calculator-col-monthly\" data-out=\"%s-monthly\">—</p>\n", col.key))
		b.WriteString(fmt.Sprintf("          <p class=\"calculator-col-yearly\" data-out=\"%s-yearly\">—</p>\n", col.key))
		b.WriteString("        </div>\n")
	}
	b.WriteString("      </div>\n")

	if ctaText := dataString(data, "cta_text"); ctaText != "" {
		ctaURL := dataString(data, "cta_url")
		if ctaURL == "" {
			ctaURL = "#"
		}
		b.WriteString(fmt.Sprintf("      <div class=\"calculator-cta\"><a href=\"%s\" class=\"btn-primary\">%s</a></div>\n",
			escapeURL(ctaURL), escapeHTML(ctaText)))
	}
	b.WriteString("    </div>\n")
	b.WriteString("  </section>\n")
	return b.String()
}

func intOrDefault(v any, def int) int {
	if v == nil {
		return def
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		// best-effort parse
		var n int
		_, err := fmt.Sscanf(x, "%d", &n)
		if err == nil {
			return n
		}
	}
	return def
}

// renderCustomBlock is the agnostic escape-hatch block. The agent writes
// arbitrary markup (Tailwind classes, atomicsite design tokens, semantic
// HTML); atomicsite emits it verbatim wrapped in a <section> with auto-id
// from the block's name. Used when a customer needs a layout that doesn't
// fit the typed templates (hero, feature_grid, pricing, etc.).
//
// Data shape: {name, markup, eyebrow?, props?}.
//   - name: human label, also drives the section id (slugified) so cross-
//     section anchors resolve.
//   - markup: full HTML for the section body. atomicsite's design-token
//     CSS variables (--color-primary, --color-text, --color-bg, --bp-*,
//     --container-width, etc.) are available; baseline utility classes
//     (.btn-primary, .btn-secondary, .actions, .container) work too.
//   - eyebrow: optional small-caps label rendered before the markup.
//   - props: optional object the agent can read at edit time (e.g. for
//     decisions in a future code-mod step). Not rendered.
//
// Security: the existing block-time guardrail rejects <script> and
// <iframe> in markup unless explicitly approved; agent-supplied alt
// text + plaintext-email rules still apply. For iframes use the `embed`
// block (CSP-aware); for scripts use a registered component (component
// block_type).
func renderCustomBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--custom\">\n")
	if eyebrow := dataString(data, "eyebrow"); eyebrow != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"eyebrow\">%s</p>\n", escapeHTML(eyebrow)))
	}
	if markup := dataString(data, "markup"); markup != "" {
		b.WriteString("    " + markup + "\n")
	}
	b.WriteString("  </section>\n")
	return b.String()
}

// renderProcessStepsBlock renders a numbered 4-up grid of process steps.
// Each step shows a big primary-coloured number + h3 title + body. Use for
// "How it works" / "Our process" / "How to get started" sections — the
// canonical 4-step marketing pattern.
//
// Each item: {number, title, description}. Number is a string (e.g. "01",
// "02") so the renderer doesn't second-guess formatting.
func renderProcessStepsBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--process_steps\">\n")
	if eyebrow := dataString(data, "eyebrow"); eyebrow != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"eyebrow\">%s</p>\n", escapeHTML(eyebrow)))
	}
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if sub := dataString(data, "subheading"); sub != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"subheading\">%s</p>\n", escapeHTML(sub)))
	}
	if itemsRaw, ok := data["items"].([]any); ok && len(itemsRaw) > 0 {
		b.WriteString("    <ol class=\"process-steps\">\n")
		for i, it := range itemsRaw {
			item, ok := it.(map[string]any)
			if !ok {
				continue
			}
			num := dataString(item, "number")
			if num == "" {
				num = fmt.Sprintf("%02d", i+1)
			}
			b.WriteString("      <li class=\"process-step\">\n")
			b.WriteString(fmt.Sprintf("        <div class=\"process-step-num\">%s</div>\n", escapeHTML(num)))
			if title := dataString(item, "title"); title != "" {
				b.WriteString(fmt.Sprintf("        <h3 class=\"process-step-title\">%s</h3>\n", escapeHTML(title)))
			}
			if desc := dataString(item, "description"); desc != "" {
				b.WriteString(fmt.Sprintf("        <p class=\"process-step-desc\">%s</p>\n", escapeHTML(desc)))
			}
			b.WriteString("      </li>\n")
		}
		b.WriteString("    </ol>\n")
	}
	b.WriteString("  </section>\n")
	return b.String()
}

// renderAboutSplitBlock renders a side-by-side about / founder section
// with content on the left + photo on the right (stacks on mobile, photo
// above content). Includes optional stats row + secondary text link CTA
// underneath. Use for founder bios, "About us", team intros — anywhere
// you want a personal photo paired with narrative.
//
// Data: {eyebrow, heading, paragraphs[], image_id, image_alt, image_position,
//        stats: [{value, label}], cta_text, cta_url}.
// image_position: "right" (default) or "left".
func renderAboutSplitBlock(data map[string]any, mediaByID map[string]store.Medium) string {
	var b strings.Builder
	pos := dataString(data, "image_position")
	cls := "block block--about_split"
	if pos == "left" {
		cls += " is-image-left"
	}
	b.WriteString(fmt.Sprintf("  <section class=\"%s\">\n", cls))
	b.WriteString("    <div class=\"about-split-content\">\n")
	if eyebrow := dataString(data, "eyebrow"); eyebrow != "" {
		b.WriteString(fmt.Sprintf("      <p class=\"eyebrow\">%s</p>\n", escapeHTML(eyebrow)))
	}
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("      <h2>%s</h2>\n", escapeHTML(heading)))
	}
	if parasRaw, ok := data["paragraphs"].([]any); ok {
		for _, p := range parasRaw {
			if ps, ok := p.(string); ok && ps != "" {
				b.WriteString(fmt.Sprintf("      <p>%s</p>\n", escapeHTML(ps)))
			}
		}
	} else if text := dataString(data, "text"); text != "" {
		// Fall back to the same paragraph splitter the text block uses.
		renderTextParagraphs(&b, text)
	}
	if statsRaw, ok := data["stats"].([]any); ok && len(statsRaw) > 0 {
		b.WriteString("      <ul class=\"about-split-stats\">\n")
		for _, s := range statsRaw {
			st, ok := s.(map[string]any)
			if !ok {
				continue
			}
			b.WriteString("        <li>\n")
			if v := dataString(st, "value"); v != "" {
				b.WriteString(fmt.Sprintf("          <div class=\"about-split-stat-value\">%s</div>\n", escapeHTML(v)))
			}
			if l := dataString(st, "label"); l != "" {
				b.WriteString(fmt.Sprintf("          <div class=\"about-split-stat-label\">%s</div>\n", escapeHTML(l)))
			}
			b.WriteString("        </li>\n")
		}
		b.WriteString("      </ul>\n")
	}
	if ctaText := dataString(data, "cta_text"); ctaText != "" {
		ctaURL := dataString(data, "cta_url")
		if ctaURL == "" {
			ctaURL = "#"
		}
		b.WriteString(fmt.Sprintf("      <a href=\"%s\" class=\"about-split-link\">%s →</a>\n",
			escapeURL(ctaURL), escapeHTML(ctaText)))
	}
	b.WriteString("    </div>\n")
	b.WriteString("    <div class=\"about-split-image\">\n")
	if imageID := dataString(data, "image_id"); imageID != "" {
		b.WriteString("      " + renderMediaImg(imageID, dataString(data, "image_alt"), dataString(data, "heading"), "about-split-img", mediaByID) + "\n")
	}
	b.WriteString("    </div>\n")
	b.WriteString("  </section>\n")
	return b.String()
}

// renderEmbedBlock renders an iframe wrapped in an aspect-ratio container.
// The src host MUST already be in the trusted-domains allowlist (kind=frame)
// or the build will pass but the browser will block via CSP frame-src. The
// admin grants frame access via /sites/{id}/settings/allowed-scripts.
// Data: {src, title, aspect_ratio?}.
func renderEmbedBlock(data map[string]any) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--embed\">\n")
	if heading := dataString(data, "heading"); heading != "" {
		b.WriteString(fmt.Sprintf("    <h2>%s</h2>\n", escapeHTML(heading)))
	}
	src := dataString(data, "src")
	if src != "" {
		title := dataString(data, "title")
		if title == "" {
			title = "Embedded content"
		}
		b.WriteString("    <div class=\"embed-wrapper\">\n")
		b.WriteString(fmt.Sprintf("      <iframe src=\"%s\" title=\"%s\" loading=\"lazy\" referrerpolicy=\"no-referrer-when-downgrade\"></iframe>\n",
			escapeURL(src), escapeAttr(title)))
		b.WriteString("    </div>\n")
	}
	b.WriteString("  </section>\n")
	return b.String()
}
