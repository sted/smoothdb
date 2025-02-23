<script lang="ts">
    import type { Snippet } from "svelte";
    import { adminDbUrl } from "../main";
    import { router } from "../routes";
    import type { Data } from "../api";

    interface Props {
        entityName: string;
        dataUrl: string;
        initialData: Data;
        currentData: Data;
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
        const schema = initialData.schema ?? router.schema;

        fetch(url, {
            method,
            headers: { "Content-Type": "application/json", "Content-Profile": schema },
            body: JSON.stringify(currentData),
        })
        .then(() => formSubmitted(true))
        .catch((error) => console.error("Failed to create or update database:", error));
    }

    async function handleDelete() {
        const method = "DELETE";
        const url = adminDbUrl + dataUrl + `/${initialData.name}`;
        const schema = initialData.schema ?? router.schema;

        fetch(url, { 
            method,
            headers: { "Content-Profile": schema },
        })
        .then(() => formSubmitted(true))
        .catch((error) => console.error("Failed to drop database:", error));
    }
</script>

<h3>{(isEditing ? "Edit" : "Create") + " " + entityName}</h3>

<form onsubmit={handleSubmit}>
    <div class="control-container">
        {@render children()}
    </div>

    <div class="button-container">
        {#if isEditing}
            <button class="button-drop" type="button" onclick={handleDelete}>Drop</button>
        {/if}
        <button class="button-primary" type="submit" disabled={isButtonDisabled}
            >{isEditing ? "Update" : "Create"}
        </button>
        <button class="button-cancel" type="button" onclick={()=>formSubmitted(false)}>Cancel</button>
    </div>
</form>

<style>
    h3 {
        margin-bottom: 10px;
    }

    form {
        display: flex;
        flex-direction: column;
        flex: 1;
        overflow: hidden;
    }

    .control-container {
        display: flex;
        flex-direction: column;
        flex: 1;
        gap: 20px;
        overflow: auto;
        padding: 2px;
    }
    
    .button-container {
        display: flex;
        justify-content: flex-start;
        margin-top: 10px;
    }

    button {
        padding: 10px;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }

    .button-primary {
        background-color: #10378cca;
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
