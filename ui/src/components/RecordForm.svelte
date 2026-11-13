<script lang="ts">
    import type { Snippet } from "svelte";
    import { adminDbUrl } from "../config.svelte";
    import { router } from "../routes";
    import { makeRequest } from "../api";
    import type { Data } from "../api";
    import { showInfo } from "../toast.svelte";
    import { confirmDialog } from "../dialog.svelte";
    import { forget } from "../data.svelte";
    import Icon from "./Icon.svelte";

    // a schema change makes the cached table definitions of the Data area stale
    function changed() {
        if (router.params.db) forget(router.params.db);
        formSubmitted(true);
    }

    interface Props {
        entityName: string;
        dataUrl: string;
        initialData: Data;
        currentData: Data;
        formSubmitted: (b:boolean) => void;
        children: Snippet;
    }
    let { entityName, dataUrl, initialData, currentData, formSubmitted, children }: Props = $props();
    let isEditing = $derived(Boolean(initialData.name));
    let isButtonDisabled = $derived(
        isEditing && JSON.stringify(initialData) === JSON.stringify(currentData));

    let busy = $state(false);

    function entityUrl(): string {
        return adminDbUrl + dataUrl + (isEditing ? `/${encodeURIComponent(initialData.name)}` : "");
    }

    function profile(): Record<string, string> {
        const schema = initialData.schema ?? router.schema;
        return schema ? { "Content-Profile": schema } : {};
    }

    // On failure the error toast reports the cause and the panel stays open.
    async function handleSubmit(ev: Event) {
        ev.preventDefault();
        busy = true;
        const response = await makeRequest(entityUrl(), {
            method: isEditing ? "PATCH" : "POST",
            headers: { "Content-Type": "application/json", ...profile() },
            body: JSON.stringify(currentData),
        });
        busy = false;
        if (response.success) changed();
    }

    async function handleDelete() {
        const ok = await confirmDialog({
            title: `Drop ${entityName} ${initialData.name}?`,
            message: `The ${entityName} and everything in it are removed from the database. You cannot undo this.`,
            confirmLabel: `Drop ${entityName}`,
            danger: true,
        });
        if (!ok) return;
        busy = true;
        const response = await makeRequest(entityUrl(), { method: "DELETE", headers: profile() });
        busy = false;
        if (response.success) {
            showInfo(`Dropped ${entityName} "${initialData.name}"`);
            changed();
        }
    }
</script>

<form class="sheet-form" onsubmit={handleSubmit}>
    <div class="sheet-header">
        <div class="sheet-title">
            <h2>{isEditing ? `Edit ${entityName}` : `New ${entityName}`}</h2>
            <p>{isEditing ? initialData.name : dataUrl}</p>
        </div>
        <button type="button" class="btn btn-ghost btn-icon" aria-label="Close" onclick={() => formSubmitted(false)}>
            <Icon name="close" size={16} />
        </button>
    </div>

    <div class="sheet-body">
        {@render children()}
    </div>

    <div class="sheet-footer">
        {#if isEditing}
            <button class="btn btn-ghost btn-danger" type="button" disabled={busy} onclick={handleDelete}>
                <Icon name="delete-bin-6" size={14} /> Drop
            </button>
        {/if}
        <span class="spacer"></span>
        <button class="btn" type="button" onclick={() => formSubmitted(false)}>Cancel</button>
        <button class="btn btn-primary" type="submit" disabled={isButtonDisabled || busy}>
            {isEditing ? "Save changes" : `Create ${entityName}`}
        </button>
    </div>
</form>
