<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import { ArrowLeft } from 'lucide-svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';

	const siteID = $derived(currentSite.value?.id ?? null);
	const settingsHref = $derived(siteID ? `/sites/${siteID}/settings/analytics` : '/sites');
	const buildHref = $derived(siteID ? `/sites/${siteID}/build` : '/sites');
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
			Personalization
		</h1>
		<p class="text-[13px] text-text-secondary">
			Bidirectional CRM loop. The site sees a returning visitor, asks the CRM, and renders
			content gated on what the CRM knows. "Welcome back Joe, still thinking about
			compliance?"
		</p>
	</header>

	<div class="mt-8 flex flex-col gap-5">
		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">The loop</h2>
			<pre
				class="mt-4 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`1. Visitor lands on built site (e.g. <slug>.atomicsite.example.com)
2. CookieProof banner -> visitor accepts -> POST /t/consent
3. Atomicsite emits identified event -> CRM webhook
4. CRM looks up the visitor by visitor_id, finds history
5. CRM POSTs metadata BACK to /t/inbound (HMAC-SHA256 signed)
6. Hydration <script> in built site GET /t/visitor on next page load
7. Script evaluates [data-asp-when] DSL on every block
8. Matching blocks reveal: "Welcome back Joe, still thinking about compliance?"`}</pre>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Two tiers</h2>
			<dl class="mt-4 space-y-4">
				<div>
					<dt class="text-[13px] font-medium text-text-primary">Anonymous-but-recognised</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Fingerprint matches a prior session, no PII attached. Fields available:
						<code class="font-mono text-[11px]">last_topic</code>,
						<code class="font-mono text-[11px]">last_page</code>,
						<code class="font-mono text-[11px]">lifecycle</code>,
						<code class="font-mono text-[11px]">lead_score</code>,
						<code class="font-mono text-[11px]">returning</code>. Works without identity
						confirmation. "Picking up where you left off, last time you read about Compliance"
						lands here.
					</dd>
				</div>
				<div>
					<dt class="text-[13px] font-medium text-text-primary">Identified</dt>
					<dd class="mt-1 text-[12px] text-text-secondary">
						Visitor previously gave consent and provided email/login. Fields:
						<code class="font-mono text-[11px]">name</code>,
						<code class="font-mono text-[11px]">email</code>,
						<code class="font-mono text-[11px]">company</code>,
						<code class="font-mono text-[11px]">phone</code>,
						<code class="font-mono text-[11px]">job_title</code>. Server-side scrubbed when
						<code class="font-mono text-[11px]">identity_confirmed_at</code> is missing or
						older than the per-site
						<code class="font-mono text-[11px]">analytics.identity_max_age_days</code> window
						(default 30). Closes the "Welcome Joe greets Joe's spouse" risk.
					</dd>
				</div>
			</dl>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Conditional blocks: the DSL
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Any block can carry <code class="font-mono text-[12px]">data.condition</code>. The
				renderer wraps it in
				<code class="font-mono text-[12px]">&lt;div data-asp-when="..." hidden&gt;</code>; the
				hydration script removes
				<code class="font-mono text-[12px]">hidden</code> on match. One condition per block. AND
				is achieved by stacking conditional blocks. No nesting, no parens.
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`condition := key " " op " " value
op        := "==" | "!=" | ">" | ">=" | "<" | "<=" | "in" | "present"
key       := [A-Za-z_][A-Za-z0-9_]*

# Examples:
lead_score >= 60
lifecycle == "SQL"
last_topic == "compliance"
last_topic in compliance,gdpr,nis2
name present
returning == "true"`}</pre>
			<p class="mt-3 text-[12px] text-text-muted">
				The agent is the intended author. Humans see conditions in the block JSON view.
			</p>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Hashed knowledge
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				Keys ending in <code class="font-mono text-[12px]">_hash</code> are matchable in
				conditions but never echoed into rendered content. Right for re-identification
				matching against external services without exposing the underlying value.
			</p>
			<pre
				class="mt-3 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`# Show a special offer when the visitor's email matches a list
# the CRM holds (without ever sending the email back to the browser)
email_hash present`}</pre>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				How to turn it on
			</h2>
			<ol class="mt-4 list-decimal space-y-3 pl-5 text-[13px] text-text-secondary">
				<li>
					Open
					<a href={settingsHref} class="text-text-primary hover:underline">Settings -> Analytics</a>
					on the site you want to personalise.
				</li>
				<li>
					Wire your CRM webhook (URL + secret). The same secret is used for inbound and
					outbound, so the CRM can both receive analytics events AND push metadata back.
				</li>
				<li>
					Toggle <strong>Personalization</strong> on. The hydration script is now injected on
					the next build.
				</li>
				<li>
					Set
					<code class="font-mono text-[11px]">analytics.identity_max_age_days</code> if you
					want a non-default identity-freshness window. 30 days is safe for most B2B sites.
				</li>
				<li>
					Trigger a <a href={buildHref} class="text-text-primary hover:underline">build</a>.
					Visit the live site as a fresh visitor; once a CRM contact is recognised on a return
					visit, conditional blocks light up.
				</li>
			</ol>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Endpoints (reference)
			</h2>
			<dl class="mt-4 space-y-3 text-[12px]">
				<div>
					<dt class="font-mono text-text-primary">POST /t/inbound</dt>
					<dd class="text-text-secondary">
						HMAC-SHA256 signed (X-Atomicsite-Signature). The CRM POSTs metadata for a known
						visitor. Body:
						<code class="font-mono">
							{'{ site_id, visitor_id, fingerprint?, email?, metadata, expires_at?, identity_confirmed_at? }'}
						</code>.
					</dd>
				</div>
				<div>
					<dt class="font-mono text-text-primary">GET /t/visitor?site_id=...</dt>
					<dd class="text-text-secondary">
						Cross-origin friendly. Server re-derives the fingerprint from request headers
						(IP + UA + Accept-Language). Returns
						<code class="font-mono">
							{'{ visitor_id, returning, anonymous: {...}, identified: {...}|null, identity_confirmed_at }'}
						</code>. Identified-tier scrubbed on stale confirmation.
					</dd>
				</div>
				<div>
					<dt class="font-mono text-text-primary">GET /api/agent/visitor-metadata?fingerprint=...</dt>
					<dd class="text-text-secondary">
						Agent-side debug endpoint scoped to the agent's site. Useful for verifying the
						round-trip during development.
					</dd>
				</div>
			</dl>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				CRM-side wiring
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				The reverse direction needs a CRM that can call <code class="font-mono">POST /t/inbound</code>.
				BrightCRM ships with this out of the box (Phase 18.7): set
				<code class="font-mono">ATOMICSITE_INBOUND_URL</code> +
				<code class="font-mono">ATOMICSITE_INBOUND_SECRET</code> on the CRM container. For any
				other CRM, the wire format is symmetric with the outbound path: same HMAC-SHA256 hex
				header, same JSON shape. Roll your own with a workflow / Zap / n8n flow that fires on
				contact updates.
			</p>
		</Card>
	</div>
</div>
