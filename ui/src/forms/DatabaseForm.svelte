<script lang="ts">
    import { onMount } from "svelte";
    import { formData } from "./formdata.svelte";
    import { adminDbUrl } from "../config.svelte";
    import { getData } from "../utils";
    import RecordForm from "../components/RecordForm.svelte";
    import Select from "../components/DataSelect.svelte";
    import Role from "./RoleForm.svelte";

    interface Props {
        data: any;
        formSubmitted: (refreshTable: boolean) => void;
    }
    let { data, formSubmitted }: Props = $props();

    interface Database {
        name: string;
        owner: string;
    }

    const { initial: initialData, current: currentData } = formData<Database>(() => data);
    let nameInput: HTMLInputElement;

    const entityName = "database";
    const dataUrl = "/databases";

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
		<label class="field-label" for="owner">Owner</label>
		<Select id="owner" bind:value={currentData.owner} data={prom_roles} fieldName="name"></Select>
	</div>
</RecordForm>
