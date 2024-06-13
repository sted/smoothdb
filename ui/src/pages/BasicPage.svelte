<script lang="ts">
    import { adminDbUrl } from "../main";
    import { router } from "../routes";
    import TableContainer from "../components/TableContainer.svelte";
    import ModalPanel from "../components/ModalPanel.svelte";

    let dataUrl: string = $state("");

    let modal: ModalPanel;
    let table: TableContainer;

    router.subscribe(() => {
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
<ModalPanel bind:this={modal} form={$router.component} {formSubmitted} />
