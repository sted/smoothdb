<script lang="ts">
    import TableContainer from "../components/TableContainer.svelte";
    import ModalPanel from "../components/ModalPanel.svelte";
    import RoleForm from "../forms/RoleForm.svelte";
</script>

<TableContainer />
<ModalPanel>
    <RoleForm slot="form"></RoleForm>
</ModalPanel>
