<script lang="ts">
	import { onMount } from "svelte";
	import { router } from "../routes";
	import type { Data } from "../api";
	import RiEditLine from "/assets/images/edit-line.svg";

	interface Props {
		dataUrl: string;
		rowClick: Function;
		rowEdit: Function;
	}
	let { dataUrl, rowClick, rowEdit }: Props = $props();

	interface Column {
		name: string;
	}
	let data: Data[] = $state([]);
	let columns: Column[] = $derived.by(() => {
		return data.length > 0 ? Object.keys(data[0]).map((name) => ({ name })) : [];
	});

	$effect(() => {
		dataUrl;
		refresh();
	});

	onMount(() => {
		fetchData();
	});

	export function refresh() {
		fetchData();
	}

	async function fetchData(): Promise<void> {
		const response = await fetch(dataUrl, {
			headers: { "Accept-Profile": router.schema },
		});
		if (!response.ok) {
			throw new Error("Failed to fetch: " + response.statusText);
		}
		data = await response.json();
	}

	function handleBodyClick(event: MouseEvent) {
		const row = (event.target as HTMLElement).closest("tr");
		const cell = (event.target as HTMLElement).closest("td");
		if (row && row.dataset.index) {
			const rowIndex = parseInt(row.dataset.index, 10);
			const d = data[rowIndex];
			if (cell?.classList.contains("edit")) {
				rowEdit(d);
			} else {
				rowClick(d);
			}
		}
	}
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<table>
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<thead onclick={handleBodyClick}>
		<tr>
			<th></th>
			{#each columns as column}
				<th>{column.name}</th>
			{/each}
		</tr>
	</thead>
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<tbody onclick={handleBodyClick}>
		{#if data.length > 0}
			{#each data as row, index}
				<tr data-index={index}>
					<!-- svelte-ignore a11y_missing_attribute -->
					<td class="edit"><img class="remixicon" src={RiEditLine} /></td>
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
		background-color: rgb(250, 250, 250);
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
	tr {
		cursor: pointer;
	}
</style>
