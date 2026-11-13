<script lang="ts">
    import type { Snippet } from "svelte";
    import { fade, fly } from "svelte/transition";
    import { router } from "../routes";

    interface Props {
        formSubmitted: (refreshTable: boolean) => void;
        // the form for the row; without it, the form of the current route
        form?: Snippet<[any]>;
    }
    let { formSubmitted, form }: Props = $props();

    let openModal = $state(false);
    let data: any = $state();

    export function open(d: any) {
        openModal = true;
        data = d;
    }
    export function close() {
        openModal = false;
    }

    function onKey(ev: KeyboardEvent) {
        if (openModal && ev.key === "Escape") close();
    }
</script>

<svelte:window onkeydown={onKey} />

{#if openModal}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="overlay" onclick={close} transition:fade={{ duration: 120 }}></div>
    <div class="sheet" role="dialog" aria-modal="true" transition:fly={{ x: 32, duration: 160 }}>
        {#if form}
            {@render form(data)}
        {:else}
            <router.component {data} {formSubmitted}></router.component>
        {/if}
    </div>
{/if}

<style>
    .overlay {
        position: fixed;
        inset: 0;
        z-index: 50;
        background: var(--bg-overlay);
    }
    .sheet {
        position: fixed;
        top: 0;
        right: 0;
        z-index: 100;
        width: min(500px, calc(100vw - 48px));
        height: 100%;
        background: var(--bg);
        border-left: 1px solid var(--border);
        box-shadow: var(--shadow-sheet);
    }
</style>
