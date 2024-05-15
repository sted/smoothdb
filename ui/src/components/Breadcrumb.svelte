<script lang="ts">
	import { router } from "../routes";
	import { modalStore } from "../stores";

	import RiExpandUpDownLine from "svelte-remixicon/RiExpandUpDownLine.svelte";
	import RiAddLine from "svelte-remixicon/RiAddLine.svelte";

	interface Breadcrumb {
		title: string;
		path: string;
		hasMultipleChoices: boolean; // New property to track if there are multiple choices
	}

	const titleize = (segment: string): string => {
		return segment.charAt(0).toUpperCase() + segment.slice(1).replace(/-/g, " ");
	};

	let breadcrumbs: Breadcrumb[] = [];
	let activeDropdownIndex: number | null = null;
	let dropdownRoutes: string[] = [];

	router.subscribe(() => {
		const segments = window.location.pathname.split("/").filter(Boolean);
		breadcrumbs = segments.map((segment, index) => {
			const path = "/" + segments.slice(0, index + 1).join("/");
			const nextRoutes = router.getAltRoutes(path);
			return {
				title: titleize(segment),
				path,
				hasMultipleChoices: nextRoutes.length > 1, // Set based on the number of alternate routes
			};
		});
		activeDropdownIndex = null;
	});

	function handleSegmentClick(event: MouseEvent, path: string): void {
		router.navigate(path);
		closeDropdown();
	}

	function handleDropdownClick(event: MouseEvent, path: string, index: number): void {
		event.stopPropagation(); // Prevent the segment click event

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

	function handleAddClick() {
		modalStore.set({ showModal: true, data: {} });
	}
</script>

<svelte:window on:click={handleClickOutside} />

<nav aria-label="breadcrumb">
	<ol>
		{#each breadcrumbs as { title, path, hasMultipleChoices }, index}
			<li class="breadcrumb-item">
				<span
					role="button"
					tabindex="0"
					on:click|preventDefault={(event) => handleSegmentClick(event, path)}
					>{title}</span
				>
				{#if hasMultipleChoices}
					<button
						tabindex="0"
						on:click={(event) => handleDropdownClick(event, path, index)}
					>
						<RiExpandUpDownLine />
					</button>
				{/if}
				{#if index === activeDropdownIndex}
					<ol class="dropdown">
						{#each dropdownRoutes as route}
							<li
								class="dropdown-item"
								role="button"
								tabindex="0"
								on:click={() => navigateFromDropdown(path, route)}
							>
								{titleize(route)}
							</li>
						{/each}
					</ol>
				{/if}
				{#if index < breadcrumbs.length - 1}<span> / </span>{/if}
			</li>
		{/each}
		<li>
			<button tabindex="0" on:click={() => handleAddClick()}>
				<RiAddLine />
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
		position: relative; /* Per posizionare il dropdown relativo a questo elemento */
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
