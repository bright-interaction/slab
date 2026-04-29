import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('agent-api: security posture + trusted-domain read access', () => {
	const cleanup: string[] = [];
	let site: Site;
	let agentKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);

		// Mint an agent key for this site so we can hit /api/agent/* with
		// X-Agent-Key. Capabilities default to read so the read-only
		// endpoints are exercised without ambient admin privilege.
		const keyRes = await adminApi.post(u(`/api/sites/${site.id}/agent-keys`), {
			data: { name: 'security-context-probe', capabilities: ['read', 'write'] }
		});
		expect(keyRes.ok()).toBe(true);
		const body = await keyRes.json();
		agentKey = body.key;
		expect(agentKey).toBeTruthy();
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('GET /api/agent/allowed-scripts lists trusted domains (read-only)', async ({
		adminApi
	}) => {
		// Seed two rows with different kinds via the admin API.
		await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain: 'js.stripe.com', purpose: 'payments', kind: 'script' }
		});
		await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain: 'cal.com', purpose: 'booking iframe', kind: 'frame' }
		});

		const res = await adminApi.get(u('/api/agent/allowed-scripts'), {
			headers: { 'X-Agent-Key': agentKey }
		});
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(Array.isArray(body.trusted_domains)).toBe(true);
		expect(body.allowed_kinds).toEqual([
			'script',
			'frame',
			'image',
			'media',
			'connect',
			'all'
		]);
		expect(body.writable_by_agent).toBe(false);
		const domains = body.trusted_domains.map((r: { domain: string }) => r.domain);
		expect(domains).toContain('js.stripe.com');
		expect(domains).toContain('cal.com');
	});

	test('GET /api/agent/security/preview returns the resolved CSP including frame-src', async ({
		adminApi
	}) => {
		const res = await adminApi.get(u('/api/agent/security/preview'), {
			headers: { 'X-Agent-Key': agentKey }
		});
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.csp).toContain('frame-src');
		expect(body.csp).toContain('cal.com');
		expect(body.csp).toContain('upgrade-insecure-requests');
		expect(body.hsts).toContain('max-age=');
		expect(body.cross_origin_opener_policy).toBe('same-origin');
	});

	test('agent context surfaces security_posture + new endpoint groups', async ({
		adminApi
	}) => {
		const res = await adminApi.get(u('/api/agent/context'), {
			headers: { 'X-Agent-Key': agentKey }
		});
		expect(res.ok()).toBe(true);
		const body = await res.json();

		// security_posture: trusted domains + allowed_kinds + generated files
		const sp = body.security_posture;
		expect(sp).toBeTruthy();
		expect(sp.allowed_kinds).toEqual([
			'script',
			'frame',
			'image',
			'media',
			'connect',
			'all'
		]);
		expect(sp.trusted_domains.frame).toContain('cal.com');
		expect(sp.trusted_domains.script).toContain('js.stripe.com');
		expect(sp.frame_ancestors).toBe("'none'");
		expect(sp.human_admin_url).toContain('/settings/allowed-scripts');
		const generatedPaths = sp.generated_files.map((g: { path: string }) => g.path);
		expect(generatedPaths).toContain('/sitemap-index.xml');
		expect(generatedPaths).toContain('/robots.txt');
		expect(generatedPaths).toContain('/.well-known/security.txt');
		expect(generatedPaths).toContain('/llms.txt');

		// Endpoints catalogue gains security + trusted_domains groups
		expect(Array.isArray(body.endpoints.security)).toBe(true);
		expect(Array.isArray(body.endpoints.trusted_domains)).toBe(true);
		const secPaths = body.endpoints.security.map((e: { path: string }) => e.path);
		expect(secPaths).toContain('/api/agent/security/preview');
		const trustPaths = body.endpoints.trusted_domains.map(
			(e: { path: string }) => e.path
		);
		expect(trustPaths).toContain('/api/agent/allowed-scripts');

		// Editing modes mention the trusted-domains + security UI
		const modeNames = body.editing_modes.map((m: { name: string }) => m.name);
		expect(modeNames).toContain('Trusted external domains');
		expect(modeNames).toContain('Security headers');
	});
});
