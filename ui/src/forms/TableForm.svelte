<script lang="ts">
	import { onMount } from "svelte";
	import { adminDbUrl } from "../main";
	import { router } from "../routes";
	import { getData } from "../utils";
	import RecordForm from "../components/RecordForm.svelte";
	import Select from "../components/DataSelect.svelte";
	import Role from "./RoleForm.svelte";

	interface Props {
		data: any;
		formSubmitted: (refreshTable: boolean) => void;
	}
	let { data, formSubmitted }: Props = $props();

	interface Table {
		name: string;
		schema: string;
		owner: string;
		rowsecurity: boolean;
		hasindexes: boolean;
		hastriggers: boolean;
		ispartition: boolean;
	}

	const initialData: Table = $state.snapshot(data);
	let currentData: Table = $state(data);
	let nameInput: HTMLInputElement;

	const entityName = "table";
	const db = $router.params["db"];
	const dataUrl = `/databases/${db}/tables`;

	let prom_schemas: Promise<Role[]> = getData(`${adminDbUrl}/databases/${db}/schemas`);
	let prom_roles: Promise<Role[]> = getData(`${adminDbUrl}/roles`);

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
		<label for="schema"> Schema </label>
		<Select id="schema" bind:value={currentData.schema} data={prom_schemas} fieldName="name"
		></Select>
	</div>
	<div>
		<label for="owner"> Owner </label>
		<Select id="schema" bind:value={currentData.owner} data={prom_roles} fieldName="name"
		></Select>
	</div>
	<div>
		<input id="login" type="checkbox" bind:checked={currentData.rowsecurity} />
		<label for="login">Row security</label>
	</div>
	<div>
		<input id="inherit" type="checkbox" disabled bind:checked={currentData.hasindexes} />
		<label for="inherit">Has indexes</label>
	</div>
	<div>
		<input id="superuser" type="checkbox" disabled bind:checked={currentData.hastriggers} />
		<label for="superuser">Has triggers</label>
	</div>
	<div>
		<input id="createdb" type="checkbox" disabled bind:checked={currentData.ispartition} />
		<label for="createdb">Is partition</label>
	</div>
</RecordForm>
