import type { SiteSetting } from '$lib/api/types';

export interface SettingsMap {
	[category: string]: Record<string, string>;
}

export function settingsToMap(rows: SiteSetting[]): SettingsMap {
	const out: SettingsMap = {};
	for (const r of rows) {
		const cat = out[r.category] ?? {};
		cat[r.key] = r.value;
		out[r.category] = cat;
	}
	return out;
}

export function categoryMap(rows: SiteSetting[]): Record<string, string> {
	const out: Record<string, string> = {};
	for (const r of rows) {
		out[r.key] = r.value;
	}
	return out;
}

function isOn(v: string | undefined): boolean {
	if (!v) return false;
	const s = v.toLowerCase();
	return s === '1' || s === 'true' || s === 'yes' || s === 'on';
}

export interface NginxPreviewInput {
	domain: string;
	security: Record<string, string>;
	server: Record<string, string>;
}

export function buildNginxPreview(input: NginxPreviewInput): string {
	const { domain, security, server } = input;
	const host = domain || 'example.com';

	const hstsEnabled = isOn(security.hsts_enabled);
	const hstsMax = security.hsts_max_age || '31536000';
	const hstsPreload = isOn(security.hsts_preload);
	const cspEnabled = isOn(security.csp_enabled);
	const cspExtra = security.csp_extra_directives || '';
	const xfo = security.x_frame_options || 'SAMEORIGIN';
	const xcto = isOn(security.x_content_type_options);
	const referrer = security.referrer_policy || 'strict-origin-when-cross-origin';
	const permissions = security.permissions_policy || '';
	const httpsRedirect = isOn(security.https_redirect);

	const cacheOn = isOn(server.cache);
	const gzipOn = isOn(server.gzip);
	const brotliOn = isOn(server.brotli);
	const rateLimit = server.rate_limit || '';

	const lines: string[] = [];
	lines.push(`# Generated nginx preview for ${host}`);
	lines.push(`# Apply on next build. Read-only preview.`);
	lines.push('');

	if (httpsRedirect) {
		lines.push(`server {`);
		lines.push(`    listen 80;`);
		lines.push(`    server_name ${host};`);
		lines.push(`    return 301 https://$host$request_uri;`);
		lines.push(`}`);
		lines.push('');
	}

	lines.push(`server {`);
	lines.push(`    listen 443 ssl http2;`);
	lines.push(`    server_name ${host};`);
	lines.push(`    root /var/www/${host}/dist;`);
	lines.push(`    index index.html;`);
	lines.push('');

	if (hstsEnabled) {
		const preload = hstsPreload ? '; preload' : '';
		lines.push(
			`    add_header Strict-Transport-Security "max-age=${hstsMax}; includeSubDomains${preload}" always;`
		);
	}
	if (cspEnabled) {
		const base = `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:;`;
		const full = cspExtra.trim() ? `${base} ${cspExtra.trim()}` : base;
		lines.push(`    add_header Content-Security-Policy "${full}" always;`);
	}
	if (xfo) {
		lines.push(`    add_header X-Frame-Options "${xfo}" always;`);
	}
	if (xcto) {
		lines.push(`    add_header X-Content-Type-Options "nosniff" always;`);
	}
	if (referrer) {
		lines.push(`    add_header Referrer-Policy "${referrer}" always;`);
	}
	if (permissions.trim()) {
		lines.push(`    add_header Permissions-Policy "${permissions.trim()}" always;`);
	}
	lines.push('');

	if (gzipOn) {
		lines.push(`    gzip on;`);
		lines.push(`    gzip_types text/plain text/css application/javascript application/json image/svg+xml;`);
	}
	if (brotliOn) {
		lines.push(`    brotli on;`);
		lines.push(`    brotli_types text/plain text/css application/javascript application/json image/svg+xml;`);
	}
	if (cacheOn) {
		lines.push(`    location ~* \\.(?:css|js|woff2?|ttf|otf|svg|png|jpg|jpeg|webp|avif)$ {`);
		lines.push(`        expires 30d;`);
		lines.push(`        add_header Cache-Control "public, immutable";`);
		lines.push(`    }`);
	}
	if (rateLimit) {
		lines.push(`    limit_req zone=site burst=${rateLimit} nodelay;`);
	}
	lines.push('');
	lines.push(`    location / {`);
	lines.push(`        try_files $uri $uri/ $uri.html =404;`);
	lines.push(`    }`);
	lines.push(`}`);

	return lines.join('\n');
}
