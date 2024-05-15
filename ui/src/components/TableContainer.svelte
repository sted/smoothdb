<script lang="ts">
    import { adminDbUrl } from "../main";
    import { router } from "../routes";
    import { modalStore } from "../stores";
    import Table from "./Table.svelte";

    export let suffixNextUrl = "";

    $: dbUrl = adminDbUrl + window.location.pathname.replace(/^\/ui/, "");

    interface TableRowData {
        [key: string]: any;
    }

    function handleRowClick(event: CustomEvent<TableRowData>) {
        if (suffixNextUrl !== "") {
            router.navigate(window.location.pathname + "/" + event.detail.name + suffixNextUrl);
        }
    }

    function handleRowEdit(event: CustomEvent<TableRowData>) {
        modalStore.set({ showModal: true, data: event.detail });
    }
</script>

<div class="table-container">
    <Table url={dbUrl} on:rowClick={handleRowClick} on:rowEdit={handleRowEdit} />
</div>

<style>
    .table-container {
        height: 100%;
        overflow: auto;
    }
</style>
