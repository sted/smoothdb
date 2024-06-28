<script lang="ts">
    import { onMount } from "svelte";
    import { adminDbUrl } from "../main";
    import { getData } from "../utils";
    import RecordForm from "../components/RecordForm.svelte";
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

    const initialData: Database = $state.snapshot(data);
    let currentData: Database = $state(data);
    let nameInput: HTMLInputElement;

    const entityName = "database";
    const dataUrl = "/databases";

    let prom_roles: Promise<Role[]> = getData(`${adminDbUrl}/roles`);

    onMount(() => {
        nameInput.focus();
    });
</script>

<RecordForm {entityName} {dataUrl} {initialData} {currentData} {formSubmitted}>
    <div>
        <label for="name">Name</label>
        <input id="name" type="text" bind:value={currentData.name} bind:this={nameInput} />
    </div>
    <div>
        <label for="owner">Owner</label>
        <select id="owner" bind:value={currentData.owner}>
            {#await prom_roles then roles}
                {#each roles as role}
                    <option value={role.name}>{role.name}</option>
                {/each}
            {/await}
        </select>
    </div>
</RecordForm>
