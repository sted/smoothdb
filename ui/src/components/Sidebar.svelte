<script lang="ts">
    import { router } from "../routes";
</script>

<aside>
    <ol>
        <li on:click={() => router.navigate("/")}>Home</li>
        <li on:click={() => router.navigate("/databases")}>Databases</li>
        <li on:click={() => router.navigate("/roles")}>Roles</li>
    </ol>
</aside>

<style>
    aside {
        display: flex;
        flex-direction: column;
        overflow-y: auto;
    }

    ol {
        list-style: none;
    }

    li {
        margin: 10px 0;
        cursor: pointer;
    }

    li:hover {
        text-decoration: underline;
    }
</style>
