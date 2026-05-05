package blocks

// init registers a schema for every block_type the renderer in
// internal/builder/pages.go (renderDataBlock) accepts. When you add a
// new block_type to the renderer, add its schema here too. Both lookups
// (editor form, agent context) read from this single map.
func init() {
	Register(Schema{
		Type: "hero", Label: "Hero", Category: "hero",
		Description: "Above-the-fold attention-grab. One per page. Drives a primary CTA.",
		Fields: []Field{
			{Key: "eyebrow", Label: "Eyebrow", Kind: KindText, Help: "Small mono uppercase label above the headline."},
			{Key: "headline", Label: "Headline", Kind: KindTextarea, Help: "Multi-line. Each newline becomes a visual <br>. Wrap an inline accent fragment in [[double brackets]] to colour it with --color-primary."},
			{Key: "headline_accent", Label: "Headline accent", Kind: KindTextarea, Help: "Optional second line in --color-primary (alternative to [[brackets]])."},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "image_id", Label: "Background image", Kind: KindImageID},
			{Key: "image_alt", Label: "Image alt text", Kind: KindText},
			{Key: "cta_text", Label: "Primary CTA label", Kind: KindText},
			{Key: "cta_url", Label: "Primary CTA URL", Kind: KindURL},
			{Key: "secondary_label", Label: "Secondary CTA label", Kind: KindText},
			{Key: "secondary_url", Label: "Secondary CTA URL", Kind: KindURL},
			{Key: "bg", Label: "Background mode", Kind: KindSelect, Options: []Option{
				{Value: "", Label: "None"},
				{Value: "circuit", Label: "Animated circuit"},
				{Value: "circuit-static", Label: "Static circuit pattern"},
			}},
		},
	})

	Register(Schema{
		Type: "split_hero", Label: "Split hero", Category: "hero",
		Description: "Side-by-side hero with text on the left and image on the right (stacks on mobile).",
		Fields: []Field{
			{Key: "eyebrow", Label: "Eyebrow", Kind: KindText},
			{Key: "headline", Label: "Headline", Kind: KindTextarea, Help: "Multi-line. Each newline becomes a visual <br>. Use [[double brackets]] for an inline accent fragment."},
			{Key: "headline_accent", Label: "Headline accent", Kind: KindTextarea},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "image_id", Label: "Right column image", Kind: KindImageID},
			{Key: "image_alt", Label: "Image alt text", Kind: KindText},
			{Key: "cta_text", Label: "Primary CTA label", Kind: KindText},
			{Key: "cta_url", Label: "Primary CTA URL", Kind: KindURL},
			{Key: "secondary_label", Label: "Secondary CTA label", Kind: KindText},
			{Key: "secondary_url", Label: "Secondary CTA URL", Kind: KindURL},
			{Key: "bg", Label: "Background mode", Kind: KindSelect, Options: []Option{
				{Value: "", Label: "None"},
				{Value: "circuit", Label: "Animated circuit"},
				{Value: "circuit-static", Label: "Static circuit pattern"},
			}},
			{Key: "layout", Label: "Layout", Kind: KindSelect, Options: []Option{
				{Value: "", Label: "Side-by-side (default)"},
				{Value: "centered", Label: "Centered"},
			}},
		},
	})

	Register(Schema{
		Type: "feature_grid", Label: "Feature grid", Category: "content",
		Description: "Three- to four-up grid of feature cards.",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "items", Label: "Features", Kind: KindArray, ItemSchema: []Field{
				{Key: "title", Label: "Title", Kind: KindText, Required: true},
				{Key: "icon", Label: "Lucide icon name", Kind: KindText, Help: "e.g. shield, zap, code"},
				{Key: "body", Label: "Body", Kind: KindTextarea},
			}},
		},
	})

	Register(Schema{
		Type: "stat_grid", Label: "Stat grid", Category: "social proof",
		Description: "Trust signals: large numbers + labels. Counts as list>=3 for AI-Friendly Formatting.",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "items", Label: "Stats", Kind: KindArray, ItemSchema: []Field{
				{Key: "value", Label: "Value", Kind: KindText, Placeholder: "98%", Required: true},
				{Key: "label", Label: "Label", Kind: KindText, Required: true},
				{Key: "context", Label: "Context", Kind: KindText, Help: "Small grey caption under the label."},
			}},
		},
	})

	Register(Schema{
		Type: "accordion_faq", Label: "FAQ accordion", Category: "content",
		Description: "Q&A using native <details>/<summary>. Auto-emits FAQPage JSON-LD for SEO.",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "items", Label: "Questions", Kind: KindArray, ItemSchema: []Field{
				{Key: "question", Label: "Question", Kind: KindText, Required: true},
				{Key: "answer", Label: "Answer", Kind: KindTextarea, Required: true},
			}},
		},
	})

	Register(Schema{
		Type: "pricing", Label: "Pricing tiers", Category: "conversion",
		Description: "Three-up pricing cards with bullets + per-tier CTA.",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "tiers", Label: "Tiers", Kind: KindArray, ItemSchema: []Field{
				{Key: "step", Label: "Step eyebrow", Kind: KindText, Help: "Optional small mono uppercase 'STEP 01' above the tier name."},
				{Key: "name", Label: "Name", Kind: KindText, Required: true},
				{Key: "price", Label: "Price", Kind: KindText, Required: true},
				{Key: "price_period", Label: "Price period", Kind: KindText, Placeholder: "/month"},
				{Key: "description", Label: "Description", Kind: KindTextarea},
				{Key: "features", Label: "Features", Kind: KindArray, ItemSchema: []Field{
					{Key: "", Label: "Feature", Kind: KindText, Required: true},
				}},
				{Key: "cta_text", Label: "CTA label", Kind: KindText},
				{Key: "cta_url", Label: "CTA URL", Kind: KindURL},
				{Key: "featured", Label: "Featured tier", Kind: KindBool, Help: "Dark fill + accent border."},
			}},
		},
	})

	Register(Schema{
		Type: "logo_strip", Label: "Logo strip", Category: "social proof",
		Description: "Row of customer/partner logos.",
		Fields: []Field{
			{Key: "label", Label: "Label", Kind: KindText, Placeholder: "Trusted by"},
			{Key: "items", Label: "Logos", Kind: KindArray, ItemSchema: []Field{
				{Key: "image_id", Label: "Logo image", Kind: KindImageID, Required: true},
				{Key: "alt", Label: "Alt text", Kind: KindText, Required: true},
				{Key: "href", Label: "Link", Kind: KindURL},
			}},
		},
	})

	Register(Schema{
		Type: "logo_carousel", Label: "Logo carousel", Category: "social proof",
		Description: "Infinite-scroll marquee of customer/partner logos. CSS only.",
		Fields: []Field{
			{Key: "label", Label: "Label", Kind: KindText, Placeholder: "Trusted by"},
			{Key: "items", Label: "Logos", Kind: KindArray, ItemSchema: []Field{
				{Key: "image_id", Label: "Logo image", Kind: KindImageID, Help: "Falls back to text mark below if empty."},
				{Key: "label", Label: "Text mark", Kind: KindText, Help: "Used when no image is provided."},
				{Key: "alt", Label: "Alt text", Kind: KindText},
				{Key: "href", Label: "Link", Kind: KindURL},
			}},
		},
	})

	Register(Schema{
		Type: "replacement_grid", Label: "Replacement grid", Category: "content",
		Description: "Bento grid of 'old → new' replacement cards.",
		Fields: []Field{
			{Key: "eyebrow", Label: "Eyebrow", Kind: KindText},
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "items", Label: "Replacements", Kind: KindArray, ItemSchema: []Field{
				{Key: "from", Label: "From (struck-through)", Kind: KindText, Required: true},
				{Key: "to", Label: "To (bold)", Kind: KindText, Required: true},
				{Key: "description", Label: "Description", Kind: KindTextarea},
				{Key: "span", Label: "Width", Kind: KindSelect, Options: []Option{
					{Value: "", Label: "Single column"},
					{Value: "wide", Label: "Two columns"},
				}},
			}},
			{Key: "footer", Label: "Footer line", Kind: KindText},
		},
	})

	Register(Schema{
		Type: "process_steps", Label: "Process steps", Category: "content",
		Description: "Numbered 4-up grid for 'How it works' sections.",
		Fields: []Field{
			{Key: "eyebrow", Label: "Eyebrow", Kind: KindText},
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "items", Label: "Steps", Kind: KindArray, ItemSchema: []Field{
				{Key: "number", Label: "Number", Kind: KindText, Placeholder: "01"},
				{Key: "title", Label: "Title", Kind: KindText, Required: true},
				{Key: "description", Label: "Description", Kind: KindTextarea},
			}},
		},
	})

	Register(Schema{
		Type: "about_split", Label: "About split", Category: "content",
		Description: "Side-by-side founder/about with photo and stats.",
		Fields: []Field{
			{Key: "eyebrow", Label: "Eyebrow", Kind: KindText},
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "paragraphs", Label: "Paragraphs", Kind: KindArray, ItemSchema: []Field{
				{Key: "", Label: "Paragraph", Kind: KindTextarea, Required: true},
			}},
			{Key: "image_id", Label: "Photo", Kind: KindImageID},
			{Key: "image_alt", Label: "Photo alt text", Kind: KindText},
			{Key: "image_position", Label: "Photo position", Kind: KindSelect, Options: []Option{
				{Value: "right", Label: "Right (default)"},
				{Value: "left", Label: "Left"},
			}},
			{Key: "stats", Label: "Stats", Kind: KindArray, ItemSchema: []Field{
				{Key: "value", Label: "Value", Kind: KindText, Required: true},
				{Key: "label", Label: "Label", Kind: KindText, Required: true},
			}},
			{Key: "cta_text", Label: "Secondary link text", Kind: KindText},
			{Key: "cta_url", Label: "Secondary link URL", Kind: KindURL},
		},
	})

	Register(Schema{
		Type: "text", Label: "Text", Category: "content",
		Description: "Long-form copy. Heading + body + optional CTA.",
		Fields: []Field{
			{Key: "eyebrow", Label: "Eyebrow", Kind: KindText},
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "text", Label: "Body", Kind: KindTextarea, Help: "Blank lines split paragraphs. Single newlines become <br>."},
			{Key: "cta_text", Label: "CTA label", Kind: KindText},
			{Key: "cta_url", Label: "CTA URL", Kind: KindURL},
			{Key: "alignment", Label: "Alignment", Kind: KindSelect, Options: []Option{
				{Value: "left", Label: "Left"},
				{Value: "center", Label: "Center"},
			}},
		},
	})

	Register(Schema{
		Type: "cta", Label: "Call to action", Category: "conversion",
		Description: "Standalone CTA banner with heading, lead-in text and a button.",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText, Required: true},
			{Key: "text", Label: "Lead-in text", Kind: KindTextarea},
			{Key: "cta_text", Label: "Button label", Kind: KindText, Required: true},
			{Key: "cta_url", Label: "Button URL", Kind: KindURL},
			{Key: "variant", Label: "Variant", Kind: KindSelect, Options: []Option{
				{Value: "primary", Label: "Primary (tinted)"},
				{Value: "secondary", Label: "Secondary (muted)"},
			}},
		},
	})

	Register(Schema{
		Type: "image", Label: "Image", Category: "content",
		Description: "Single image with optional caption.",
		Fields: []Field{
			{Key: "image_id", Label: "Image", Kind: KindImageID, Required: true},
			{Key: "alt", Label: "Alt text", Kind: KindText, Required: true, Help: "Always describe the image. Never 'image' or 'photo'."},
			{Key: "caption", Label: "Caption", Kind: KindText},
		},
	})

	Register(Schema{
		Type: "quote", Label: "Quote", Category: "content",
		Description: "Testimonial / pull quote.",
		Fields: []Field{
			{Key: "quote", Label: "Quote", Kind: KindTextarea, Required: true},
			{Key: "attribution", Label: "Attribution", Kind: KindText},
		},
	})

	Register(Schema{
		Type: "code_block", Label: "Code block", Category: "content",
		Description: "Monospace code presentation with optional language label.",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "language", Label: "Language", Kind: KindText, Placeholder: "go"},
			{Key: "code", Label: "Code", Kind: KindTextarea, Required: true},
		},
	})

	Register(Schema{
		Type: "form", Label: "Form", Category: "conversion",
		Description: "Basic HTML form. Browser POSTs to the action URL.",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "action", Label: "Action URL", Kind: KindURL, Required: true},
			{Key: "method", Label: "Method", Kind: KindSelect, Options: []Option{
				{Value: "post", Label: "POST"},
				{Value: "get", Label: "GET"},
			}},
			{Key: "fields", Label: "Fields", Kind: KindArray, ItemSchema: []Field{
				{Key: "name", Label: "Name", Kind: KindText, Required: true},
				{Key: "type", Label: "Type", Kind: KindSelect, Required: true, Options: []Option{
					{Value: "text", Label: "Text"},
					{Value: "email", Label: "Email"},
					{Value: "tel", Label: "Telephone"},
					{Value: "url", Label: "URL"},
					{Value: "textarea", Label: "Textarea"},
					{Value: "select", Label: "Select"},
					{Value: "checkbox", Label: "Checkbox"},
					{Value: "radio", Label: "Radio"},
				}},
				{Key: "label", Label: "Label", Kind: KindText, Required: true},
				{Key: "placeholder", Label: "Placeholder", Kind: KindText},
				{Key: "required", Label: "Required", Kind: KindBool},
				{Key: "options", Label: "Options (select only)", Kind: KindArray, ItemSchema: []Field{
					{Key: "", Label: "Option", Kind: KindText, Required: true},
				}},
			}},
			{Key: "submit_label", Label: "Submit button label", Kind: KindText, Placeholder: "Submit"},
		},
	})

	Register(Schema{
		Type: "embed", Label: "Embed", Category: "content",
		Description: "Iframe in a 16:9 wrapper. Src host MUST be in trusted_domains (kind=frame).",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "src", Label: "Iframe src URL", Kind: KindURL, Required: true},
			{Key: "title", Label: "Iframe title", Kind: KindText, Required: true, Help: "For screen readers."},
			{Key: "aspect_ratio", Label: "Aspect ratio", Kind: KindText, Placeholder: "16/9"},
		},
	})

	Register(Schema{
		Type: "custom", Label: "Custom HTML", Category: "escape hatch",
		Description: "Hand-written markup wrapped in a <section>. Use when no typed primitive fits.",
		Fields: []Field{
			{Key: "name", Label: "Section name", Kind: KindText, Required: true, Help: "Drives the auto-generated section id."},
			{Key: "eyebrow", Label: "Eyebrow", Kind: KindText},
			{Key: "markup", Label: "HTML markup", Kind: KindTextarea, Required: true, Help: "Atomicsite design tokens (--color-*, --font-*) and utility classes (.btn-primary, .container) are available."},
		},
	})

	Register(Schema{
		Type: "raw_astro", Label: "Raw Astro", Category: "escape hatch",
		Description: "Raw Astro/HTML/Tailwind code emitted verbatim. No auto chrome.",
		Fields: []Field{
			{Key: "code", Label: "Astro/HTML code", Kind: KindTextarea, Required: true, Help: "Wrapped in <section> only when not opening with one. No inline <script>; CSP blocks them."},
			{Key: "class", Label: "Section class", Kind: KindText, Help: "Added to the auto-section wrapper, ignored when code already opens with a section/header/etc."},
		},
	})

	// Sprint 4 (2026-05-04): Custom Collections list block. Renders
	// N items from a Collection at build time. Items themselves
	// already render as standalone pages (when settings.render_as_pages
	// is true); this block lets any page show a curated subset.
	Register(Schema{
		Type: "collection_list", Label: "Collection list", Category: "content",
		Description: "Renders items from a Custom Collection (case studies, products, team members, etc.) at build time. Sort, limit, and optionally filter by visitor metadata via the personalization DSL.",
		Fields: []Field{
			{Key: "collection_id", Label: "Collection", Kind: KindText, Required: true, Help: "ID of the Collection to pull items from. Pick from /api/sites/{siteID}/collections."},
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "limit", Label: "Max items", Kind: KindNumber, Help: "Default 12. Use 0 for unlimited (subject to a 200 cap)."},
			{Key: "sort_by", Label: "Sort by", Kind: KindSelect, Options: []Option{
				{Value: "sort_order", Label: "Manual order"},
				{Value: "published_at", Label: "Date published"},
				{Value: "created_at", Label: "Date created"},
				{Value: "title", Label: "Title (A-Z)"},
			}},
			{Key: "sort_dir", Label: "Sort direction", Kind: KindSelect, Options: []Option{
				{Value: "asc", Label: "Ascending"},
				{Value: "desc", Label: "Descending"},
			}},
			{Key: "card_template", Label: "Card layout", Kind: KindSelect, Options: []Option{
				{Value: "title_image", Label: "Title above image"},
				{Value: "image_title", Label: "Image above title"},
				{Value: "compact", Label: "Compact list"},
				{Value: "grid_2", Label: "Two-column grid"},
				{Value: "grid_3", Label: "Three-column grid"},
			}},
			{Key: "filter", Label: "Static filter (DSL)", Kind: KindText, Help: `Personalization-DSL filter applied at build time, e.g. industry == "finance" OR featured present.`},
		},
	})
}
