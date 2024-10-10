<script lang="ts">
    import { adminDbUrl } from "../main";
    import TableContainer from "../components/TableContainer.svelte";
    import ModalPanel from "../components/ModalPanel.svelte";
    import { router } from "../routes";

    let dataUrl: string = $state("");

    let modal: ModalPanel;
    let table: TableContainer;

    $effect(() => {
        router.path;
        dataUrl = adminDbUrl + window.location.pathname.replace(/^\/ui/, "");
    });

    export function rowAdd() {
        modal.open({});
    }

    function rowEdit(data: any) {
        modal.open(data);
    }

    function formSubmitted(refreshTable: boolean) {
        modal.close();
        if (refreshTable) table.refresh();
    }
</script>

<TableContainer bind:this={table} {dataUrl} {rowEdit} />
<ModalPanel bind:this={modal} {formSubmitted} />
