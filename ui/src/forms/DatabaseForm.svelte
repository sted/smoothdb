<script lang="ts">
    import { onMount } from "svelte";
    import { adminDbUrl } from "../main";
    import { modalStore, shouldRefresh } from "../stores";
    import { getData } from "../utils";
    import Role from "./RoleForm.svelte";

    interface Database {
        name: string;
        owner: string;
    }

    let editData: Database = {
        name: "",
        owner: "",
    };
    let initialData: Database;
    let isEditing = Boolean($modalStore.data.name);
    let isButtonDisabled = isEditing;
    let nameInput: HTMLInputElement;
    let rolesData: Role[];

    let prom_db: Promise<any>;
    let prom_roles: Promise<any>;
    let prom_all: Promise<any[]>;

    $: isButtonDisabled = isEditing && JSON.stringify(initialData) === JSON.stringify(editData);

    function closeModal() {
        modalStore.set({ showModal: false, data: {} });
    }

    async function handleSubmit() {
        const method = isEditing ? "PATCH" : "POST";
        const url = adminDbUrl + (isEditing ? `/databases/${initialData.name}` : "/databases");

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

    async function handleDelete() {
        const method = "DELETE";
        const url = adminDbUrl + `/databases/${initialData.name}`;

        fetch(url, { method })
            .then(() => {
                shouldRefresh.set(true);
                closeModal();
            })
            .catch((error) => console.error("Failed to drop database:", error));
    }

    if (isEditing) {
        initialData = $modalStore.data;
        prom_db = getData(`${adminDbUrl}/databases/${initialData.name}`);
    } else {
        prom_db = Promise.resolve(editData);
    }
    prom_roles = getData(`${adminDbUrl}/roles`);
    prom_all = Promise.all([prom_db, prom_roles]);
    prom_all.then(([database, roles]) => {
        editData = database;
        rolesData = roles;
    });

    onMount(() => {
        nameInput.focus();
    });
</script>

<h3>{(isEditing ? "Edit" : "Create") + " database"}</h3>

<form on:submit|preventDefault={handleSubmit}>
    {#await prom_all then []}
        <label for="name">
            <b>Name</b>
            <input id="name" type="text" bind:value={editData.name} bind:this={nameInput} />
        </label>
        <label for="owner">
            <b>Owner</b>
            <select id="owner" bind:value={editData.owner}>
                {#each rolesData as role}
                    <option value={role.name}>{role.name}</option>
                {/each}
            </select>
        </label>
    {/await}

    <div class="button-container">
        {#if isEditing}
            <button class="button-drop" type="button" on:click={handleDelete}>Drop</button>
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

    label {
        display: flex;
        flex-direction: column;
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
