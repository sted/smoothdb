<script lang="ts">
	import { onMount } from "svelte";
	import { formData } from "./formdata.svelte";
	import { adminDbUrl } from "../config.svelte";
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

	const { initial: initialData, current: currentData } = formData<Column>(() => data);
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
	<div class="field">
		<label class="field-label" for="name">Name</label>
		<input id="name" class="input" type="text" autocomplete="off" bind:value={currentData.name} bind:this={nameInput} />
	</div>
	<div class="field">
		<label class="field-label" for="type">Type</label>
		<Select id="type" bind:value={currentData.type} data={types}></Select>
	</div>
	<label class="toggle-row" for="notnull">
		<input id="notnull" class="switch" type="checkbox" bind:checked={currentData.notnull} />
		<span>Not null</span>
	</label>
	<div class="field">
		<label class="field-label" for="default">Default</label>
		<input id="default" class="input" type="text" autocomplete="off" bind:value={currentData.default} />
	</div>
</RecordForm>
