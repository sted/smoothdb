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

    interface Schema {
        name: string;
        owner: string;
    }

    const initialData: Schema = $state.snapshot(data);
    let currentData: Schema = $state(data);
    let nameInput: HTMLInputElement;

    const entityName = "schema";
    const db = router.params["db"];
    const dataUrl = `/databases/${db}/schemas`;

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
        <label for="owner"> Owner </label>
        <Select id="owner" bind:value={currentData.owner} data={prom_roles} fieldName="name"
        ></Select>
    </div>
</RecordForm>
