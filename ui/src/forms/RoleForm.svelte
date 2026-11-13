<script lang="ts">
    import { onMount } from "svelte";
    import { formData } from "./formdata.svelte";
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

    const { initial: initialData, current: currentData } = formData<Role>(() => data);
    let nameInput: HTMLInputElement;

    const entityName = "role";
    const dataUrl = "/roles";

    onMount(() => {
        nameInput.focus();
    });
</script>

<RecordForm {entityName} {dataUrl} {initialData} {currentData} {formSubmitted}>
    <div class="field">
		<label class="field-label" for="name">Name</label>
		<input id="name" class="input" type="text" autocomplete="off" bind:value={currentData.name} bind:this={nameInput} />
	</div>
    <label class="toggle-row" for="login">
		<input id="login" class="switch" type="checkbox" bind:checked={currentData.canlogin} />
		<span>Can login</span>
	</label>
    <label class="toggle-row" for="inherit">
		<input id="inherit" class="switch" type="checkbox" bind:checked={currentData.noinherit} />
		<span>No inherit privileges</span>
	</label>
    <label class="toggle-row" for="superuser">
		<input id="superuser" class="switch" type="checkbox" bind:checked={currentData.issuperuser} />
		<span>Superuser</span>
	</label>
    <label class="toggle-row" for="createdb">
		<input id="createdb" class="switch" type="checkbox" bind:checked={currentData.cancreatedatabases} />
		<span>Can create databases</span>
	</label>
    <label class="toggle-row" for="createrole">
		<input id="createrole" class="switch" type="checkbox" bind:checked={currentData.cancreateroles} />
		<span>Can create roles</span>
	</label>
    <label class="toggle-row" for="bypassrls">
		<input id="bypassrls" class="switch" type="checkbox" bind:checked={currentData.canbypassrls} />
		<span>Can bypass RLS</span>
	</label>
</RecordForm>
