<script lang="ts">
	import { onMount } from "svelte";
	import { formData } from "./formdata.svelte";
	import { adminDbUrl } from "../config.svelte";
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

	const { initial: initialData, current: currentData } = formData<Constraint>(() => data);
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
	<div class="field">
		<label class="field-label" for="name">Name</label>
		<input id="name" class="input" type="text" autocomplete="off" bind:value={currentData.name} bind:this={nameInput} />
	</div>
	<div class="field">
		<label class="field-label" for="type">Type</label>
		<Select id="type" bind:value={currentData.type} data={types}></Select>
	</div>
</RecordForm>
