<script lang="ts">
	import { onMount } from "svelte";
	import { adminDbUrl } from "../main";
	import { router } from "../routes";
	import { getData } from "../utils";
	import RecordForm from "../components/RecordForm.svelte";
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
	<label for="name">
		Name
		<input id="name" type="text" bind:value={currentData.name} bind:this={nameInput} />
	</label>
	<label for="schema">
		Schema
		<select id="schema" bind:value={currentData.schema}>
			{#await prom_schemas then schemas}
				{#each schemas as schema}
					<option value={schema.name}>{schema.name}</option>
				{/each}
			{/await}
		</select>
	</label>
	<label for="owner">
		Owner
		<select id="owner" bind:value={currentData.owner}>
			{#await prom_roles then roles}
				{#each roles as role}
					<option value={role.name}>{role.name}</option>
				{/each}
			{/await}
		</select>
	</label>
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
