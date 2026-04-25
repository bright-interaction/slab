<script lang="ts">
	import { goto } from '$app/navigation';
	import { page as pageState } from '$app/state';
	import * as componentsApi from '$lib/api/components';
	import { ApiError } from '$lib/api/client';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Tabs from '$lib/components/ui/Tabs.svelte';
	import { Tabs as BitsTabs } from 'bits-ui';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { ArrowLeft, X } from 'lucide-svelte';
	import type { Component } from '$lib/api/types';

	const siteID = $derived(pageState.params.siteID as string);
	const componentID = $derived(pageState.params.componentID as string);

	let component = $state<Component | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let name = $state('');
	let category = $state('section');
	let template = $state('');
	let propsSchema = $state('');
	let cssClassesList = $state<string[]>([]);
	let cssClassDraft = $state('');
	let usageNote = $state('');

	let saving = $state(false);
	let deleting = $state(false);
	let activeTab = $state('template');

	const categoryOptions = [
		{ value: 'section', label: 'Section' },
		{ value: 'layout', label: 'Layout' },
		{ value: 'ui', label: 'UI' },
		{ value: 'navigation', label: 'Navigation' },
		{ value: 'form', label: 'Form' },
		{ value: 'custom', label: 'Custom' }
	];

	const tabs = [
		{ id: 'template', label: 'Template' },
		{ id: 'props', label: 'Props schema' },
		{ id: 'css', label: 'CSS classes used' },
		{ id: 'notes', label: 'Usage notes' }
	];

	function parseCssClasses(raw: string): string[] {
		if (!raw || raw.trim() === '') return [];
		try {
			const v = JSON.parse(raw);
			if (Array.isArray(v)) return v.filter((x): x is string => typeof x === 'string');
			return [];
		} catch {
			return [];
		}
	}

	function formatPropsSchema(raw: string): string {
		if (!raw || raw.trim() === '') return '';
		try {
			return JSON.stringify(JSON.parse(raw), null, 2);
		} catch {
			return raw;
		}
	}

	function hydrate(c: Component) {
		name = c.name;
		category = c.category || 'section';
		template = c.template || '';
		propsSchema = formatPropsSchema(c.props_schema || '');
		cssClassesList = parseCssClasses(c.css_classes || '');
		usageNote = c.usage_note || '';
	}

	async function load() {
		loading = true;
		loadError = null;
		try {
			const c = await componentsApi.get(siteID, componentID);
			component = c;
			hydrate(c);
		} catch (err) {
			loadError = err instanceof ApiError ? err.message : 'Failed to load component';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const dirty = $derived.by(() => {
		if (!component) return false;
		const initialClasses = parseCssClasses(component.css_classes || '');
		const sameClasses =
			initialClasses.length === cssClassesList.length &&
			initialClasses.every((v, i) => v === cssClassesList[i]);
		return (
			name !== component.name ||
			category !== (component.category || 'section') ||
			template !== (component.template || '') ||
			propsSchema !== formatPropsSchema(component.props_schema || '') ||
			usageNote !== (component.usage_note || '') ||
			!sameClasses
		);
	});

	const propsSchemaError = $derived.by(() => {
		const v = propsSchema.trim();
		if (v === '') return null;
		try {
			JSON.parse(v);
			return null;
		} catch (e) {
			return e instanceof Error ? e.message : 'Invalid JSON';
		}
	});

	function addCssClass() {
		const v = cssClassDraft.trim();
		if (!v) return;
		if (cssClassesList.includes(v)) {
			cssClassDraft = '';
			return;
		}
		cssClassesList = [...cssClassesList, v];
		cssClassDraft = '';
	}

	function removeCssClass(value: string) {
		cssClassesList = cssClassesList.filter((c) => c !== value);
	}

	function handleCssKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ',') {
			e.preventDefault();
			addCssClass();
		}
	}

	async function save() {
		if (saving || !dirty || !component) return;
		if (propsSchemaError) {
			toast.error('Props schema is not valid JSON.');
			return;
		}
		saving = true;
		try {
			const parsedSchema =
				propsSchema.trim() === '' ? undefined : (JSON.parse(propsSchema) as unknown);
			const updated = await componentsApi.update(siteID, componentID, {
				name,
				category,
				template,
				props_schema: parsedSchema,
				css_classes: cssClassesList,
				usage_note: usageNote
			});
			component = updated;
			hydrate(updated);
			toast.success('Component saved.');
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save component';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}

	function discard() {
		if (component) hydrate(component);
	}

	async function deleteComponent() {
		if (!component) return;
		const ok = await confirm({
			title: `Delete "${component.name}"?`,
			message: 'This removes the component permanently. This cannot be undone.',
			confirmText: 'Delete',
			cancelText: 'Cancel',
			variant: 'danger'
		});
		if (!ok) return;
		deleting = true;
		try {
			await componentsApi.remove(siteID, componentID);
			toast.success('Component deleted.');
			await goto(`/sites/${siteID}/components`);
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to delete component';
			toast.error(msg);
			deleting = false;
		}
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<div class="mb-4">
		<a
			href={`/sites/${siteID}/components`}
			class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
		>
			<ArrowLeft class="h-3.5 w-3.5" />
			All components
		</a>
	</div>

	{#if loading}
		<div class="space-y-4">
			<Skeleton width="40%" height="2rem" />
			<Skeleton width="20%" height="0.9rem" />
			<Card padding="md">
				<Skeleton width="100%" height="6rem" />
			</Card>
		</div>
	{:else if loadError}
		<Card padding="md">
			<p class="text-[13px] text-danger">{loadError}</p>
		</Card>
	{:else if component}
		<header class="flex items-start justify-between gap-4">
			<div class="flex min-w-0 flex-1 flex-col gap-2">
				<input
					class="w-full bg-transparent font-display text-3xl font-extralight tracking-tight text-text-primary placeholder:text-text-muted focus:outline-none"
					bind:value={name}
					placeholder="Component name"
				/>
				<div class="flex items-center gap-3">
					<div class="w-48">
						<Select options={categoryOptions} bind:value={category} />
					</div>
					<span class="text-[11px] text-text-muted">id: <span class="font-mono">{component.id}</span></span>
				</div>
			</div>
			<div class="flex shrink-0 items-center gap-2">
				<Button variant="danger" onclick={deleteComponent} loading={deleting}>Delete</Button>
			</div>
		</header>

		<section class="mt-8">
			<Tabs {tabs} bind:value={activeTab}>
				<BitsTabs.Content value="template" class="pt-4">
					<Textarea
						class="font-mono text-[13px]"
						rows={16}
						value={template}
						oninput={(e) => {
							template = (e.currentTarget as HTMLTextAreaElement).value;
						}}
						placeholder={'---\nconst { headline } = Astro.props;\n---\n<section>...</section>'}
						hint="Astro source for this component."
					/>
				</BitsTabs.Content>

				<BitsTabs.Content value="props" class="pt-4">
					<Textarea
						class="font-mono text-[13px]"
						rows={14}
						value={propsSchema}
						oninput={(e) => {
							propsSchema = (e.currentTarget as HTMLTextAreaElement).value;
						}}
						placeholder={'{\n  "type": "object",\n  "properties": {\n    "headline": { "type": "string" }\n  }\n}'}
						error={propsSchemaError ?? undefined}
						hint={propsSchemaError ? undefined : 'JSON Schema. Leave empty to skip validation.'}
					/>
				</BitsTabs.Content>

				<BitsTabs.Content value="css" class="pt-4">
					<div class="flex flex-col gap-3">
						<div class="flex flex-wrap items-center gap-1.5 rounded-lg border border-border bg-bg-elevated px-2 py-2">
							{#each cssClassesList as cls (cls)}
								<span
									class="inline-flex items-center gap-1 rounded-md border border-border-light bg-bg-surface px-2 py-1 font-mono text-[12px] text-text-primary"
								>
									{cls}
									<button
										type="button"
										class="inline-flex h-4 w-4 items-center justify-center rounded text-text-muted hover:text-danger"
										aria-label={`Remove ${cls}`}
										onclick={() => removeCssClass(cls)}
									>
										<X class="h-3 w-3" />
									</button>
								</span>
							{/each}
							<input
								class="min-w-[8rem] flex-1 bg-transparent font-mono text-[13px] text-text-primary placeholder:text-text-muted focus:outline-none"
								placeholder={cssClassesList.length === 0 ? 'btn-primary, hero-grid' : ''}
								value={cssClassDraft}
								oninput={(e) => {
									cssClassDraft = (e.currentTarget as HTMLInputElement).value;
								}}
								onkeydown={handleCssKeydown}
								onblur={addCssClass}
							/>
						</div>
						<p class="text-[12px] text-text-muted">
							Press Enter or comma to add. CSS class names this component depends on.
						</p>
					</div>
				</BitsTabs.Content>

				<BitsTabs.Content value="notes" class="pt-4">
					<Textarea
						rows={6}
						value={usageNote}
						oninput={(e) => {
							usageNote = (e.currentTarget as HTMLTextAreaElement).value;
						}}
						placeholder="When and how to use this component."
					/>
				</BitsTabs.Content>
			</Tabs>
		</section>

		<div
			class="sticky bottom-0 mt-6 flex items-center justify-between gap-3 border-t border-border-light bg-bg-surface/85 py-3 backdrop-blur"
		>
			<span class="text-[12px] text-text-muted">
				{#if propsSchemaError}
					Props schema JSON is invalid.
				{:else if dirty}
					Unsaved changes.
				{:else}
					Up to date.
				{/if}
			</span>
			<div class="flex items-center gap-2">
				<Button variant="ghost" onclick={discard} disabled={!dirty || saving}>Discard</Button>
				<Button
					variant="primary"
					onclick={save}
					loading={saving}
					disabled={!dirty || propsSchemaError !== null}
				>
					Save
				</Button>
			</div>
		</div>
	{/if}
</div>
