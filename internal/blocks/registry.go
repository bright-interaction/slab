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
			{Key: "hero_graphic", Label: "Hero graphic", Kind: KindSelect, Help: "Curated visual library. Each option ships its own CSS, all inspector-pre-vetted. See DesignPlaybook.HeroGraphics for full guidance.", Options: []Option{
				{Value: "", Label: "None"},
				{Value: "mesh", Label: "Mesh gradient (AI/SaaS)"},
				{Value: "pulse", Label: "Radial pulse (consumer/agency)"},
				{Value: "monogram", Label: "Brand monogram (editorial)"},
				{Value: "audit-receipt", Label: "Audit receipt (inspector/trust)"},
				{Value: "gradient-orb", Label: "Gradient orb (drifting soft accent)"},
				{Value: "globe-wire", Label: "Globe + flowing wires (platform/infra)"},
			}},
			{Key: "monogram_char", Label: "Monogram character", Kind: KindText, Help: "Single character used by hero_graphic=monogram. Defaults to first letter of site name."},
			{Key: "audit_score", Label: "Audit score", Kind: KindText, Help: "Numerator shown by hero_graphic=audit-receipt (e.g. '100')."},
			{Key: "audit_score_max", Label: "Audit score max", Kind: KindText, Help: "Denominator shown by hero_graphic=audit-receipt (e.g. '100')."},
			{Key: "audit_baseline", Label: "Audit baseline", Kind: KindText, Help: "Industry-average comparison number shown by hero_graphic=audit-receipt (e.g. '62')."},
			{Key: "audit_label", Label: "Audit label", Kind: KindText, Help: "Caption above the score on hero_graphic=audit-receipt (e.g. 'Live site audit')."},
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
			{Key: "hero_graphic", Label: "Right-column graphic", Kind: KindSelect, Help: "Use one of the curated graphics instead of a photo for image_id. See DesignPlaybook.HeroGraphics.", Options: []Option{
				{Value: "", Label: "None (use image_id)"},
				{Value: "mesh", Label: "Mesh gradient (AI/SaaS)"},
				{Value: "pulse", Label: "Radial pulse (consumer/agency)"},
				{Value: "monogram", Label: "Brand monogram (editorial)"},
				{Value: "audit-receipt", Label: "Audit receipt (inspector/trust)"},
				{Value: "gradient-orb", Label: "Gradient orb (drifting soft accent)"},
				{Value: "globe-wire", Label: "Globe + flowing wires (platform/infra)"},
			}},
			{Key: "monogram_char", Label: "Monogram character", Kind: KindText, Help: "Single character used by hero_graphic=monogram."},
			{Key: "audit_score", Label: "Audit score", Kind: KindText},
			{Key: "audit_score_max", Label: "Audit score max", Kind: KindText},
			{Key: "audit_baseline", Label: "Audit baseline", Kind: KindText},
			{Key: "audit_label", Label: "Audit label", Kind: KindText},
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
		Type: "comparison_table", Label: "Comparison table", Category: "conversion",
		Description: "Feature matrix comparing us against named alternatives. The us column always renders first; competitor columns follow in declared order. Each cell can be a yes/no boolean, a string (price, score, '-'), or empty.",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "us_label", Label: "Our column label", Kind: KindText, Placeholder: "Atomicsite"},
			{Key: "columns", Label: "Competitor columns", Kind: KindArray, ItemSchema: []Field{
				{Key: "", Label: "Name", Kind: KindText, Required: true},
			}},
			{Key: "rows", Label: "Feature rows", Kind: KindArray, ItemSchema: []Field{
				{Key: "feature", Label: "Feature", Kind: KindText, Required: true},
				{Key: "us", Label: "Us cell", Kind: KindText, Help: "Use 'true' / 'false' for tick/cross, or any string like '100/100' or 'Free' for raw text."},
				{Key: "values", Label: "Competitor cells", Kind: KindArray, Help: "Same order as columns. Use 'true' / 'false' for tick/cross, or any string for raw text.", ItemSchema: []Field{
					{Key: "", Label: "Cell", Kind: KindText},
				}},
			}},
		},
	})

	Register(Schema{
		Type: "video_hero", Label: "Video hero", Category: "hero",
		Description: "Hero block with a background video (self-hosted or YouTube/Vimeo no-cookie). Poster image preloads first so LCP stays fast.",
		Fields: []Field{
			{Key: "eyebrow", Label: "Eyebrow", Kind: KindText},
			{Key: "headline", Label: "Headline", Kind: KindText, Required: true},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "video_source", Label: "Video source", Kind: KindSelect, Options: []Option{
				{Value: "self_hosted", Label: "Self-hosted (uploaded video)"},
				{Value: "youtube", Label: "YouTube (no-cookie)"},
				{Value: "vimeo", Label: "Vimeo"},
			}, Help: "youtube uses youtube-nocookie.com so no third-party cookies are set before consent."},
			{Key: "video_id", Label: "Video ID or media ID", Kind: KindText, Required: true, Help: "For self_hosted: paste the uploaded media ID. For YouTube/Vimeo: the public video ID."},
			{Key: "poster_image_id", Label: "Poster image", Kind: KindImageID, Help: "Shown while the video loads. Becomes the LCP element so make it sharp."},
			{Key: "overlay_opacity", Label: "Overlay darkness (0-100)", Kind: KindText, Placeholder: "35", Help: "Dark overlay between video and text. Higher = more legible copy."},
			{Key: "cta_text", Label: "Primary CTA text", Kind: KindText},
			{Key: "cta_url", Label: "Primary CTA URL", Kind: KindURL},
			{Key: "secondary_label", Label: "Secondary link text", Kind: KindText},
			{Key: "secondary_url", Label: "Secondary link URL", Kind: KindURL},
		},
	})

	Register(Schema{
		Type: "testimonial_wall", Label: "Testimonial wall", Category: "social proof",
		Description: "Customer quote wall. Three layouts: 'wall' (masonry grid), 'carousel' (horizontal scroll-snap), 'single_featured' (one large quote).",
		Fields: []Field{
			{Key: "eyebrow", Label: "Eyebrow", Kind: KindText},
			{Key: "heading", Label: "Heading", Kind: KindText, Placeholder: "What teams say after switching"},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "layout", Label: "Layout", Kind: KindSelect, Options: []Option{
				{Value: "wall", Label: "Wall (masonry 3-up)"},
				{Value: "carousel", Label: "Carousel (scroll-snap)"},
				{Value: "single_featured", Label: "Single featured"},
			}},
			{Key: "items", Label: "Testimonials", Kind: KindArray, ItemSchema: []Field{
				{Key: "quote", Label: "Quote", Kind: KindTextarea, Required: true},
				{Key: "author_name", Label: "Author name", Kind: KindText, Required: true},
				{Key: "author_role", Label: "Author role", Kind: KindText},
				{Key: "author_company", Label: "Author company", Kind: KindText},
				{Key: "author_image_id", Label: "Author photo", Kind: KindImageID},
				{Key: "company_logo_id", Label: "Company logo", Kind: KindImageID},
				{Key: "rating", Label: "Rating (1-5)", Kind: KindText, Help: "Optional. Renders as star pips if set."},
				{Key: "source_url", Label: "Source link", Kind: KindURL, Help: "Optional. Links the quote to the original review / case study."},
			}},
			{Key: "footer", Label: "Footer line", Kind: KindText, Help: "Optional small line below the wall (e.g. 'Read 200+ more reviews →')."},
			{Key: "footer_url", Label: "Footer link URL", Kind: KindURL},
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
				// Sprint 5 quick-win (2026-05-24): conditional field
				// visibility. When set, the runtime show/hide script
				// in the form renderer only shows this field when the
				// named other field's value equals the supplied
				// string. Hidden fields are NOT removed from the DOM,
				// just `hidden` attribute toggled, so re-showing is
				// instant + state survives across reveal/hide cycles.
				{Key: "visible_if_field", Label: "Show only when field", Kind: KindText, Help: "Optional. Name of another field on this form whose value gates this one."},
				{Key: "visible_if_equals", Label: "...equals this value", Kind: KindText, Help: "The value the gating field must hold for this one to show. Exact string match. Empty value means 'when the gating field has any non-empty value'."},
				// Sprint 5 quick-win (2026-05-24): multi-step grouping.
				// All fields with step=0 (or missing) render on the
				// initial step; fields with step=1+ render on
				// subsequent steps with Next / Previous buttons
				// gating between them. Set 0 (default) to keep the
				// field on the single-step classic form.
				{Key: "step", Label: "Step (multi-step forms)", Kind: KindNumber, Help: "0 = single-step (default). 1+ groups this field on the Nth step; Next/Previous buttons appear automatically when any field has step >= 1."},
			}},
			{Key: "submit_label", Label: "Submit button label", Kind: KindText, Placeholder: "Submit"},
			{Key: "next_label", Label: "Next button label (multi-step)", Kind: KindText, Placeholder: "Next"},
			{Key: "previous_label", Label: "Previous button label (multi-step)", Kind: KindText, Placeholder: "Back"},
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

	// Sprint 2 slice C (2026-05-22): Public storefront blocks. The 4
	// blocks below complete the e-commerce surface on top of slice B's
	// orders + Mollie checkout backend. product_grid and product_detail
	// resolve product rows at build time (same pattern as collection_list).
	// cart_drawer and checkout_form rely on the storefront island
	// (internal/builder/storefront_island.go) for client-side cart state
	// and the POST to /api/sites/{siteID}/checkout.
	Register(Schema{
		Type: "product_grid", Label: "Product grid", Category: "store",
		Description: "Grid of products from this site's catalog. Resolves active products at build time. Each card has an Add-to-cart button that populates the storefront cart.",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "category_filter", Label: "Category filter", Kind: KindText, Help: "Optional. Only show products whose category matches this exact string."},
			{Key: "limit", Label: "Max products", Kind: KindNumber, Help: "Default 12. Hard cap 48 for LCP."},
			{Key: "columns", Label: "Columns", Kind: KindSelect, Options: []Option{
				{Value: "3", Label: "Three columns (default)"},
				{Value: "2", Label: "Two columns"},
				{Value: "4", Label: "Four columns"},
			}},
			{Key: "sort_by", Label: "Sort by", Kind: KindSelect, Options: []Option{
				{Value: "sort_order", Label: "Manual sort order (default)"},
				{Value: "name", Label: "Name (A-Z)"},
				{Value: "price_asc", Label: "Price (low to high)"},
				{Value: "price_desc", Label: "Price (high to low)"},
			}},
			{Key: "cta_label", Label: "Add-to-cart label", Kind: KindText, Placeholder: "Add to cart"},
			{Key: "link_to_detail", Label: "Link card to product page", Kind: KindBool, Help: "When on, the card title links to /products/{slug}. Pair with a page using the product_detail block."},
		},
	})

	Register(Schema{
		Type: "product_detail", Label: "Product detail", Category: "store",
		Description: "Single product page: gallery, name, description, variant picker, Add to cart. Resolves the product by slug at build time.",
		Fields: []Field{
			{Key: "product_slug", Label: "Product slug", Kind: KindText, Required: true, Help: "The product.slug to render. Find it in Store -> Products."},
			{Key: "show_variants", Label: "Show variant picker", Kind: KindBool, Help: "On by default. Hides when product has a single variant."},
			{Key: "cta_label", Label: "Add-to-cart label", Kind: KindText, Placeholder: "Add to cart"},
			{Key: "gallery_layout", Label: "Gallery layout", Kind: KindSelect, Options: []Option{
				{Value: "side-by-side", Label: "Side by side (default)"},
				{Value: "stacked", Label: "Stacked (image above text)"},
			}},
		},
	})

	Register(Schema{
		Type: "cart_drawer", Label: "Cart drawer", Category: "store",
		Description: "Floating cart trigger + slide-out drawer. Place once per page (typically on every page that has Add-to-cart buttons). Cart state persists in localStorage.",
		Fields: []Field{
			{Key: "trigger_label", Label: "Trigger label", Kind: KindText, Placeholder: "Cart"},
			{Key: "empty_message", Label: "Empty state message", Kind: KindTextarea, Placeholder: "Your cart is empty."},
			{Key: "checkout_url", Label: "Checkout page URL", Kind: KindURL, Required: true, Help: "Path to the page with the checkout_form block, e.g. /checkout."},
			{Key: "currency_locale", Label: "Currency locale", Kind: KindText, Help: "Optional BCP 47 tag for price formatting (e.g. sv-SE, en-US). Falls back to en-US."},
		},
	})

	Register(Schema{
		Type: "search_box", Label: "Search box", Category: "navigation",
		Description: "Pagefind-powered on-site search. Renders a search input + result list. Index is built post-Astro at deploy time when search.pagefind_enabled is on (Settings -> SEO). The block emits a single <div id=\"search\"></div> mount + a same-origin loader script; CSP-safe, no third-party CDN. Place once per page where you want the search affordance (header, dedicated /search page, etc).",
		Fields: []Field{
			{Key: "placeholder", Label: "Input placeholder", Kind: KindText, Placeholder: "Search the site"},
			{Key: "show_empty_state", Label: "Show empty state copy", Kind: KindBool, Help: "Display 'No results' inside the result panel when a query returns nothing. On by default."},
			{Key: "results_max", Label: "Max results shown", Kind: KindNumber, Help: "Default 6. Pagefind ranks all matches; the UI clips at this number."},
		},
	})

	Register(Schema{
		Type: "locale_switcher", Label: "Locale switcher", Category: "navigation",
		Description: "Visitor-facing language picker. Lists every configured locale that has a published counterpart for the current page; the active locale renders as a marker (aria-current=true) instead of a link. Place once globally (in the header or footer) so it appears on every page. Renders as a static link list, zero JS.",
		Fields: []Field{
			{Key: "label", Label: "Strip label", Kind: KindText, Placeholder: "Language"},
			{Key: "style", Label: "Style", Kind: KindSelect, Options: []Option{
				{Value: "inline", Label: "Inline links (default)"},
				{Value: "dropdown", Label: "Native select dropdown"},
				{Value: "list", Label: "Stacked list"},
			}},
			{Key: "show_label", Label: "Show 'Language' label", Kind: KindBool, Help: "On by default. Turn off when placing in a tight nav."},
		},
	})

	Register(Schema{
		Type: "checkout_form", Label: "Checkout form", Category: "store",
		Description: "Customer details form that submits the cart to /api/sites/{siteID}/checkout and redirects to Mollie. Reads cart items from localStorage. Lives on its own page (typically /checkout).",
		Fields: []Field{
			{Key: "heading", Label: "Heading", Kind: KindText, Placeholder: "Checkout"},
			{Key: "subheading", Label: "Subheading", Kind: KindTextarea},
			{Key: "return_url", Label: "Return URL", Kind: KindURL, Required: true, Help: "Where Mollie redirects the visitor after paying, e.g. /thank-you. Receives ?order=ATM-... in the query string."},
			{Key: "submit_label", Label: "Submit button label", Kind: KindText, Placeholder: "Pay now"},
			{Key: "require_phone", Label: "Require phone number", Kind: KindBool},
			{Key: "require_shipping_address", Label: "Require shipping address", Kind: KindBool, Help: "On by default. Turn off for digital-only stores."},
		},
	})
}
