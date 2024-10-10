<script lang="ts">
    import { router } from "../routes";
    import Table from "./Table.svelte";
    import type { Data } from "../api";

    interface Props {
        dataUrl: string;
        rowEdit: (d: Data) => void;
    }
    let { dataUrl, rowEdit }: Props = $props();
    let table: Table;

    export function refresh() {
        table.refresh();
    }

    function rowClick(d: Data) {
        const name = d.name;
        const schema = d.schema ?? "";
        if (!router.navigate(window.location.pathname + "/" + name, schema)) rowEdit(d);
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
