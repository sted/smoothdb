<script lang="ts">
	import { router } from "../routes";
	import type { MouseEventHandler } from "svelte/elements";
	import RiExpandUpDownLine from "/assets/images/expand-up-down-line.svg";
	import RiAddLine from "/assets/images/add-line.svg";

	interface Props {
		rowAdd: MouseEventHandler<HTMLButtonElement>;
	}

	let { rowAdd }: Props = $props();

	interface Breadcrumb {
		title: string;
		path: string;
		hasMultipleChoices: boolean;
	}

	let breadcrumbs: Breadcrumb[] = $state([]);
	let activeDropdownIndex: number | null = $state(null);
	let dropdownRoutes: string[] = $state([]);

	router.subscribe(() => {
		const segments = window.location.pathname.split("/").filter(Boolean);
		breadcrumbs = segments.map((segment, index) => {
			const path = "/" + segments.slice(0, index + 1).join("/");
			const nextRoutes = router.getAltRoutes(path);
			return {
				title: segment,
				path,
				hasMultipleChoices: nextRoutes.length > 1,
			};
		});
		activeDropdownIndex = null;
		breadcrumbs.shift();
	});

	function handleSegmentClick(event: MouseEvent, path: string): void {
		event.preventDefault();
		router.navigate(path);
		closeDropdown();
	}

	function handleDropdownClick(event: MouseEvent, path: string, index: number): void {
		event.stopPropagation();

		const nextRoutes = router.getAltRoutes(path);
		if (nextRoutes.length > 1) {
			activeDropdownIndex = index === activeDropdownIndex ? null : index;
			dropdownRoutes = nextRoutes;
		}
	}

	function closeDropdown(): void {
		activeDropdownIndex = null;
	}

	function handleClickOutside(event: MouseEvent) {
		if (!(event.target instanceof Element)) return;
		if (
			activeDropdownIndex !== null &&
			!event.target.closest(".dropdown") &&
			!event.target.closest(".breadcrumb-item")
		) {
			closeDropdown();
		}
	}

	function navigateFromDropdown(path: string, route: string): void {
		const segments = path.split("/").filter(Boolean);
		const newPath = "/" + segments.slice(0, -1).join("/") + "/" + route;
		router.navigate(newPath);
		closeDropdown();
	}
</script>

<svelte:window on:click={handleClickOutside} />

<nav aria-label="breadcrumb">
	<ol>
		{#each breadcrumbs as { title, path, hasMultipleChoices }, index}
			<li class="breadcrumb-item">
				<!-- svelte-ignore a11y_click_events_have_key_events -->
				<span
					role="button"
					tabindex="0"
					onclick={(event) => handleSegmentClick(event, path)}>{title}</span
				>
				{#if hasMultipleChoices}
					<button
						tabindex="0"
						onclick={(event) => handleDropdownClick(event, path, index)}
					>
						<img class="remixicon" src={RiExpandUpDownLine} />
					</button>
				{/if}
				{#if index === activeDropdownIndex}
					<ol class="dropdown">
						{#each dropdownRoutes as route}
							<li
								class="dropdown-item"
								role="button"
								tabindex="0"
								onclick={() => navigateFromDropdown(path, route)}
							>
								{route}
							</li>
						{/each}
					</ol>
				{/if}
				{#if index < breadcrumbs.length - 1}<span> / </span>{/if}
			</li>
		{/each}
		<li>
			<button tabindex="0" onclick={rowAdd}>
				<img class="remixicon" src={RiAddLine} />
			</button>
		</li>
	</ol>
</nav>

<style>
	ol {
		display: flex;
		list-style: none;
		padding: 0;
		margin-top: 20px;
	}

	li {
		padding: 5px;
		color: #4d4d4d;
		cursor: pointer;
	}

	li:hover {
		text-decoration: underline;
	}

	li span {
		padding-left: 5px;
	}

	li:not(:last-child)::after {
		margin-left: 10px;
		color: #ccc;
	}

	.breadcrumb-item {
		position: relative;
	}

	.dropdown {
		flex-direction: column;
		position: absolute;
		left: 20px;
		background-color: #f9f9f9;
		box-shadow: 0 8px 16px 0 rgba(0, 0, 0, 0.2);
		z-index: 1;
	}

	.dropdown li {
		padding: 8px 16px;
		cursor: pointer;
	}

	.dropdown li:hover {
		background-color: #f1f1f1;
	}
</style>
