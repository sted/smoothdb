<script lang="ts">
    import { adminDbUrl } from "../config.svelte";
    import TableContainer from "../components/TableContainer.svelte";
    import ModalPanel from "../components/ModalPanel.svelte";
    import DataPage from "./DataPage.svelte";
    import { router } from "../routes";

    let isDataRoute = $derived(router.path === "/ui/data" || router.path.startsWith("/ui/data/"));
    // the admin API mirrors the admin routes of the UI
    let dataUrl = $derived(adminDbUrl + (router.path.replace(/^\/ui/, "") || "/databases"));

    let modal: ModalPanel | undefined = $state();
    let table: TableContainer | undefined = $state();
    let dataPage: DataPage | undefined = $state();

    export function rowAdd() {
        if (isDataRoute) dataPage?.rowAdd();
        else modal?.open({});
    }

    function rowEdit(data: any) {
        modal?.open(data);
    }

    function formSubmitted(refreshTable: boolean) {
        modal?.close();
        if (refreshTable) table?.refresh();
    }
</script>

{#if isDataRoute}
    <DataPage bind:this={dataPage} />
{:else}
    <TableContainer bind:this={table} {dataUrl} {rowEdit} />
    <ModalPanel bind:this={modal} {formSubmitted} />
{/if}
