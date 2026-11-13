<script lang="ts">
    import type { Snippet } from "svelte";
    import { dialog } from "../dialog.svelte";
    import Icon from "./Icon.svelte";

    // The frame of every side panel of the UI, for a row of the Data area and
    // for an object of the admin area: a header with Edit and Close, the body,
    // and in editing mode a footer with Delete, the count of changes, Cancel
    // and Save. Cmd+Enter saves, Escape cancels.
    interface Props {
        icon?: string;
        title: string;
        subtitle?: string;
        editing: boolean;
        isNew?: boolean;
        canEdit?: boolean;
        canDelete?: boolean;
        busy?: boolean;
        error?: string;
        changed?: number;
        // the label of the save button
        saveLabel?: string;
        onEdit?: () => void;
        onCancel: () => void;
        onSave: () => void;
        onDelete?: () => void;
        onClose: () => void;
        // links or buttons next to Edit
        actions?: Snippet;
        children: Snippet;
    }
    let {
        icon = "table",
        title,
        subtitle = "",
        editing,
        isNew = false,
        canEdit = true,
        canDelete = true,
        busy = false,
        error = "",
        changed = 0,
        saveLabel,
        onEdit,
        onCancel,
        onSave,
        onDelete,
        onClose,
        actions,
        children,
    }: Props = $props();

    function onKey(ev: KeyboardEvent) {
        if (!editing || dialog.current) return;
        if (ev.key === "Enter" && (ev.metaKey || ev.ctrlKey)) {
            ev.preventDefault();
            onSave();
        } else if (ev.key === "Escape") onCancel();
    }
</script>

<svelte:window onkeydown={onKey} />

<aside class="side-panel" aria-label={`${title} ${subtitle}`}>
    <header>
        <Icon name={icon} size={14} />
        <span class="mono title">{title}</span>
        {#if subtitle}<span class="mono subtitle">{subtitle}</span>{/if}
        <span class="tools">
            {#if !editing}
                {@render actions?.()}
                {#if canEdit && onEdit}
                    <button class="btn btn-sm" onclick={onEdit}><Icon name="edit" size={13} /> Edit</button>
                {/if}
            {/if}
            <button class="btn btn-sm btn-ghost btn-icon" aria-label="Close" onclick={onClose}>
                <Icon name="close" size={15} />
            </button>
        </span>
    </header>

    <form
        class="body"
        onsubmit={(ev) => {
            ev.preventDefault();
            onSave();
        }}
    >
        {#if error}
            <div class="banner banner-danger" role="alert">
                <Icon name="error-warning" size={15} />
                {error}
            </div>
        {/if}
        {@render children()}
    </form>

    {#if editing}
        <footer>
            {#if !isNew && canDelete && onDelete}
                <button type="button" class="btn btn-sm btn-ghost btn-danger" disabled={busy} onclick={onDelete}>
                    <Icon name="delete-bin-6" size={13} /> Delete
                </button>
            {/if}
            <span class="spacer"></span>
            <span class="hint">
                {#if changed}{changed} {changed === 1 ? "change" : "changes"}{:else}<span class="kbd">⌘</span><span class="kbd">↵</span>{/if}
            </span>
            <button type="button" class="btn btn-sm" onclick={onCancel}>Cancel</button>
            <button type="button" class="btn btn-sm btn-primary" disabled={busy || (!isNew && changed === 0)} onclick={onSave}>
                {busy ? "Saving..." : (saveLabel ?? (isNew ? "Create" : "Save"))}
            </button>
        </footer>
    {/if}
</aside>

<style>
    .side-panel {
        display: flex;
        flex-direction: column;
        width: 400px;
        flex-shrink: 0;
        border-left: 1px solid var(--border);
        background: var(--bg);
        font-size: 12.5px;
    }
    header {
        display: flex;
        align-items: center;
        gap: 6px;
        height: 40px;
        padding: 0 6px 0 12px;
        border-bottom: 1px solid var(--border);
        color: var(--text-3);
    }
    .title {
        color: var(--text);
        font-weight: 600;
        white-space: nowrap;
    }
    .subtitle {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: var(--text-3);
    }
    .tools {
        margin-left: auto;
        display: flex;
        align-items: center;
        gap: 4px;
    }
    .body {
        flex: 1;
        overflow: auto;
        padding: 10px 12px 16px;
    }
    .banner {
        margin-bottom: 10px;
    }
    footer {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 8px 10px;
        border-top: 1px solid var(--border);
        background: var(--bg-subtle);
    }
    .spacer {
        flex: 1;
    }
    .hint {
        display: inline-flex;
        align-items: center;
        gap: 3px;
        color: var(--text-3);
        font-size: 11.5px;
    }
</style>
