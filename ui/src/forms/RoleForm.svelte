<script lang="ts">
    import { onMount } from "svelte";
    import { adminDbUrl } from "../main";
    import { modalStore, shouldRefresh } from "../stores";

    interface Role {
        name: string;
        canlogin: boolean;
        noinherit: boolean;
        issuperuser: boolean;
        cancreatedatabases: boolean;
        cancreateroles: boolean;
        canbypassrls: boolean;
    }

    let editData: Role = {
        name: "",
        canlogin: false,
        noinherit: false,
        issuperuser: false,
        cancreatedatabases: false,
        cancreateroles: false,
        canbypassrls: false,
    };
    let initialData: Role;
    let isEditing = Boolean($modalStore.data.name);
    let isButtonDisabled = isEditing;
    let nameInput: HTMLInputElement;

    $: isButtonDisabled = isEditing && JSON.stringify(initialData) === JSON.stringify(editData);

    function closeModal() {
        modalStore.set({ showModal: false, data: {} });
    }

    if (isEditing) {
        initialData = $modalStore.data;

        fetch(`${adminDbUrl}/roles/${initialData.name}`)
            .then((response) => response.json())
            .then((data) => {
                editData = data;
            })
            .catch((error) => console.error("Failed to fetch role details:", error));
    }

    onMount(() => {
        nameInput.focus();
    });

    async function handleSubmit() {
        const method = isEditing ? "PATCH" : "POST";
        const url = adminDbUrl + (isEditing ? `/roles/${initialData.name}` : "/roles");

        fetch(url, {
            method,
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(editData),
        })
            .then(() => {
                shouldRefresh.set(true);
                closeModal();
            })
            .catch((error) => console.error("Failed to create or update database:", error));
    }

    async function handleDrop() {
        const method = "DELETE";
        const url = adminDbUrl + `/roles/${initialData.name}`;

        fetch(url, { method })
            .then(() => {
                shouldRefresh.set(true);
                closeModal();
            })
            .catch((error) => console.error("Failed to drop database:", error));
    }
</script>

<h3>{(isEditing ? "Edit" : "Create") + " role"}</h3>

<form on:submit|preventDefault={handleSubmit}>
    <label for="name">
        Name
        <input id="name" type="text" bind:value={editData.name} bind:this={nameInput} />
    </label>
    <div>
        <input id="login" type="checkbox" bind:checked={editData.canlogin} />
        <label for="login">Can login</label>
    </div>
    <div>
        <input id="inherit" type="checkbox" bind:checked={editData.noinherit} />
        <label for="inherit">No inherit privileges</label>
    </div>
    <div>
        <input id="superuser" type="checkbox" bind:checked={editData.issuperuser} />
        <label for="superuser">Superuser</label>
    </div>
    <div>
        <input id="createdb" type="checkbox" bind:checked={editData.cancreatedatabases} />
        <label for="createdb">Can create databases</label>
    </div>
    <div>
        <input id="createrole" type="checkbox" bind:checked={editData.cancreateroles} />
        <label for="createrole">Can create roles</label>
    </div>
    <div>
        <input id="bypassrls" type="checkbox" bind:checked={editData.canbypassrls} />
        <label for="bypassrls">Can bypass RLS</label>
    </div>

    <div class="button-container">
        {#if isEditing}
            <button class="button-drop" type="button" on:click={handleDrop}>Drop</button>
        {/if}
        <button class="button-primary" type="submit" disabled={isButtonDisabled}
            >{isEditing ? "Update" : "Create"}
        </button>
        <button class="button-cancel" type="button" on:click={closeModal}>Cancel</button>
    </div>
</form>

<style>
    form {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    input {
        margin-top: 5px;
        padding: 8px;
        border: 1px solid #ccc;
        border-radius: 4px;
    }

    .button-container {
        display: flex;
        justify-content: flex-start;
        margin-top: 20px;
    }

    button {
        padding: 10px;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }

    .button-primary {
        background-color: #007bff;
        margin-left: auto;
    }

    .button-drop {
        background-color: rgb(255, 0, 0);
    }
    .button-cancel {
        background-color: #6e6e6e;
        margin-left: 10px;
    }

    button:hover {
        filter: brightness(80%);
    }

    button:disabled {
        pointer-events: none;
        opacity: 0.5;
    }
</style>
