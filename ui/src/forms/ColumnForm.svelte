<script lang="ts">
	import { onMount } from "svelte";
	import { adminDbUrl } from "../main";
	import { router } from "../routes";
	import { getData } from "../utils";
	import RecordForm from "../components/RecordForm.svelte";
	import Select from "../components/DataSelect.svelte";
	import Constraint from "./ConstraintForm.svelte";

	interface Props {
		data: any;
		formSubmitted: (refreshTable: boolean) => void;
	}
	let { data, formSubmitted }: Props = $props();

	interface Column {
		name: string;
		type: string;
		notnull: boolean;
		default: string;
		constraints: string;
	}

	const initialData: Column = $state.snapshot(data);
	let currentData: Column = $state(data);
	currentData.type ||= "";
	let nameInput: HTMLInputElement;

	const entityName = "column";
	const db = router.params["db"];
	const table = router.params["table"];
	const dataUrl = `/databases/${db}/tables/${table}/columns`;

	const types = [
		"int2",
		"int4",
		"int8",
		"float4",
		"float8",
		"bool",
		"text",
		"timestamp",
		"date",
		"interval",
		"json",
		"jsonb",
	];

	onMount(() => {
		nameInput.focus();
	});
</script>

<RecordForm {entityName} {dataUrl} {initialData} {currentData} {formSubmitted}>
	<div>
		<label for="name"> Name </label>
		<input id="name" type="text" bind:value={currentData.name} bind:this={nameInput} />
	</div>
	<div>
		<label for="type"> Type </label>
		<Select id="type" bind:value={currentData.type} data={types}></Select>
	</div>
	<div>
		<input id="notnull" type="checkbox" bind:checked={currentData.notnull} />
		<label for="notnull">Not null</label>
	</div>
	<div>
		<label for="default"> Default </label>
		<input id="default" type="text" bind:value={currentData.default} />
	</div>
</RecordForm>
