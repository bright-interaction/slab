package builder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bright-interaction/slab/internal/store"
)

// renderSplitHeroBlock renders a side-by-side hero (image right + text left
// on desktop, stacked on mobile). Use when the page wants a SaaS-marketing
// shape rather than the centered hero. Layout flips to centered when the
// site setting general.hero_layout=centered or the block's data.layout=centered.
func renderSplitHeroBlock(data map[string]any, mediaByID map[string]store.Medium) string {
	var b strings.Builder
	b.WriteString("  <section class=\"block block--split_hero\">\n")
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
