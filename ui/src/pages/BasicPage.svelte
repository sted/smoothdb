<script lang="ts">
    import DataPage from "./DataPage.svelte";
    import AdminPage from "../admin/AdminPage.svelte";
    import { router } from "../routes";

    let isDataRoute = $derived(router.path === "/ui/data" || router.path.startsWith("/ui/data/"));
</script>

{#if isDataRoute}
    <DataPage />
{:else}
    <AdminPage />
{/if}
