<script lang="ts">
    import { slide } from "svelte/transition";
    import RiCloseLine from "svelte-remixicon/RiCloseLine.svelte";

    interface Props {
        form: any;
        formSubmitted: (refreshTable: boolean) => void;
    }
    let { form, formSubmitted }: Props = $props();

    let openModal = $state(false);
    let data: any = $state();

    export function open(d: any) {
        openModal = true;
        data = d;
    }
    export function close() {
        openModal = false;
    }
</script>

{#if openModal}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="overlay" onclick={close}></div>
    <div class="panel" transition:slide={{ duration: 200, axis: "x" }}>
        <button class="close-btn" onclick={close}><RiCloseLine /></button>
        <div class="content">
            <svelte:component this={form} {data} {formSubmitted} />
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
    .panel {
        display: flex;
        flex-direction: column;
        overflow: hidden;
        position: fixed;
        right: 0;
        top: 0;
        width: 30%;
        max-width: 400px;
        height: 100%;
        padding: 30px;
        background: white;
        z-index: 100;
        box-shadow: -2px 0 8px rgba(0, 0, 0, 0.2);
    }
    .content {
        display: flex;
        flex-direction: column;
        flex: 1;
        overflow: hidden;
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
