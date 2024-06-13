<script lang="ts">
    import type { Snippet } from "svelte";
    import { adminDbUrl } from "../main";

    interface Props {
        entityName: string;
        dataUrl: string;
        initialData: any;
        currentData: any;
        formSubmitted: (b:boolean) => void;
        children: Snippet;
    }
    let { entityName, dataUrl, initialData, currentData, formSubmitted, children }: Props = $props();
    const isEditing = Boolean(initialData.name);
    let isButtonDisabled = $derived(
        isEditing && JSON.stringify(initialData) === JSON.stringify(currentData));

    async function handleSubmit(ev: Event) {
        ev.preventDefault();
        const method = isEditing ? "PATCH" : "POST";
        const url = adminDbUrl + dataUrl + (isEditing ? `/${initialData.name}`: '');

        fetch(url, {
            method,
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(currentData),
        })
            .then(() => formSubmitted(true))
            .catch((error) => console.error("Failed to create or update database:", error));
    }

    async function handleDrop() {
        const method = "DELETE";
        const url = adminDbUrl + dataUrl + `/${initialData.name}`;

        fetch(url, { method })
            .then(() => formSubmitted(true))
            .catch((error) => console.error("Failed to drop database:", error));
    }
</script>

<h3>{(isEditing ? "Edit" : "Create") + " " + entityName}</h3>

<form onsubmit={handleSubmit}>
    {@render children()}

    <div class="button-container">
        {#if isEditing}
            <button class="button-drop" type="button" onclick={handleDrop}>Drop</button>
        {/if}
        <button class="button-primary" type="submit" disabled={isButtonDisabled}
            >{isEditing ? "Update" : "Create"}
        </button>
        <button class="button-cancel" type="button" onclick={()=>formSubmitted(false)}>Cancel</button>
    </div>
</form>

<style>
    form {
        display: flex;
        flex-direction: column;
        gap: 20px;
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
        background-color: #007bff9e;
        margin-left: auto;
    }

    .button-drop {
        background-color: rgb(255, 102, 102);
    }
    .button-cancel {
        background-color: #6e6e6ed2;
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
