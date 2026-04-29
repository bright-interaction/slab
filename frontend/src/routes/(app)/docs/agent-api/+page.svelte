<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import { ArrowLeft } from 'lucide-svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';

	const siteID = $derived(currentSite.value?.id ?? null);

	type Method = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
	type Endpoint = { method: Method; path: string; write?: boolean; purpose: string };
	type Group = { title: string; endpoints: Endpoint[] };

	const groups: Group[] = [
		{
			title: 'Context',
			endpoints: [
				{
					method: 'GET',
					path: '/api/agent/context',
					purpose: 'Full site structure, knowledgebase, components, CSS classes, guardrails, pending_setup.'
				},
				{
					method: 'GET',
					path: '/api/agent/bootstrap',
					purpose: 'Re-fetch the personalised CLAUDE.md (text/markdown) for this site.'
				}
			]
		},
		{
			title: 'Pages',
			endpoints: [
				{ method: 'POST', path: '/api/agent/pages', write: true, purpose: 'Create a page.' },
				{
					method: 'PATCH',
					path: '/api/agent/pages/{slug}',
					write: true,
					purpose: 'Update page by slug.'
				},
				{
					method: 'DELETE',
					path: '/api/agent/pages/{slug}',
					write: true,
					purpose: 'Delete page by slug.'
				}
			]
		},
		{
			title: 'Blocks',
			endpoints: [
				{
					method: 'POST',
					path: '/api/agent/blocks?page={slug}',
					write: true,
					purpose: 'Create a block on a page.'
				},
				{
					method: 'POST',
					path: '/api/agent/pages/{slug}/blocks',
					write: true,
					purpose: 'Create a block (slug-in-path variant).'
				},
				{
					method: 'PATCH',
					path: '/api/agent/blocks/{blockID}',
					write: true,
					purpose: 'Update a block.'
				},
				{
					method: 'DELETE',
					path: '/api/agent/blocks/{blockID}',
					write: true,
					purpose: 'Delete a block.'
				}
			]
		},
		{
			title: 'Components',
			endpoints: [
				{
					method: 'POST',
					path: '/api/agent/components',
					write: true,
					purpose: 'Register a reusable component.'
				},
				{
					method: 'PATCH',
					path: '/api/agent/components/{name}',
					write: true,
					purpose: 'Update a component by name.'
				}
			]
		},
		{
			title: 'CSS classes',
			endpoints: [
				{
					method: 'POST',
					path: '/api/agent/css-classes',
					write: true,
					purpose: 'Add a global CSS class.'
				},
				{
					method: 'PATCH',
					path: '/api/agent/css-classes/{name}',
					write: true,
					purpose: 'Update a CSS class by name.'
				}
			]
		},
		{
			title: 'Global blocks',
			endpoints: [
				{
					method: 'PUT',
					path: '/api/agent/global/{slot}',
					write: true,
					purpose: 'Create or update a global block (header, footer, etc).'
				}
			]
		},
		{
			title: 'Branding',
			endpoints: [
				{
					method: 'GET',
					path: '/api/agent/branding',
					purpose: 'Read site branding (colours, fonts, design references).'
				},
				{
					method: 'PATCH',
					path: '/api/agent/branding',
					write: true,
					purpose: 'Update branding.'
				}
			]
		},
		{
			title: 'Profile',
			endpoints: [
				{
					method: 'GET',
					path: '/api/agent/profile',
					purpose: 'Read site profile (business name, address, contact emails).'
				},
				{
					method: 'PATCH',
					path: '/api/agent/profile',
					write: true,
					purpose: 'Patch profile fields (drives Organization JSON-LD, security.txt, legal pages).'
				}
			]
		},
		{
			title: 'Settings',
			endpoints: [
				{
					method: 'GET',
					path: '/api/agent/settings',
					purpose: 'List all settings rows for the site.'
				},
				{
					method: 'GET',
					path: '/api/agent/settings/{category}',
					purpose: 'List settings for one category (seo, analytics, general, etc.).'
				},
				{
					method: 'PATCH',
					path: '/api/agent/settings',
					write: true,
					purpose: 'Bulk upsert. Writes to seo / analytics / general; other categories admin-only.'
				}
			]
		},
		{
			title: 'Fonts',
			endpoints: [
				{ method: 'GET', path: '/api/agent/fonts', purpose: 'List self-hosted woff2 fonts.' },
				{
					method: 'POST',
					path: '/api/agent/fonts',
					write: true,
					purpose: 'Upload a woff2 font (multipart, magic-header validated, 2 MB cap).'
				},
				{
					method: 'DELETE',
					path: '/api/agent/fonts/{fontID}',
					write: true,
					purpose: 'Delete a font.'
				}
			]
		},
		{
			title: 'Design references',
			endpoints: [
				{
					method: 'GET',
					path: '/api/agent/design-references',
					purpose: 'List GitHub design references.'
				},
				{
					method: 'POST',
					path: '/api/agent/design-references',
					write: true,
					purpose: 'Add a public GitHub URL as a design reference.'
				},
				{
					method: 'DELETE',
					path: '/api/agent/design-references/{refID}',
					write: true,
					purpose: 'Remove a design reference.'
				}
			]
		},
		{
			title: 'Media',
			endpoints: [
				{ method: 'GET', path: '/api/agent/media', purpose: 'List media for the site.' },
				{
					method: 'POST',
					path: '/api/agent/media',
					write: true,
					purpose: 'Upload media (multipart).'
				},
				{
					method: 'POST',
					path: '/api/agent/media/from-url',
					write: true,
					purpose: 'Ingest from URL (SSRF-guarded).'
				},
				{
					method: 'POST',
					path: '/api/agent/media/from-base64',
					write: true,
					purpose: 'Ingest base64-encoded image.'
				},
				{
					method: 'PATCH',
					path: '/api/agent/media/{mediaID}',
					write: true,
					purpose: 'Update alt text or folder.'
				},
				{
					method: 'DELETE',
					path: '/api/agent/media/{mediaID}',
					write: true,
					purpose: 'Delete media.'
				}
			]
		},
		{
			title: 'Media folders',
			endpoints: [
				{
					method: 'GET',
					path: '/api/agent/media/folders',
					purpose: 'List folders with item counts.'
				},
				{
					method: 'POST',
					path: '/api/agent/media/folders',
					write: true,
					purpose: 'Create a named folder.'
				},
				{
					method: 'DELETE',
					path: '/api/agent/media/folders/{folderName}',
					write: true,
					purpose: 'Delete a folder; items move to unfiled.'
				}
			]
		},
		{
			title: 'Build',
			endpoints: [
				{ method: 'POST', path: '/api/agent/build', write: true, purpose: 'Trigger a build.' },
				{
					method: 'GET',
					path: '/api/agent/build/{buildID}/status',
					purpose: 'Poll build status.'
				}
			]
		},
		{
			title: 'Evaluation',
			endpoints: [
				{
					method: 'GET',
					path: '/api/agent/evaluation/{buildID}',
					purpose: 'Fetch eval results for a build.'
				}
			]
		}
	];

	const methodBadgeClass: Record<Method, string> = {
		GET: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
		POST: 'bg-sky-500/10 text-sky-600 dark:text-sky-400',
		PATCH: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
		PUT: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
		DELETE: 'bg-red-500/10 text-red-600 dark:text-red-400'
	};
</script>

<div class="mx-auto max-w-5xl px-6 py-8">
	<a
		href="/docs"
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Documentation
	</a>
	<header class="mt-3 flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Agent API
		</h1>
		<p class="text-[13px] text-text-secondary">
			Full endpoint catalogue. Use these from any AI agent, CI job, or HTTP client.
		</p>
	</header>

	<div class="mt-8 flex flex-col gap-5">
		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Authentication
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Every request must carry an agent key. Either header works:
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`X-Agent-Key: atomic_<hex>
Authorization: Bearer atomic_<hex>`}</pre>
			<p class="mt-3 text-[13px] text-text-secondary">
				Smoke test:
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`curl -H "X-Agent-Key: $ATOMICSITE_KEY" \\
  https://app.slab.example.com/api/agent/context | jq .`}</pre>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Get a key</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Generate, view capabilities, and revoke keys per site. The raw key is shown once at
				creation; only its SHA-256 hash is stored.
			</p>
			<a
				href={siteID ? `/sites/${siteID}/agent-keys` : '/sites'}
				class="mt-4 inline-flex items-center gap-1.5 text-[12px] text-text-primary transition-colors hover:text-text-secondary"
			>
				Open Agent keys for this site
			</a>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Capability model
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Keys carry a JSON capabilities array. Today the system checks two values:
			</p>
			<ul class="mt-3 space-y-2 text-[13px] text-text-secondary">
				<li>
					<span class="font-mono text-[12px] text-text-primary">read</span>: required for every
					endpoint.
				</li>
				<li>
					<span class="font-mono text-[12px] text-text-primary">write</span>: required for every
					state-changing endpoint (marked with the
					<span class="ml-1 inline-flex items-center rounded border border-border-light px-1.5 py-0.5 font-mono text-[10px] text-text-muted">write</span>
					badge in the catalogue below).
				</li>
			</ul>
			<p class="mt-3 text-[12px] text-text-muted">
				New keys default to <span class="font-mono">["read", "write"]</span>. Strip "write" to
				create a read-only key for monitoring or context bots.
			</p>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Site scoping
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Every agent key is bound to one site. The site ID is read from the agent identity, not
				the URL. A key for site A cannot manipulate site B even if the path says otherwise.
			</p>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Pending setup
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Every <span class="font-mono text-[12px]">/api/agent/context</span> response carries a
				<span class="font-mono text-[12px]">pending_setup</span> array. Each entry is a
				configuration gap the agent should resolve before declaring the site done. Walk through
				it on the first call.
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`"pending_setup": [
  {
    "id": "profile.business_name",
    "category": "profile",
    "title": "Add the business name",
    "why": "Drives Organization JSON-LD schema, legal pages, and security.txt.",
    "action": "Ask the user for the legal business name, then PATCH it onto the profile.",
    "endpoint": "PATCH /api/agent/profile",
    "severity": "required"
  },
  {
    "id": "analytics.consent",
    "category": "analytics",
    "title": "Add a consent banner",
    "why": "Tracking is on but no banner is in place. EU GDPR requires consent.",
    "action": "Flip on CookieProof or paste a banner snippet (Cookiebot, OneTrust, Termly, etc.).",
    "endpoint": "PATCH /api/agent/settings",
    "severity": "required"
  }
]`}</pre>
			<p class="mt-3 text-[12px] text-text-muted">
				Severity is <span class="font-mono">required</span> (blocks an A+ build) or
				<span class="font-mono">recommended</span> (improves quality, build still passes).
				The list shrinks as the agent applies fixes; an empty array means the site is fully
				configured.
			</p>
		</Card>

		{#each groups as group (group.title)}
			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					{group.title}
				</h2>
				<div class="mt-4 overflow-x-auto">
					<table class="w-full text-left text-[12px]">
						<thead>
							<tr class="border-b border-border-light text-text-muted">
								<th class="w-20 pb-2 pr-3 font-medium">Method</th>
								<th class="pb-2 pr-3 font-medium">Path</th>
								<th class="pb-2 pr-3 font-medium">Caps</th>
								<th class="pb-2 font-medium">Purpose</th>
							</tr>
						</thead>
						<tbody>
							{#each group.endpoints as e (e.method + e.path)}
								<tr class="border-b border-border-light/40 last:border-b-0">
									<td class="py-2 pr-3 align-top">
										<span
											class="inline-flex items-center justify-center rounded px-1.5 py-0.5 font-mono text-[10px] font-medium {methodBadgeClass[
												e.method
											]}"
										>
											{e.method}
										</span>
									</td>
									<td class="py-2 pr-3 align-top">
										<code class="font-mono text-[12px] text-text-primary">{e.path}</code>
									</td>
									<td class="py-2 pr-3 align-top">
										{#if e.write}
											<span
												class="inline-flex items-center rounded border border-border-light px-1.5 py-0.5 font-mono text-[10px] text-text-muted"
												>write</span
											>
										{:else}
											<span
												class="inline-flex items-center rounded border border-border-light px-1.5 py-0.5 font-mono text-[10px] text-text-muted"
												>read</span
											>
										{/if}
									</td>
									<td class="py-2 align-top text-text-secondary">{e.purpose}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</Card>
		{/each}

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Sample flow
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				The standard build-and-evaluate loop. Run from any shell with bash + curl + jq.
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`# 1. Bootstrap context
curl -sH "X-Agent-Key: $ATOMICSITE_KEY" "$ATOMICSITE_API/api/agent/context" | jq .

# 2. Create a page
curl -sH "X-Agent-Key: $ATOMICSITE_KEY" -H "Content-Type: application/json" \\
  -d '{"slug":"/about","title":"About us: what we do and why","status":"draft"}' \\
  "$ATOMICSITE_API/api/agent/pages"

# 3. Add a hero block
curl -sH "X-Agent-Key: $ATOMICSITE_KEY" -H "Content-Type: application/json" \\
  -d '{"type":"hero","data":{"heading":"Welcome","cta_label":"Get started"}}' \\
  "$ATOMICSITE_API/api/agent/pages/about/blocks"

# 4. Trigger a build
BUILD=$(curl -sH "X-Agent-Key: $ATOMICSITE_KEY" -X POST "$ATOMICSITE_API/api/agent/build" | jq -r .build_id)

# 5. Poll status
until [ "$(curl -sH "X-Agent-Key: $ATOMICSITE_KEY" "$ATOMICSITE_API/api/agent/build/$BUILD/status" | jq -r .status)" = "done" ]; do sleep 2; done

# 6. Read evaluation
curl -sH "X-Agent-Key: $ATOMICSITE_KEY" "$ATOMICSITE_API/api/agent/evaluation/$BUILD" | jq .

# 7. Fix any failing checks, repeat from step 2.`}</pre>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Error format
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				HTTP status + JSON body. Errors are the agent's signal to self-correct.
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`{
  "error": "title must be 30-60 chars (got 12)",
  "code": "validation_failed",
  "details": { "field": "title", "min": 30, "max": 60 }
}`}</pre>
			<dl class="mt-4 grid grid-cols-1 gap-2 text-[12px] sm:grid-cols-2">
				<div>
					<dt class="font-mono text-text-primary">400</dt>
					<dd class="text-text-secondary">Malformed JSON, missing required field.</dd>
				</div>
				<div>
					<dt class="font-mono text-text-primary">401</dt>
					<dd class="text-text-secondary">Missing or invalid agent key.</dd>
				</div>
				<div>
					<dt class="font-mono text-text-primary">403</dt>
					<dd class="text-text-secondary">Capability missing (often "write").</dd>
				</div>
				<div>
					<dt class="font-mono text-text-primary">404</dt>
					<dd class="text-text-secondary">Page, block, or media not found.</dd>
				</div>
				<div>
					<dt class="font-mono text-text-primary">409</dt>
					<dd class="text-text-secondary">Slug already taken.</dd>
				</div>
				<div>
					<dt class="font-mono text-text-primary">422</dt>
					<dd class="text-text-secondary">Guardrail rejection. Read details, fix, retry.</dd>
				</div>
			</dl>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Limits and safety
			</h2>
			<ul class="mt-3 list-disc space-y-2 pl-5 text-[13px] text-text-secondary">
				<li>Media upload: per-type size cap, MIME sniffed server-side, SVG rejected.</li>
				<li>woff2 font: 2 MB per file, magic header (wOF2 at offset 0) validated.</li>
				<li>
					GitHub design references: 32 KB per file, 200 KB total per import. Public repos only.
				</li>
				<li>
					from-url media: SSRF-guarded. Rejects private IPs, loopback, link-local (169.254.x), and
					file:// URLs. Pre-resolves DNS.
				</li>
				<li>Concurrent builds per site: serialised. A second trigger waits.</li>
			</ul>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Source of truth
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				If you find a discrepancy between this page and behaviour,
				<span class="font-mono text-[12px]">internal/server/server.go</span> is canonical. Capability
				checks live in <span class="font-mono text-[12px]">internal/middleware/agent_auth.go</span>;
				guardrail rules live in
				<span class="font-mono text-[12px]">internal/agent/guardrails.go</span>.
			</p>
		</Card>
	</div>
</div>
