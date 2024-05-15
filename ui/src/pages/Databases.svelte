<script lang="ts">
    import TableContainer from "../components/TableContainer.svelte";
    import ModalPanel from "../components/ModalPanel.svelte";
    import DatabaseForm from "../forms/DatabaseForm.svelte";

    let suffixNextUrl = "/tables";
</script>

<TableContainer {suffixNextUrl} />
<ModalPanel>
    <DatabaseForm slot="form"></DatabaseForm>
</ModalPanel>
