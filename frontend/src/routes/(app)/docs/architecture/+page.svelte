<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import { ArrowLeft } from 'lucide-svelte';
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
			Architecture
		</h1>
		<p class="text-[13px] text-text-secondary">
			How Atomic Site works, end to end.
		</p>
	</header>

	<div class="mt-8 flex flex-col gap-5">
		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">One-liner</h2>
			<p class="mt-3 text-[14px] text-text-primary">
				AI-agent-driven website builder: structured canvas + BYOAI + SSR Astro output + built-in
				evaluation engine.
			</p>
			<p class="mt-2 text-[13px] text-text-secondary">
				Atomic Site is not another AI website builder competing with Lovable, Bolt, or v0. It is a
				structured canvas that any AI agent can paint on. You connect your own Claude CLI, Cursor,
				or any agent via an open API. The platform provides organisation, guardrails,
				knowledgebase, and a build pipeline that outputs pure SSR or static Astro code.
			</p>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Four differentiators
			</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<p class="text-[13px] font-medium text-text-primary">BYOAI (bring your own AI)</p>
					<p class="mt-1 text-[12px] text-text-secondary">
						Connect any AI agent via API key. No credits, no locked model, no per-token surcharge.
						The same agent you already pay for builds your site.
					</p>
				</div>
				<div>
					<p class="text-[13px] font-medium text-text-primary">SSR / static output</p>
					<p class="mt-1 text-[12px] text-text-secondary">
						Production-grade Astro project. No client-side React SPA on every page load.
						Static-first, hydrated only where needed. The Astro workspace is real code you can
						work in directly.
					</p>
				</div>
				<div>
					<p class="text-[13px] font-medium text-text-primary">Structure-first</p>
					<p class="mt-1 text-[12px] text-text-secondary">
						Global CSS classes, components with typed props, knowledgebase, guardrails. Your
						agent gets a frame to work inside, not a blank canvas it has to invent every time.
					</p>
				</div>
				<div>
					<p class="text-[13px] font-medium text-text-primary">Built-in evaluation</p>
					<p class="mt-1 text-[12px] text-text-secondary">
						130+ checks (security, SEO, GEO, performance, accessibility, privacy) run after every
						build. A+ through F grading. Agents can read the eval and self-correct.
					</p>
				</div>
			</div>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Pipeline</h2>
			<pre
				class="mt-4 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[12px] text-text-primary"
			>{`AI agent --> Go API (guardrails) --> SQLite --> builder --> Astro project --> bun build --> dist/`}</pre>
			<ol class="mt-4 list-decimal space-y-2 pl-5 text-[13px] text-text-secondary">
				<li>
					<span class="text-text-primary">AI agent</span> calls the agent API with an
					<span class="font-mono text-[12px]">X-Agent-Key</span>. It reads context, then writes
					pages, blocks, components, CSS classes, and media.
				</li>
				<li>
					<span class="text-text-primary">Go API</span> (chi + sqlc) validates every write through
					the guardrails engine. Invalid writes are rejected with an explanation the agent can
					act on.
				</li>
				<li>
					<span class="text-text-primary">SQLite</span> stores everything: sites, pages, blocks,
					components, CSS, knowledgebase, guardrails, settings, agent keys.
				</li>
				<li>
					<span class="text-text-primary">Builder</span> reads the DB and writes a complete Astro
					project to disk. Pages, layouts, components, CSS, media, robots.txt, sitemap, llms.txt,
					security.txt, the lot.
				</li>
				<li>
					<span class="text-text-primary">bun build</span> runs Astro's build. Output is pure
					static HTML + CSS + JS in
					<span class="font-mono text-[12px]">dist/</span>.
				</li>
				<li>
					<span class="text-text-primary">Deploy target</span> ships the dist folder. Local bind
					mount, rsync over SSH, or Dockyard route. Built sites land at
					<span class="font-mono text-[12px]">&lt;slug&gt;.slab.example.com</span> by
					default.
				</li>
			</ol>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Production topology
			</h2>
			<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<dt class="text-[12px] text-text-muted">Admin app</dt>
					<dd class="mt-1 font-mono text-[12px] text-text-primary">
						app.slab.example.com
					</dd>
					<p class="mt-1 text-[12px] text-text-secondary">
						Host nginx routes 443 to 127.0.0.1:8091 to the atomicsite container. Deploy via the
						<span class="font-mono">deploy-atomicsite.yml</span> Forgejo workflow.
					</p>
				</div>
				<div>
					<dt class="text-[12px] text-text-muted">Built sites</dt>
					<dd class="mt-1 font-mono text-[12px] text-text-primary">
						&lt;slug&gt;.slab.example.com
					</dd>
					<p class="mt-1 text-[12px] text-text-secondary">
						Caddy on :8080 with a wildcard route serves
						<span class="font-mono">/srv/atomicsite/&lt;slug&gt;/dist</span>. Wildcard DNS A record
						and wildcard TLS cert cover any slug. Local-kind deploy targets land here directly
						because <span class="font-mono">/srv/atomicsite</span> is bind-mounted into the
						atomicsite container.
					</p>
				</div>
			</dl>
			<pre
				class="mt-4 overflow-x-auto rounded-lg border border-border-light bg-bg-elevated p-4 font-mono text-[11px] text-text-primary"
			>{`http://*.slab.example.com {
  root * /srv/atomicsite/{labels.3}/dist
  file_server
}`}</pre>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Site lifecycle
			</h2>
			<ol class="mt-4 list-decimal space-y-2 pl-5 text-[13px] text-text-secondary">
				<li>
					<span class="text-text-primary">Wizard</span> seeds a site from a starter kit (legal
					pages, schema, navigation, sitemap, robots.txt, llms.txt, components, knowledgebase).
				</li>
				<li>
					<span class="text-text-primary">AI agent</span> reads
					<span class="font-mono text-[12px]">/api/agent/context</span> and edits pages, blocks,
					CSS classes, components, media.
				</li>
				<li>
					<span class="text-text-primary">Build</span> compiles the DB into an Astro project and
					runs <span class="font-mono text-[12px]">bun run build</span>.
				</li>
				<li>
					<span class="text-text-primary">Evaluation</span> runs 130+ checks against the built
					output and grades each category A+ through F.
				</li>
				<li>
					<span class="text-text-primary">Deploy target</span> ships the dist folder to its
					destination (local bind mount, rsync, Dockyard route).
				</li>
				<li>
					<span class="text-text-primary">Analytics</span> tails the nginx log to record visits,
					unique sessions, browsers, OS, devices, countries, languages, UTM parameters.
				</li>
				<li>
					<span class="text-text-primary">CRM sync</span> forwards visitor events to BrightCRM via
					HMAC-signed webhook so anonymous and identified journeys stitch together.
				</li>
			</ol>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Starter kits
			</h2>
			<p class="mt-3 text-[13px] text-text-secondary">
				The wizard seeds different defaults depending on what you are building. Each kit ships
				its own legal pages, JSON-LD schema, components, and knowledgebase entries.
			</p>
			<dl class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
				<div>
					<dt class="font-mono text-[12px] text-text-primary">b2b-services</dt>
					<dd class="text-[12px] text-text-muted">
						Professional services, consulting, agencies. Replaces the legacy law-firm kit.
					</dd>
				</div>
				<div>
					<dt class="font-mono text-[12px] text-text-primary">b2c-local</dt>
					<dd class="text-[12px] text-text-muted">
						LocalBusiness schema, hours, reviews, NAP profile.
					</dd>
				</div>
				<div>
					<dt class="font-mono text-[12px] text-text-primary">ecommerce</dt>
					<dd class="text-[12px] text-text-muted">
						Astro + Svelte cart island + Stripe Checkout redirect, Product/Offer microdata.
					</dd>
				</div>
				<div>
					<dt class="font-mono text-[12px] text-text-primary">saas-landing</dt>
					<dd class="text-[12px] text-text-muted">
						Hero, feature grid, pricing, FAQ. Tuned for product launches.
					</dd>
				</div>
				<div>
					<dt class="font-mono text-[12px] text-text-primary">portfolio</dt>
					<dd class="text-[12px] text-text-muted">
						Personal site, case studies, project gallery.
					</dd>
				</div>
				<div>
					<dt class="font-mono text-[12px] text-text-primary">law-firm <span class="text-text-muted">(legacy)</span></dt>
					<dd class="text-[12px] text-text-muted">
						Hidden in the wizard. Existing law-firm sites still work.
					</dd>
				</div>
			</dl>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Site architecture types
			</h2>
			<dl class="mt-4 space-y-3">
				<div>
					<dt class="text-[13px] font-medium text-text-primary">One-pager</dt>
					<dd class="text-[12px] text-text-secondary">
						Single page with anchor sections. No silos. Right for a launch page or a holding
						site.
					</dd>
				</div>
				<div>
					<dt class="text-[13px] font-medium text-text-primary">Soft silo</dt>
					<dd class="text-[12px] text-text-secondary">
						Topic clusters cross-link freely. Right for content sites where breadth matters
						more than topical authority.
					</dd>
				</div>
				<div>
					<dt class="text-[13px] font-medium text-text-primary">Hard silo</dt>
					<dd class="text-[12px] text-text-secondary">
						Topic clusters do not cross-link. Right for SEO-led sites optimising for topical
						authority on a focused set of subjects.
					</dd>
				</div>
			</dl>
			<p class="mt-4 text-[12px] text-text-muted">
				Max 3 URL levels deep, enforced by guardrail. Orphan pages default to noindex. Structure
				auto-generates sitemap, robots.txt, navigation, breadcrumbs.
			</p>
		</Card>
	</div>
</div>
