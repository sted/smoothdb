<script lang="ts">
    import { onMount } from "svelte";
    import { ui } from "../config.svelte";
    import { databaseList, listDatabases } from "../data.svelte";
    import { router } from "../routes";
    import Icon from "./Icon.svelte";

    // the list is shared: an admin change of the databases refreshes it
    let databases = $derived(databaseList.items);
    let search = $state("");
    let inData = $derived(router.path === "/ui/data" || router.path.startsWith("/ui/data/"));
    // The section of the current route is open; the user can toggle both.
    let dataExpanded = $state(false);
    let adminExpanded = $state(false);

    $effect(() => {
        dataExpanded = inData;
        adminExpanded = !inData;
    });

    onMount(async () => {
        databaseList.items = await listDatabases();
    });

    // the search box appears when the list is long
    let shown = $derived(
        search ? databases.filter((d) => d.name.toLowerCase().includes(search.toLowerCase())) : databases,
    );

    const dataHref = (db: string) => `/ui/data/${encodeURIComponent(db)}`;
    const isActive = (href: string) => router.path === href || router.path.startsWith(href + "/");
</script>

<aside>
    <nav>
        <button class="section" aria-expanded={dataExpanded} onclick={() => (dataExpanded = !dataExpanded)}>
            <Icon name={dataExpanded ? "arrow-down-s" : "arrow-right-s"} size={14} />
            Data
        </button>
        {#if dataExpanded}
            {#if databases.length > 8}
                <div class="search">
                    <Icon name="search" size={13} />
                    <input class="input input-sm" placeholder="Find a database" bind:value={search} />
                </div>
            {/if}
            <ul>
                {#each shown as database}
                    {@const href = dataHref(database.name)}
                    <li>
                        <a class="item" class:active={isActive(href)} {href}>
                            <Icon name="database-2" size={14} />
                            <span class="name">{database.name}</span>
                        </a>
                    </li>
                {/each}
            </ul>
        {/if}

        {#if ui.admin}
            <button class="section" aria-expanded={adminExpanded} onclick={() => (adminExpanded = !adminExpanded)}>
                <Icon name={adminExpanded ? "arrow-down-s" : "arrow-right-s"} size={14} />
                Admin
            </button>
            {#if adminExpanded}
                <ul>
                    <li>
                        <a class="item" class:active={router.path.startsWith("/ui/databases")} href="/ui/databases">
                            <Icon name="stack" size={14} /> Databases
                        </a>
                    </li>
                    <li>
                        <a class="item" class:active={router.path.startsWith("/ui/roles")} href="/ui/roles">
                            <Icon name="group" size={14} /> Roles
                        </a>
                    </li>
                </ul>
            {/if}
        {/if}
    </nav>
</aside>

<style>
    aside {
        height: 100%;
        overflow-y: auto;
        padding: 10px 8px;
    }
    .section {
        display: flex;
        align-items: center;
        gap: 4px;
        width: 100%;
        height: 26px;
        margin-top: 6px;
        padding: 0 4px;
        border: none;
        border-radius: var(--radius-sm);
        background: none;
        color: var(--text-3);
        font-size: 11.5px;
        font-weight: 600;
        cursor: pointer;
    }
    .section:first-child {
        margin-top: 0;
    }
    .section:hover {
        color: var(--text-2);
    }
    .search {
        position: relative;
        margin: 2px 0 4px;
        color: var(--text-3);
    }
    .search :global(.icon) {
        position: absolute;
        left: 7px;
        top: 6px;
    }
    .search .input {
        padding-left: 24px;
    }
    ul {
        list-style: none;
        display: flex;
        flex-direction: column;
        gap: 1px;
    }
    .item {
        display: flex;
        align-items: center;
        gap: 8px;
        height: 28px;
        padding: 0 8px;
        border-radius: var(--radius);
        color: var(--text-2);
    }
    .item :global(.icon) {
        color: var(--text-3);
    }
    .item:hover {
        background: var(--bg-hover);
        color: var(--text);
    }
    .item.active {
        background: var(--bg-active);
        color: var(--text);
        font-weight: 500;
    }
    .item.active :global(.icon) {
        color: var(--accent);
    }
    .name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
</style>
