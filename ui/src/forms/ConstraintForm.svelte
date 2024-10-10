<script lang="ts">
	import { onMount } from "svelte";
	import { adminDbUrl } from "../main";
	import { router } from "../routes";
	import { getData } from "../utils";
	import RecordForm from "../components/RecordForm.svelte";
	import Select from "../components/DataSelect.svelte";

	interface Props {
		data: any;
		formSubmitted: (refreshTable: boolean) => void;
	}
	let { data, formSubmitted }: Props = $props();

	interface Constraint {
		name: string;
		type: string;
		//columns: string[];
	}

	const initialData: Constraint = $state.snapshot(data);
	let currentData: Constraint = $state(data);
	currentData.type ||= "";
	let nameInput: HTMLInputElement;

	const entityName = "constraint";
	const db = router.params["db"];
	const table = router.params["table"];
	const dataUrl = `/databases/${db}/tables/${table}/constraints`;

	const types = {
		"primary key": "primary",
		unique: "unique",
		check: "check",
		"foreign key": "foreign",
	};

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
</RecordForm>
