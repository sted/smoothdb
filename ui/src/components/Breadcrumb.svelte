<script lang="ts">
	import { router } from "../routes";
	import Icon from "./Icon.svelte";

	interface Crumb {
		title: string;
		path: string;
		// the routes beside this one, this one included: the crumb is a switcher
		siblings: string[];
	}

	// alternatives that are not pages of their own
	const hidden = (seg: string) => seg === "login" || seg.startsWith(":");

	let crumbs: Crumb[] = $derived.by(() => {
		const segments = router.path.split("/").filter(Boolean);
		return segments
			.map((segment, i) => {
				const path = "/" + segments.slice(0, i + 1).join("/");
				const siblings = router.getAltRoutes(path).filter((s) => !hidden(s));
				return { title: decodeURIComponent(segment), path, siblings: siblings.length > 1 ? siblings : [] };
			})
			.slice(1); // the "ui" base
	});

	let openIndex: number | null = $state(null);

	// close the menu on any route change
	$effect(() => {
		router.path;
		openIndex = null;
	});

	function siblingPath(path: string, sibling: string): string {
		return path.slice(0, path.lastIndexOf("/") + 1) + sibling;
	}

	function handleWindowClick(ev: MouseEvent) {
		if (openIndex !== null && !(ev.target as Element).closest?.(".switcher")) openIndex = null;
	}

	function handleKey(ev: KeyboardEvent) {
		if (ev.key === "Escape") openIndex = null;
	}
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleKey} />

<nav aria-label="breadcrumb">
	<ol class="crumbs">
		{#each crumbs as crumb, index}
			<li>
				{#if crumb.siblings.length > 0}
					<div class="switcher">
						<button
							class="crumb"
							class:current={index === crumbs.length - 1}
							aria-haspopup="menu"
							aria-expanded={openIndex === index}
							onclick={() => (openIndex = openIndex === index ? null : index)}
						>
							{crumb.title}
							<Icon name="expand-up-down" size={13} />
						</button>
						{#if openIndex === index}
							<ul class="menu" role="menu">
								{#each crumb.siblings as sibling}
									<li role="none">
										<a
											role="menuitem"
											class:selected={sibling === crumb.title}
											href={siblingPath(crumb.path, sibling)}>{sibling}</a
										>
									</li>
								{/each}
							</ul>
						{/if}
					</div>
				{:else if index < crumbs.length - 1}
					<a class="crumb" href={crumb.path}>{crumb.title}</a>
				{:else}
					<span class="crumb current">{crumb.title}</span>
				{/if}
			</li>
			{#if index < crumbs.length - 1}<li class="sep" aria-hidden="true"><Icon name="arrow-right-s" size={14} /></li>{/if}
		{/each}

	</ol>
</nav>

<style>
	nav {
		height: 100%;
	}
	.crumbs {
		display: flex;
		align-items: center;
		gap: 2px;
		height: 100%;
		padding: 0 12px;
		list-style: none;
	}
	.sep {
		display: inline-flex;
		color: var(--text-3);
	}
	.crumb {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		height: 24px;
		padding: 0 6px;
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--text-2);
		font-size: 12.5px;
		cursor: pointer;
	}
	.crumb :global(.icon) {
		color: var(--text-3);
	}
	a.crumb:hover,
	button.crumb:hover {
		background: var(--bg-hover);
		color: var(--text);
	}
	.crumb.current {
		color: var(--text);
		font-weight: 500;
	}
	span.crumb {
		cursor: default;
	}
	.switcher {
		position: relative;
	}
	.menu {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		z-index: 60;
		min-width: 160px;
		padding: 4px;
		list-style: none;
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--bg);
		box-shadow: var(--shadow-pop);
	}
	.menu a {
		display: flex;
		align-items: center;
		height: 28px;
		padding: 0 8px;
		border-radius: var(--radius-sm);
		color: var(--text-2);
	}
	.menu a:hover {
		background: var(--bg-hover);
		color: var(--text);
	}
	.menu a.selected {
		background: var(--accent-soft);
		color: var(--accent-text);
		font-weight: 500;
	}
</style>
