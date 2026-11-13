<script lang="ts">
	import { onMount } from "svelte";
	import { formData } from "./formdata.svelte";
	import { adminDbUrl } from "../config.svelte";
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

	const { initial: initialData, current: currentData } = formData<Table>(() => data);
	let nameInput: HTMLInputElement;

	const entityName = "table";
	const db = router.params["db"];
	const dataUrl = `/databases/${db}/tables`;

	let prom_schemas: Promise<Role[]> = getData(`${adminDbUrl}/databases/${db}/schemas`);
	let prom_roles: Promise<Role[]> = getData(`${adminDbUrl}/roles`);

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
		<label class="field-label" for="schema">Schema</label>
		<Select id="schema" bind:value={currentData.schema} data={prom_schemas} fieldName="name"></Select>
	</div>
	<div class="field">
		<label class="field-label" for="owner">Owner</label>
		<Select id="owner" bind:value={currentData.owner} data={prom_roles} fieldName="name"></Select>
	</div>
	<label class="toggle-row" for="login">
		<input id="login" class="switch" type="checkbox" bind:checked={currentData.rowsecurity} />
		<span>Row security</span>
	</label>
	<label class="toggle-row" for="inherit">
		<input id="inherit" class="switch" type="checkbox" disabled bind:checked={currentData.hasindexes} />
		<span>Has indexes</span>
	</label>
	<label class="toggle-row" for="superuser">
		<input id="superuser" class="switch" type="checkbox" disabled bind:checked={currentData.hastriggers} />
		<span>Has triggers</span>
	</label>
	<label class="toggle-row" for="createdb">
		<input id="createdb" class="switch" type="checkbox" disabled bind:checked={currentData.ispartition} />
		<span>Is partition</span>
	</label>
</RecordForm>
