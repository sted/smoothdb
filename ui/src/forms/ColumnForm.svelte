<script lang="ts">
	import { onMount } from "svelte";
	import { adminDbUrl } from "../main";
	import { router } from "../routes";
	import { getData } from "../utils";
	import RecordForm from "../components/RecordForm.svelte";

	interface Props {
		data: any;
		formSubmitted: (refreshTable: boolean) => void;
	}
	let { data, formSubmitted }: Props = $props();

	interface Column {
		name: string;
		type: string;
		notnull: boolean;
	}

	const initialData: Column = $state.snapshot(data);
	let currentData: Column = $state(data);
	let nameInput: HTMLInputElement;

	const entityName = "column";
	const db = $router.params["db"];
	const table = $router.params["table"];
	const dataUrl = `/databases/${db}/tables/${table}/columns`;

	const types = ["int2", "int4", "int8", "float4", "float8", "bool", "text"];

	onMount(() => {
		nameInput.focus();
	});
</script>

<RecordForm {entityName} {dataUrl} {initialData} {currentData} {formSubmitted}>
	<label for="name">
		Name
		<input id="name" type="text" bind:value={currentData.name} bind:this={nameInput} />
	</label>
	<label for="type">
		Type
		<select id="type" bind:value={currentData.type}>
			{#each types as type}
				<option value={type}>{type}</option>
			{/each}
		</select>
	</label>
	<div>
		<input id="notnull" type="checkbox" bind:checked={currentData.notnull} />
		<label for="notnull">Not null</label>
	</div>
</RecordForm>
