// Maps an evaluation check id (or normalized name) to the in-app settings
// route that lets the user fix the issue. Returns null when no route is known.
//
// Backend check IDs are not stable identifiers; eval/* sources use display
// names like "Strict-Transport-Security" or "Has Meta Description". We
// normalize to a snake_case slug and match prefix patterns.

const PREFIX_TABLE: Array<{ prefixes: string[]; route: (siteID: string) => string }> = [
	{
		// Security headers and policies.
		prefixes: [
			'csp',
			'content_security_policy',
			'hsts',
			'strict_transport_security',
			'x_frame',
			'x_content',
			'referrer',
			'permissions',
			'coop',
			'cross_origin_opener',
			'corp',
			'cross_origin_resource',
			'coep',
			'cross_origin_embedder',
			'security_txt',
			'no_mixed_content'
		],
		route: (s) => `/sites/${s}/settings/security`
	},
	{
		// SEO meta + technical files at the site/page meta level.
		prefixes: [
			'robots',
			'xml_sitemap',
			'sitemap',
			'llms',
			'canonical',
			'hreflang',
			'open_graph',
			'og_',
			'twitter',
			'jsonld',
			'json_ld',
			'schema',
			'language_declared',
			'lang_'
		],
		route: (s) => `/sites/${s}/settings/seo`
	},
	{
		// Allowed scripts / SRI lives on the trust-domain whitelist page.
		prefixes: ['allowed_script', 'sri', 'subresource_integrity'],
		route: (s) => `/sites/${s}/settings/allowed-scripts`
	},
	{
		// Privacy + analytics + AI bots.
		prefixes: [
			'cookie_consent',
			'consent',
			'consent_banner',
			'tracker',
			'tracker_count',
			'pre_consent',
			'ai_bot',
			'ai_training_bots',
			'privacy_policy',
			'terms',
			'cookie_policy'
		],
		route: (s) => `/sites/${s}/settings/analytics`
	},
	{
		// Branding: typography, colors, contrast.
		prefixes: ['color_contrast', 'font_loading', 'font_count', 'font_'],
		route: (s) => `/sites/${s}/branding`
	},
	{
		// Image-specific issues live on the media library.
		prefixes: [
			'alt_text_image',
			'image_format',
			'image_size',
			'lazy_load',
			'lazy_loaded_images',
			'images_have_dimensions',
			'modern_image_formats'
		],
		route: (s) => `/sites/${s}/media`
	},
	{
		// Page-level on-page SEO + accessibility issues are fixed in pages.
		prefixes: [
			'meta_description',
			'has_meta_description',
			'title_length',
			'has_title',
			'h1',
			'has_h1',
			'single_h1',
			'page_has_h1',
			'heading_hierarchy',
			'heading_order',
			'no_empty_headings',
			'alt_text',
			'images_have_alt_text',
			'main_landmark',
			'navigation_landmark',
			'banner_landmark',
			'contentinfo_landmark',
			'page_language',
			'viewport_meta',
			'charset_declared',
			'html5_doctype',
			'not_noindexed',
			'url_length',
			'url_lowercase',
			'url_no_underscores',
			'has_internal_links',
			'form_inputs_have_labels',
			'buttons_have_labels',
			'links_have_labels',
			'no_positive_tabindex',
			'tables_have_headers'
		],
		route: (s) => `/sites/${s}/pages`
	}
];

function normalize(input: string): string {
	return input
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '_')
		.replace(/^_+|_+$/g, '');
}

export function fixRoute(checkID: string, siteID: string): string | null {
	if (!checkID) return null;
	const slug = normalize(checkID);
	for (const entry of PREFIX_TABLE) {
		for (const p of entry.prefixes) {
			if (slug === p || slug.startsWith(p + '_') || slug.startsWith(p)) {
				return entry.route(siteID);
			}
		}
	}
	return null;
}
