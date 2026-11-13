<script lang="ts">
    import { listRelations, type Relation } from "../data.svelte";
    import Icon from "./Icon.svelte";

    interface Props {
        dbName: string;
        currentSchema?: string;
        currentTable?: string;
    }
    let { dbName, currentSchema = "", currentTable = "" }: Props = $props();

    let tables: Relation[] = $state([]);
    let views: Relation[] = $state([]);
    let loading = $state(false);
    let tablesExpanded = $state(true);
    let viewsExpanded = $state(true);
    let search = $state("");
    const match = (r: Relation) => !search || r.name.toLowerCase().includes(search.toLowerCase());

    $effect(() => {
        load(dbName);
    });

    async function load(db: string) {
        tables = [];
        views = [];
        if (!db) return;
        loading = true;
        const r = await listRelations(db);
        loading = false;
        if (db !== dbName) return; // a later database replaced this one
        tables = r.tables;
        views = r.views;
    }

    // two schemas can have a relation with the same name: the schema is part of the route
    const href = (r: Relation) =>
        `/ui/data/${encodeURIComponent(dbName)}/${encodeURIComponent(r.schema)}/${encodeURIComponent(r.name)}`;
    const isCurrent = (r: Relation) => r.schema === currentSchema && r.name === currentTable;
    // the schema is shown only when the database has more than one
    let multiSchema = $derived(new Set([...tables, ...views].map((r) => r.schema)).size > 1);
</script>

{#snippet section(title: string, items: Relation[], icon: string, expanded: boolean, toggle: () => void)}
    {#if items.length > 0}
        <button class="section" aria-expanded={expanded} onclick={toggle}>
            <Icon name={expanded ? "arrow-down-s" : "arrow-right-s"} size={14} />
            <span>{title}</span>
            <span class="count">{items.length}</span>
        </button>
        {#if expanded}
            <ul>
                {#each items.filter(match) as r}
                    <li>
                        <a class="item" class:active={isCurrent(r)} href={href(r)} title={`${r.schema}.${r.name}`}>
                            <Icon name={icon} size={14} />
                            <span class="name">
                                {#if multiSchema}<span class="schema">{r.schema}.</span>{/if}{r.name}
                            </span>
                        </a>
                    </li>
                {/each}
            </ul>
        {/if}
    {/if}
{/snippet}

<div class="tables-browser">
    <div class="search">
        <Icon name="search" size={13} />
        <input class="input input-sm" placeholder="Find a table" bind:value={search} />
    </div>
    {#if loading}
        <p class="empty">Loading...</p>
    {:else}
        <div class="list">
            {@render section("Tables", tables, "table", tablesExpanded, () => (tablesExpanded = !tablesExpanded))}
            {@render section("Views", views, "eye", viewsExpanded, () => (viewsExpanded = !viewsExpanded))}
            {#if tables.length === 0 && views.length === 0}
                <p class="empty">Your role sees no tables in this database.</p>
            {/if}
        </div>
    {/if}
</div>

<style>
    .tables-browser {
        display: flex;
        flex-direction: column;
        width: 220px;
        height: 100%;
        border-right: 1px solid var(--border);
        background: var(--bg-subtle);
    }
    .search {
        position: relative;
        padding: 8px 8px 4px;
        color: var(--text-3);
    }
    .search :global(.icon) {
        position: absolute;
        left: 15px;
        top: 14px;
    }
    .search .input {
        padding-left: 24px;
    }
    .list {
        flex: 1;
        overflow-y: auto;
        padding: 0 8px 10px;
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
        background: none;
        color: var(--text-3);
        font-size: 11.5px;
        font-weight: 600;
        cursor: pointer;
    }
    .section:hover {
        color: var(--text-2);
    }
    .count {
        margin-left: auto;
        font-weight: 400;
        font-variant-numeric: tabular-nums;
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
        height: 26px;
        padding: 0 8px;
        border-radius: var(--radius);
        color: var(--text-2);
        font-family: var(--font-mono);
        font-size: 12px;
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
    }
    .item.active :global(.icon) {
        color: var(--accent);
    }
    .name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .schema {
        color: var(--text-3);
    }
    .empty {
        padding: 16px 8px;
        color: var(--text-3);
        font-size: 12px;
    }
</style>
