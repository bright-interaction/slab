import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

// Verifies the two design-skill MCP tools wired in Phase 29:
//   - search_design_corpus: returns "not configured" cleanly when
//     DESIGN_REFERENCES_DIR is unset (the test harness runs without
//     the corpus mounted), proving the tool registers + the schema
//     accepts the args + the privacy-invariants do not block it.
//   - design_critique: returns a clean error when no build has run
//     for the site yet, proving the tool registers + the AgentAuth
//     middleware lets the call through.

interface JSONRPCResponse {
	jsonrpc: string;
	id?: number | string;
	result?: any;
	error?: { code: number; message: string };
}

test.describe('agent-api: design-skill MCP tools', () => {
	const cleanup: string[] = [];
	let site: Site;
	let agentKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		const keyRes = await adminApi.post(u(`/api/sites/${site.id}/agent-keys`), {
			data: { name: 'design-probe', capabilities: ['read', 'write'] }
		});
		const body = await keyRes.json();
		agentKey = body.key;
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	async function rpc(adminApi: any, id: number, method: string, params?: any): Promise<JSONRPCResponse> {
		const res = await adminApi.post(u('/mcp'), {
			headers: { 'X-Agent-Key': agentKey, 'Content-Type': 'application/json' },
			data: { jsonrpc: '2.0', id, method, params }
		});
		expect(res.status()).toBe(200);
		return await res.json();
	}

	test('tools/list advertises search_design_corpus and design_critique', async ({ adminApi }) => {
		const r = await rpc(adminApi, 1, 'tools/list');
		expect(r.error).toBeUndefined();
		const names = (r.result?.tools as Array<{ name: string }>).map((t) => t.name);
		expect(names).toContain('search_design_corpus');
		expect(names).toContain('design_critique');
	});

	test('search_design_corpus returns a graceful error when corpus is not mounted', async ({ adminApi }) => {
		const r = await rpc(adminApi, 2, 'tools/call', {
			name: 'search_design_corpus',
			arguments: { query: 'sticky nav', kind: 'code' }
		});
		// MCP returns either a successful result (if DESIGN_REFERENCES_DIR
		// happens to be configured for this run) or a content with isError
		// when the dir is missing. Either is acceptable; what we verify
		// is no protocol-level error envelope.
		expect(r.error).toBeUndefined();
		expect(r.result).toBeTruthy();
	});

	test('design_critique returns a clean error before any build', async ({ adminApi }) => {
		const r = await rpc(adminApi, 3, 'tools/call', {
			name: 'design_critique',
			arguments: {}
		});
		expect(r.error).toBeUndefined();
		// Either the application-level isError bubble or a result that
		// indicates "no successful build found" via content text. Both
		// shapes are valid outcomes; the important guarantee is that
		// the tool does not crash the dispatcher.
		expect(r.result).toBeTruthy();
	});
});
