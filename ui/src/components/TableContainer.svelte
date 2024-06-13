<script lang="ts">
    import { router } from "../routes";
    import Table from "./Table.svelte";

    interface Props {
        dataUrl: string;
        rowEdit: (e: CustomEvent) => void;
    }
    let { dataUrl, rowEdit }: Props = $props();
    let table: Table;

    export function refresh() {
        table.refresh();
    }

    function rowClick(name: string) {
        router.navigate(window.location.pathname + "/" + name);
    }
</script>

<div class="table-container">
    <Table bind:this={table} {dataUrl} {rowClick} {rowEdit} />
</div>

<style>
    .table-container {
        height: 100%;
        overflow: auto;
    }
</style>
