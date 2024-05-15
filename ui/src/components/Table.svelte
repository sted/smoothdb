<script lang="ts">
	import { onMount, onDestroy } from "svelte";
	import { createEventDispatcher } from "svelte";
	import { shouldRefresh } from "../stores";
	import RiPencilLine from "svelte-remixicon/RiPencilLine.svelte";
	import type { Unsubscriber } from "svelte/store";

	export let url: string;
	export let limit: number = 0;
	export let sortableColumns: string[] = [];
	export let columnsDef: string[] = [];

	let unsubscribe: Unsubscriber;

	onMount(() => {
		fetchData();

		unsubscribe = shouldRefresh.subscribe((value) => {
			if (value) {
				fetchData();
				shouldRefresh.set(false);
			}
		});
	});

	onDestroy(() => {
		unsubscribe();
	});

	type DataItem = Record<string, any>;

	let data: DataItem[] = [];
	interface Column {
		name: string;
		sortable: boolean;
	}
	let columns: Column[] = [];
	let offset: number = 0;
	const dispatch = createEventDispatcher();

	$: columns =
		columnsDef.length > 0
			? columnsDef.map((name) => ({
					name,
					sortable: sortableColumns.includes(name),
				}))
			: data.length > 0
				? Object.keys(data[0]).map((name) => ({
						name,
						sortable: sortableColumns.includes(name),
					}))
				: [];

	async function fetchData(): Promise<void> {
		if (limit > 0) {
			url += `?offset=${offset}&limit=${limit}`;
		}

		const response = await fetch(url);
		data = await response.json();

		if (limit > 0) {
			//total = result.total;
		}
	}

	// function nextPage(): void {
	// 	if (limit > 0 && offset + limit < total) {
	// 		offset += limit;
	// 		fetchData();
	// 	}
	// }

	// function prevPage(): void {
	// 	if (limit > 0 && offset - limit >= 0) {
	// 		offset -= limit;
	// 		fetchData();
	// 	}
	// }

	function handleBodyClick(event: MouseEvent) {
		const row = (event.target as HTMLElement).closest("tr");
		const cell = (event.target as HTMLElement).closest("td");
		if (row && row.dataset.index) {
			const rowIndex = parseInt(row.dataset.index, 10);
			if (cell?.classList.contains("edit")) {
				dispatch("rowEdit", data[rowIndex]);
			} else {
				dispatch("rowClick", data[rowIndex]);
			}
		}
	}
</script>

<!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
<table>
	<thead on:click={handleBodyClick} on:keyup={() => {}}>
		<tr>
			<th></th>
			{#each columns as column}
				<th>{column.name}</th>
			{/each}
		</tr>
	</thead>
	<tbody on:click={handleBodyClick} on:keyup={() => {}}>
		{#if data.length > 0}
			{#each data as row, index}
				<tr data-index={index}>
					<td class="edit"><RiPencilLine /></td>
					{#each columns as column}
						<td>{row[column.name]}</td>
					{/each}
				</tr>
			{/each}
		{:else}
			<tr><td colspan={columns.length}>No data</td></tr>
		{/if}
	</tbody>
</table>

<!-- {#if limit > 0}
	<button on:click={prevPage} disabled={offset === 0}>Precedente</button>
	<button on:click={nextPage} disabled={offset + limit >= total}
		>Successivo</button
	>
{/if} -->

<style>
	table {
		width: 100%;
		border-spacing: 0;
		text-align: left;
	}
	thead {
		color: rgb(98, 98, 98);
		background-color: rgb(255, 255, 255);
		position: sticky;
		top: 0;
	}
	thead th:nth-child(1) {
		width: 30px;
	}
	thead th:hover {
		background-color: rgb(240, 240, 240);
		cursor: pointer;
	}
	tbody {
		background-color: rgb(247, 252, 255);
	}
	tbody tr:hover {
		background-color: rgb(240, 240, 240);
	}
	th,
	td {
		min-width: 100px;
		border-bottom: 1px solid rgb(235, 235, 235);
		padding: 10px;
		outline: 0;
	}
	tr > td:nth-child(1),
	tr > td:nth-child(2) {
		cursor: pointer;
		text-decoration: underline;
	}
</style>
