<script lang="ts">
    import { onMount } from "svelte";
    import RecordForm from "../components/RecordForm.svelte";

    interface Props {
        data: any;
        formSubmitted: (refreshTable: boolean) => void;
    }
    let { data, formSubmitted }: Props = $props();

    interface Role {
        name: string;
        canlogin: boolean;
        noinherit: boolean;
        issuperuser: boolean;
        cancreatedatabases: boolean;
        cancreateroles: boolean;
        canbypassrls: boolean;
    }

    const initialData: Role = $state.snapshot(data);
    let currentData: Role = $state(data);
    let nameInput: HTMLInputElement;

    const entityName = "role";
    const dataUrl = "/roles";

    onMount(() => {
        nameInput.focus();
    });
</script>

<RecordForm {entityName} {dataUrl} {initialData} {currentData} {formSubmitted}>
    <label for="name">
        Name
        <input id="name" type="text" bind:value={currentData.name} bind:this={nameInput} />
    </label>
    <div>
        <input id="login" type="checkbox" bind:checked={currentData.canlogin} />
        <label for="login">Can login</label>
    </div>
    <div>
        <input id="inherit" type="checkbox" bind:checked={currentData.noinherit} />
        <label for="inherit">No inherit privileges</label>
    </div>
    <div>
        <input id="superuser" type="checkbox" bind:checked={currentData.issuperuser} />
        <label for="superuser">Superuser</label>
    </div>
    <div>
        <input id="createdb" type="checkbox" bind:checked={currentData.cancreatedatabases} />
        <label for="createdb">Can create databases</label>
    </div>
    <div>
        <input id="createrole" type="checkbox" bind:checked={currentData.cancreateroles} />
        <label for="createrole">Can create roles</label>
    </div>
    <div>
        <input id="bypassrls" type="checkbox" bind:checked={currentData.canbypassrls} />
        <label for="bypassrls">Can bypass RLS</label>
    </div>
</RecordForm>
