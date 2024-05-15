<script lang="ts">
    import { slide } from "svelte/transition";
    import RiCloseLine from "svelte-remixicon/RiCloseLine.svelte";
    import { modalStore } from "../stores";

    $: show = $modalStore.showModal;

    function close(): void {
        modalStore.set({ showModal: false, data: {} });
    }
</script>

<div>modal</div>
{#if show}
    <div class="overlay" on:click={close}></div>
    <div class="sidebar" transition:slide={{ duration: 200, axis: "x" }}>
        <button class="close-btn" on:click={close}><RiCloseLine size="1rem" /></button>
        <div class="content">
            <slot name="form" />
        </div>
    </div>
{/if}

<style>
    .overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.5);
        z-index: 50;
    }
    .sidebar {
        display: flex;
        flex-direction: column;
        position: fixed;
        right: 0;
        top: 0;
        width: auto;
        max-width: 100%;
        min-width: 40%;
        height: 100%;
        padding: 0 25px;
        background: white;
        z-index: 100;
        overflow: auto; /* Per gestire contenuti lunghi */
        box-shadow: -2px 0 8px rgba(0, 0, 0, 0.2);
    }
    .content {
        flex-grow: 1;
        margin-top: 10px;
    }
    .close-btn {
        position: absolute;
        top: 0;
        left: 0;
        border: none;
        background: transparent;
        /* font-size: 0.6rem; */
        cursor: pointer;
    }
</style>
