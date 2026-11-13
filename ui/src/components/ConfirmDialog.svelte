<script lang="ts">
    import { fade, scale } from "svelte/transition";
    import { closeDialog, dialog } from "../dialog.svelte";
    import Icon from "./Icon.svelte";

    let confirmBtn: HTMLButtonElement | undefined = $state();
    let cancelBtn: HTMLButtonElement | undefined = $state();

    $effect(() => {
        if (dialog.current) (dialog.current.danger ? cancelBtn : confirmBtn)?.focus();
    });

    function onKey(ev: KeyboardEvent) {
        if (!dialog.current) return;
        if (ev.key === "Escape") {
            ev.preventDefault();
            ev.stopImmediatePropagation();
            closeDialog(false);
        }
    }
</script>

<svelte:window onkeydowncapture={onKey} />

{#if dialog.current}
    {@const d = dialog.current}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="overlay" onclick={() => closeDialog(false)} transition:fade={{ duration: 100 }}></div>
    <div
        class="dialog"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="dlg-title"
        aria-describedby={d.message ? "dlg-msg" : undefined}
        transition:scale={{ start: 0.97, duration: 120 }}
    >
        <div class="head">
            {#if d.danger}<span class="icon danger"><Icon name="error-warning" size={16} /></span>{/if}
            <h2 id="dlg-title">{d.title}</h2>
        </div>
        {#if d.message}<p id="dlg-msg">{d.message}</p>{/if}
        <div class="actions">
            <button class="btn" bind:this={cancelBtn} onclick={() => closeDialog(false)}>{d.cancelLabel ?? "Cancel"}</button>
            <button
                class="btn"
                class:btn-primary={!d.danger}
                class:btn-destructive={d.danger}
                bind:this={confirmBtn}
                onclick={() => closeDialog(true)}>{d.confirmLabel ?? "OK"}</button
            >
        </div>
    </div>
{/if}

<style>
    .overlay {
        position: fixed;
        inset: 0;
        z-index: 300;
        background: var(--bg-overlay);
    }
    .dialog {
        position: fixed;
        top: 24vh;
        left: 50%;
        z-index: 301;
        width: min(400px, calc(100vw - 32px));
        margin-left: calc(min(400px, calc(100vw - 32px)) / -2);
        display: flex;
        flex-direction: column;
        gap: 10px;
        padding: 18px 18px 14px;
        border: 1px solid var(--border);
        border-radius: var(--radius-lg);
        background: var(--bg);
        box-shadow: var(--shadow-pop);
    }
    .head {
        display: flex;
        align-items: center;
        gap: 10px;
    }
    .icon {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 28px;
        height: 28px;
        border-radius: 7px;
    }
    .icon.danger {
        background: var(--danger-soft);
        color: var(--danger);
    }
    h2 {
        font-size: 14px;
        font-weight: 600;
    }
    p {
        color: var(--text-2);
        line-height: 1.5;
    }
    .actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        margin-top: 6px;
    }
    .btn-destructive {
        border-color: var(--danger);
        background: var(--danger);
        color: #fff;
    }
    .btn-destructive:hover {
        filter: brightness(0.92);
        background: var(--danger);
    }
</style>
