<script lang="ts">
    import { makeRequest } from "../api";
    import { router } from "../routes";
    import Grid, { type GridColumn } from "../components/Grid.svelte";
    import Icon from "../components/Icon.svelte";
    import EntityPanel, { type EntityTarget } from "./EntityPanel.svelte";
    import { dbSections, entities, tableSections, uiPath, type AdminContext, type Item } from "./entities";

    // The page of every admin list: the kind comes from the route (meta.entity),
    // the database, schema and table from its parameters.
    let def = $derived(entities[router.meta.entity ?? "databases"] ?? entities.databases);
    let ctx: AdminContext = $derived({
        db: router.params.db ?? "",
        schema: router.params.schema ?? "",
        table: router.params.table ?? "",
    });

    let items: Item[] = $state([]);
    let loading = $state(false);
    let failed = $state("");
    let search = $state("");
    let sortKey = $state("");
    let sortDesc = $state(false);

    let target = $state<EntityTarget | null>(null);
    let editing = $state(false);
    let panel: EntityPanel | undefined = $state();

    // every load has a number: a response that arrives after a later load is dropped
    let loadSeq = 0;

    $effect(() => {
        load(def.kind, ctx);
    });

    async function load(kind: string, c: AdminContext, select?: string, selectName?: string) {
        const seq = ++loadSeq;
        loading = true;
        failed = "";
        const req = entities[kind].list(c);
        const res = await makeRequest(req.url, { headers: req.headers, quiet: true });
        if (seq !== loadSeq) return;
        loading = false;
        items = res.success && Array.isArray(res.data) ? res.data : [];
        if (!res.success) failed = res.error || "The list could not be read.";
        if (select !== undefined) {
            const byName = items.filter((i) => selectName && i.name === selectName);
            const found = items.find((i) => entities[kind].key(i) === select) ?? (byName.length === 1 ? byName[0] : undefined);
            target = found ? { kind: "item", item: found } : null;
        }
    }

    // a new route starts with no selection and no filter
    let routeKey = $derived(`${def.kind}|${ctx.db}|${ctx.schema}|${ctx.table}`);
    $effect(() => {
        routeKey;
        target = null;
        editing = false;
        search = "";
        sortKey = "";
    });

    async function show(next: EntityTarget | null, edit = false) {
        if (panel && !(await panel.confirmLeave())) return;
        target = next;
        editing = edit || next?.kind === "new";
    }

    // the text of a cell, for the filter and the sort
    function cellText(item: Item, key: string): string {
        const col = def.columns.find((c) => c.key === key);
        const v = col?.text ? col.text(item) : item[key];
        if (col?.flag) return v ? "yes" : "";
        return v === null || v === undefined ? "" : typeof v === "object" ? JSON.stringify(v) : String(v);
    }

    let shown = $derived.by(() => {
        const q = search.trim().toLowerCase();
        let out = q ? items.filter((i) => def.columns.some((c) => cellText(i, c.key).toLowerCase().includes(q))) : items;
        if (sortKey) {
            out = [...out].sort((a, b) => cellText(a, sortKey).localeCompare(cellText(b, sortKey), undefined, { numeric: true }));
            if (sortDesc) out.reverse();
        }
        return out;
    });

    function sortBy(key: string) {
        if (sortKey === key) sortDesc = !sortDesc;
        else {
            sortKey = key;
            sortDesc = false;
        }
    }

    let gridColumns: GridColumn[] = $derived(def.columns.map((c) => ({ key: c.key, label: c.label, sortable: true })));
    let order = $derived(sortKey ? `${sortKey}.${sortDesc ? "desc" : "asc"}` : "");

    const isSelected = (row: Item) => target?.kind === "item" && def.key(target.item) === def.key(row);

    // the sections of the page: of the table when there is one, else of the database
    let sections = $derived(ctx.table ? tableSections : ctx.db ? dbSections : []);
    const sectionHref = (key: string) =>
        ctx.table ? uiPath("databases", ctx.db, "tables", ctx.schema, ctx.table, key) : uiPath("databases", ctx.db, key);
    let currentSection = $derived(router.path.split("/").pop() ?? "");

    let heading = $derived(ctx.table ? ctx.table : ctx.db ? ctx.db : def.plural);
</script>

<div class="admin-page">
    <div class="content">
        <div class="page-header">
            <Icon name={ctx.table ? "table" : ctx.db ? "database-2" : def.icon} size={16} />
            <h1 class="mono">{heading}</h1>
            {#if ctx.table}<span class="badge mono">{ctx.schema}</span>{/if}
            <span class="actions">
                {#if ctx.table}
                    <a class="btn btn-sm btn-ghost" href={uiPath("data", ctx.db, ctx.schema, ctx.table)}>
                        <Icon name="table" size={14} /> Data
                    </a>
                {:else if ctx.db}
                    <a class="btn btn-sm btn-ghost" href={uiPath("data", ctx.db)}>
                        <Icon name="table" size={14} /> Data
                    </a>
                {/if}
                {#if def.create}
                    <button class="btn btn-sm btn-primary" onclick={() => show({ kind: "new" })}>
                        <Icon name="add" size={14} /> New {def.singular}
                    </button>
                {/if}
            </span>
        </div>

        {#if sections.length}
            <nav class="tabs" aria-label="Sections">
                {#each sections as [key, label]}
                    <a class="tab" class:on={currentSection === key} href={sectionHref(key)}>{label}</a>
                {/each}
            </nav>
        {/if}

        <div class="toolbar">
            <div class="search">
                <Icon name="search" size={13} />
                <input class="input input-sm" placeholder={`Find ${def.plural.toLowerCase()}`} bind:value={search} />
            </div>
            <span class="count">
                {#if !loading}{shown.length}{#if shown.length !== items.length} of {items.length}{/if} {shown.length === 1 ? def.singular : def.plural.toLowerCase()}{/if}
            </span>
        </div>

        <div class="grid-and-panel">
            <div class="grid-wrap">
                {#if failed}
                    <div class="empty">
                        <Icon name="error-warning" size={20} />
                        <p>{failed}</p>
                    </div>
                {:else if !loading && items.length === 0}
                    <div class="empty">
                        <Icon name={def.icon} size={22} />
                        <p>There are no {def.plural.toLowerCase()} here yet.</p>
                        {#if def.create}
                            <button class="btn btn-sm" onclick={() => show({ kind: "new" })}><Icon name="add" size={13} /> New {def.singular}</button>
                        {/if}
                    </div>
                {:else}
                    <Grid
                        rows={shown}
                        {loading}
                        columns={gridColumns}
                        {order}
                        onSort={sortBy}
                        {isSelected}
                        onRowClick={(row) => show({ kind: "item", item: row })}
                        onEdit={def.update ? (row) => show({ kind: "item", item: row }, true) : undefined}
                    >
                        {#snippet cell(row, key)}
                            {@const col = def.columns.find((c) => c.key === key)}
                            {@const text = cellText(row, key)}
                            {#if col?.flag}
                                {#if row[key]}<span class="badge badge-accent">{col.label}</span>{/if}
                            {:else if col?.link && def.open}
                                <a class="link" class:mono={col.mono} href={def.open(ctx, row)}>{text}</a>
                            {:else if text === ""}
                                <span class="cell-null">—</span>
                            {:else}
                                <span class:mono={col?.mono} title={text}>{text}</span>
                            {/if}
                        {/snippet}
                    </Grid>
                {/if}
            </div>
            {#if target}
                <EntityPanel
                    bind:this={panel}
                    {def}
                    {ctx}
                    {target}
                    {editing}
                    onEditing={(on) => (editing = on)}
                    onSaved={(key, name) => {
                        editing = false;
                        load(def.kind, ctx, key, name);
                    }}
                    onRemoved={() => {
                        target = null;
                        editing = false;
                        load(def.kind, ctx);
                    }}
                    onClose={() => {
                        target = null;
                        editing = false;
                    }}
                />
            {/if}
        </div>
    </div>
</div>

<style>
    .admin-page {
        display: flex;
        height: 100%;
        overflow: hidden;
    }
    .content {
        flex: 1;
        min-width: 0;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }
    .page-header {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 12px 14px 6px;
        color: var(--text-3);
    }
    h1 {
        color: var(--text);
        font-size: 15px;
        font-weight: 600;
    }
    .actions {
        margin-left: auto;
        display: flex;
        gap: 6px;
    }
    .tabs {
        display: flex;
        gap: 2px;
        padding: 0 10px;
        border-bottom: 1px solid var(--border);
    }
    .tab {
        padding: 6px 8px;
        border-bottom: 2px solid transparent;
        color: var(--text-2);
        font-size: 12.5px;
    }
    .tab:hover {
        color: var(--text);
    }
    .tab.on {
        border-bottom-color: var(--accent);
        color: var(--text);
        font-weight: 500;
    }
    .toolbar {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 8px 14px;
    }
    .search {
        position: relative;
        width: 240px;
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
    .count {
        margin-left: auto;
        color: var(--text-3);
        font-size: 12px;
    }
    .grid-and-panel {
        flex: 1;
        display: flex;
        min-height: 0;
        border-top: 1px solid var(--border);
    }
    .grid-wrap {
        flex: 1;
        min-width: 0;
        overflow: hidden;
    }
    .link {
        color: var(--accent-text);
    }
    .link:hover {
        text-decoration: underline;
        text-underline-offset: 2px;
    }
    .empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 8px;
        height: 100%;
        color: var(--text-3);
    }
</style>
