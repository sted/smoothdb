<script lang="ts">
    import { toasts, dismiss } from "../toast.svelte";
    import Icon from "./Icon.svelte";
</script>

<div class="toasts" role="status" aria-live="polite">
    {#each toasts as t (t.id)}
        <div class="toast {t.kind}">
            <Icon name={t.kind === "error" ? "error-warning" : "check"} size={15} />
            <div class="text">
                {t.text}
                {#if t.detail}<small class="mono">{t.detail}</small>{/if}
            </div>
            <button class="btn btn-ghost btn-sm btn-icon" aria-label="Close" onclick={() => dismiss(t.id)}>
                <Icon name="close" size={14} />
            </button>
        </div>
    {/each}
</div>

<style>
    .toasts {
        position: fixed;
        right: 16px;
        bottom: 16px;
        z-index: 200;
        display: flex;
        flex-direction: column;
        gap: 8px;
        max-width: min(420px, calc(100vw - 32px));
    }
    .toast {
        display: flex;
        align-items: flex-start;
        gap: 8px;
        padding: 10px 6px 10px 12px;
        border: 1px solid var(--border);
        border-radius: var(--radius-lg);
        background: var(--bg);
        box-shadow: var(--shadow-pop);
        font-size: 12.5px;
    }
    .toast > :global(.icon) {
        margin-top: 1px;
        color: var(--success);
    }
    .toast.error > :global(.icon) {
        color: var(--danger);
    }
    .text {
        flex: 1;
        overflow-wrap: anywhere;
    }
    small {
        display: block;
        margin-top: 2px;
        color: var(--text-3);
        white-space: pre-wrap;
    }
</style>
