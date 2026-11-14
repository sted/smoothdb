<script lang="ts">
    import { untrack } from "svelte";
    import { tableUrl, ui } from "../config.svelte";
    import { makeRequest } from "../api";
    import { references, tableDef, viewColumns, type Column, type ForeignKey, type Reference, type TableDef } from "../data.svelte";
    import { formatCell } from "../utils";
    import { displayValue } from "../rowvalues";
    import Grid, { type GridColumn } from "../components/Grid.svelte";
    import ColumnPicker from "../components/ColumnPicker.svelte";
    import {
        buildSelect,
        cellValue,
        columnKey,
        columnLabel,
        fkAlias,
        readColumns,
        rollupHref,
        writeColumns,
        type ColumnSpec,
    } from "../columns";
    import FilterBar, { type FilterColumn } from "../components/FilterBar.svelte";
    import { readFilters, writeFilters, type FilterState } from "../filters";
    import { columnKind, inferKind, isNumeric, type Kind } from "../coltypes";
    import RecordPanel, { type PanelTarget, type RecordRef } from "../components/RecordPanel.svelte";
    import TablesBrowser from "../components/TablesBrowser.svelte";
    import Icon from "../components/Icon.svelte";
    import ExplainPanel from "../components/ExplainPanel.svelte";
    import { router } from "../routes";

    const pageSize = 50;

    let db = $derived(router.params.db ?? "");
    let schema = $derived(router.params.schema ?? "");
    let table = $derived(router.params.table ?? "");

    let query = $derived(new URLSearchParams(router.query));
    let page = $derived(Math.max(1, Number(query.get("page")) || 1));

    let def = $state<TableDef | null>(null);
    // the label column of the table each foreign key references
    let fkLabels: Record<string, string> = $state({});
    // the tables the foreign keys point to, and the tables that point here
    let targets: Record<string, TableDef | null> = $state({});
    let refs: { ref: Reference; def: TableDef | null }[] = $state([]);
    let defLoaded = $state(false);
    let rows: any[] = $state([]);
    let total: number | null = $state(null);
    let loading = $state(false);
    // the record panel: what it shows, and whether it edits
    let panelTarget = $state<PanelTarget | null>(null);
    let editing = $state(false);
    let focusField = $state<string | undefined>(undefined);
    let panel: RecordPanel | undefined = $state();
    // the plan of the grid's request, beside the grid in place of the record panel
    let explaining = $state(false);

    // the default order is the primary key, so that pages are stable
    let order = $derived(query.get("order") ?? (def?.primaryKey.map((k) => `${k}.asc`).join(",") || ""));
    let specs: ColumnSpec[] = $derived(readColumns(query, def));
    // the kinds of the columns of a view, guessed from the rows on screen; a
    // kind found once stays while the view is open, so that an empty page keeps it
    let viewKinds: Record<string, Kind> = $state({});
    // the columns of a view with their types, when the admin API gives them
    let viewCols = $state<Column[] | null>(null);
    $effect(() => {
        if (def || rows.length === 0 || viewCols) return;
        // read without tracking: this effect writes viewKinds
        const next = { ...untrack(() => viewKinds) };
        for (const name of Object.keys(rows[0])) {
            const k = inferKind(rows.map((r) => r[name]));
            if (k !== "any" || !next[name]) next[name] = k;
        }
        viewKinds = next;
    });
    const kindOfColumn = (name: string): Kind =>
        def ? columnKind(def.columns.find((c) => c.name === name)?.type) : (viewKinds[name] ?? "any");

    // a view with known columns takes its kinds from the types
    $effect(() => {
        if (viewCols) viewKinds = Object.fromEntries(viewCols.map((c) => [c.name, columnKind(c.type)]));
    });

    // the columns the filters offer, with their types
    let filterColumns: FilterColumn[] = $derived(
        def
            ? def.columns.map((c) => ({ name: c.name, type: c.type, notnull: c.notnull }))
            : viewCols
              ? viewCols.map((c) => ({ name: c.name, type: c.type }))
              : Object.keys(viewKinds).map((name) => ({ name, kind: viewKinds[name] })),
    );
    let filterState: FilterState = $derived(readFilters(query, kindOfColumn));

    $effect(() => {
        loadDef(db, schema, table);
    });

    $effect(() => {
        // reload on any change of table, page, order or filters
        if (defLoaded) fetchRows(db, schema, table, router.query, order);
    });

    async function loadDef(db: string, schema: string, table: string) {
        defLoaded = false;
        def = null;
        fkLabels = {};
        viewKinds = {};
        viewCols = null;
        targets = {};
        refs = [];
        panelTarget = null;
        editing = false;
        explaining = false;
        rows = [];
        total = null;
        if (!db || !table) return;
        const d = await tableDef(db, schema, table);
        const labels: Record<string, string> = {};
        const tgts: Record<string, TableDef | null> = {};
        for (const fk of d?.foreignKeys ?? []) {
            if (fk.schema !== schema) continue; // embedding stays in one schema
            const target = await tableDef(db, fk.schema, fk.table);
            tgts[fk.column] = target;
            if (target?.labelColumn) labels[fk.column] = target.labelColumn;
        }
        if (db !== router.params.db || table !== router.params.table) return;
        def = d;
        fkLabels = labels;
        targets = tgts;
        defLoaded = true;
        if (!d) {
            const cols = await viewColumns(db, schema, table);
            if (table === router.params.table) viewCols = cols;
        }
        if (d) {
            // the tables that point here, for the rollups; read after the rows
            const found = (await references(db, schema, table)).filter((r) => r.schema === schema);
            const withDefs = await Promise.all(found.map(async (ref) => ({ ref, def: await tableDef(db, ref.schema, ref.table) })));
            if (table === router.params.table) refs = withDefs;
        }
    }

    export async function refresh() {
        await fetchRows(db, schema, table, router.query, order);
    }

    // the query string of the grid's request, from the state of the page
    function rowsParams(q: string, order: string): URLSearchParams {
        const params = new URLSearchParams(q);
        params.delete("page");
        params.delete("order");
        params.delete("cols");
        if (def) params.set("select", buildSelect(def, specs, fkLabels));
        if (order) params.set("order", order);
        params.set("limit", String(pageSize));
        params.set("offset", String((page - 1) * pageSize));
        return params;
    }

    async function fetchRows(db: string, schema: string, table: string, q: string, order: string) {
        const params = rowsParams(q, order);
        loading = true;
        const response = await makeRequest(`${tableUrl(db, table)}?${params}`, {
            headers: { "Accept-Profile": schema, Prefer: "count=exact" },
        });
        if (table !== router.params.table) return;
        loading = false;
        rows = response.success && Array.isArray(response.data) ? response.data : [];
        const m = response.headers?.get("Content-Range")?.match(/\/(\d+)$/);
        total = m ? Number(m[1]) : null;
    }

    function setParams(change: (p: URLSearchParams) => void) {
        const p = new URLSearchParams(router.query);
        change(p);
        router.setQuery(p);
    }

    // a click on a header sorts by it, a second click reverses the order
    function sortBy(column: string) {
        const current = order.split(",")[0];
        const next = current === `${column}.asc` ? `${column}.desc` : `${column}.asc`;
        setParams((p) => {
            p.set("order", next);
            p.delete("page");
        });
    }

    function setColumns(next: ColumnSpec[] | null) {
        setParams((p) => {
            writeColumns(p, next);
            // an order on a column that is gone falls back to the key
            const o = p.get("order");
            if (o && next && !next.some((s) => o.startsWith(columnKey(s) + "."))) p.delete("order");
        });
    }

    function setFilters(next: FilterState) {
        setParams((p) => writeFilters(p, next));
    }

    function goTo(n: number) {
        setParams((p) => (n > 1 ? p.set("page", String(n)) : p.delete("page")));
    }

    let lastPage = $derived(total === null ? page : Math.max(1, Math.ceil(total / pageSize)));
    let from = $derived(rows.length ? (page - 1) * pageSize + 1 : 0);
    let to = $derived((page - 1) * pageSize + rows.length);

    // show sets what the panel shows; unsaved changes in the panel are confirmed first
    async function show(next: PanelTarget | null, edit = false, focus?: string) {
        if (panel && !(await panel.confirmLeave())) return;
        panelTarget = next;
        if (next) explaining = false;
        editing = edit || next?.kind === "new";
        focusField = focus;
    }

    export function rowAdd() {
        if (def) show({ kind: "new", schema, table });
    }

    // the record of a row of the grid, by its single-column primary key
    function rowRecord(row: Record<string, any>): PanelTarget | null {
        const k = def?.primaryKey;
        return k?.length === 1 ? { kind: "row", record: { schema, table, column: k[0], value: row[k[0]] } } : null;
    }

    function openRow(row: Record<string, any>) {
        const t = rowRecord(row);
        if (t) show(t);
    }

    function editRow(row: Record<string, any>, focus?: string) {
        const t = rowRecord(row);
        if (t) show(t, true, focus);
    }

    let currentRecord = $derived(panelTarget?.kind === "row" ? panelTarget.record : null);

    let fkByColumn = $derived(new Map<string, ForeignKey>((def?.foreignKeys ?? []).map((f) => [f.column, f])));
    let specByKey = $derived(new Map(specs.map((x) => [columnKey(x), x])));

    // numbers align right; a foreign key shows the name of the row it points to
    function isNumberColumn(x: ColumnSpec): boolean {
        if (x.kind === "own") return isNumeric(columnKind(def?.columns.find((c) => c.name === x.name)?.type)) && !fkByColumn.has(x.name);
        if (x.kind === "lookup") return isNumeric(columnKind(targets[x.fk]?.columns.find((c) => c.name === x.target)?.type));
        return x.fn !== "max";
    }

    let gridColumns: GridColumn[] | undefined = $derived(
        def
            ? specs.map((x) => ({
                  key: columnKey(x),
                  label: columnLabel(x, fkByColumn),
                  numeric: isNumberColumn(x),
                  // PostgREST cannot order by an aggregate of an embed
                  sortable: x.kind !== "rollup",
                  related: x.kind !== "own",
              }))
            : Object.keys(viewKinds).length
              ? Object.keys(viewKinds).map((key) => ({ key, numeric: isNumeric(viewKinds[key]), sortable: true }))
              : undefined,
    );

    const refOf = (x: ColumnSpec) =>
        x.kind === "rollup" ? refs.find((r) => r.ref.table === x.table && r.ref.column === x.column)?.ref : undefined;
    const isSelected = (row: Record<string, any>) =>
        !!currentRecord &&
        currentRecord.schema === schema &&
        currentRecord.table === table &&
        row[currentRecord.column] === currentRecord.value;

</script>

<div class="data-page-layout">
    {#if db}
        <TablesBrowser dbName={db} currentSchema={schema} currentTable={table} />

        <div class="content-pane">
            {#if table}
                <div class="table-header">
                    <Icon name={defLoaded && !def ? "eye" : "table"} size={16} />
                    <h1 class="mono">{table}</h1>
                    {#if total !== null}<span class="badge">{total.toLocaleString()} {total === 1 ? "row" : "rows"}</span>{/if}
                    {#if defLoaded && !def}<span class="badge">View · read-only</span>{/if}
                    <span class="actions">
                        {#if defLoaded}
                            <button
                                class="btn btn-sm btn-ghost"
                                class:active={explaining}
                                aria-pressed={explaining}
                                title="Show how PostgreSQL runs the request of this page"
                                onclick={async () => {
                                    if (!explaining) await show(null);
                                    explaining = !explaining;
                                }}
                            >
                                <Icon name="pulse" size={14} /> Explain
                            </button>
                        {/if}
                        {#if ui.admin}
                            <a class="btn btn-sm btn-ghost" href={`/ui/databases/${encodeURIComponent(db)}/tables/${encodeURIComponent(table)}`}>
                                <Icon name="settings-3" size={14} /> Schema
                            </a>
                        {/if}
                        {#if def}
                            <button class="btn btn-sm btn-primary" onclick={rowAdd}>
                                <Icon name="add" size={14} /> New row
                            </button>
                        {/if}
                    </span>
                </div>
                <div class="toolbar">
                    <FilterBar {filterState} columns={filterColumns} onChange={setFilters} />
                    {#if def}
                        <ColumnPicker {def} {specs} {targets} {refs} onChange={setColumns} />
                    {/if}
                    <span class="pager">
                        {#if total !== null}
                            <span class="range">{from.toLocaleString()}–{to.toLocaleString()} of {total.toLocaleString()}</span>
                        {/if}
                        <button class="btn btn-sm btn-icon" aria-label="Previous page" disabled={page <= 1} onclick={() => goTo(page - 1)}>
                            <Icon name="arrow-left-s" size={16} />
                        </button>
                        <button class="btn btn-sm btn-icon" aria-label="Next page" disabled={page >= lastPage} onclick={() => goTo(page + 1)}>
                            <Icon name="arrow-right-s" size={16} />
                        </button>
                    </span>
                </div>
                <div class="grid-and-panel">
                    <div class="grid-wrap">
                        <Grid
                            {rows}
                            {loading}
                            columns={gridColumns}
                            {order}
                            {isSelected}
                            onSort={sortBy}
                            onEdit={def ? (row) => editRow(row) : undefined}
                            onCellDoubleClick={def ? (row, key) => (specByKey.get(key)?.kind === "own" ? editRow(row, key) : null) : undefined}
                            onRowClick={def?.primaryKey.length === 1 ? openRow : undefined}
                        >
                            {#snippet cell(row, key)}
                                {@const x = specByKey.get(key)}
                                {@const fk = x?.kind === "own" ? fkByColumn.get(x.name) : undefined}
                                {#if x && x.kind !== "own"}
                                    {@const v = cellValue(row, x)}
                                    {@const ref = refOf(x)}
                                    {#if v === null || v === undefined}
                                        <span class="cell-null">NULL</span>
                                    {:else if ref}
                                        <a class="rollup" href={rollupHref(db, ref, row)}>{formatCell(v)}</a>
                                    {:else}
                                        {@const text = displayValue(v, x.kind === "lookup" ? columnKind(targets[x.fk]?.columns.find((c) => c.name === x.target)?.type) : "any")}
                                        <span title={text}>{text}</span>
                                    {/if}
                                {:else if fk && row[key] !== null}
                                    <button
                                        class="fk"
                                        title={`${fk.table}.${fk.refColumn} = ${formatCell(row[key])}`}
                                        onclick={(ev) => {
                                            ev.stopPropagation();
                                            show({ kind: "row", record: { schema: fk.schema, table: fk.table, column: fk.refColumn, value: row[key] } });
                                        }}
                                        >{row[fkAlias(key)]?.[fkLabels[key]] ?? formatCell(row[key])}</button
                                    >
                                {:else if row[key] === null || row[key] === undefined}
                                    <span class="cell-null">NULL</span>
                                {:else}
                                    {@const text = displayValue(row[key], kindOfColumn(key))}
                                    <span title={text}>{text}</span>
                                {/if}
                            {/snippet}
                        </Grid>
                    </div>
                    {#if explaining}
                        <ExplainPanel {db} {schema} {table} params={rowsParams(router.query, order).toString()} onClose={() => (explaining = false)} />
                    {:else if panelTarget}
                        <RecordPanel
                            bind:this={panel}
                            {db}
                            target={panelTarget}
                            {editing}
                            focus={focusField}
                            onOpen={(r) => {
                                // a save of a new row opens it; the panel has nothing left to confirm
                                panelTarget = { kind: "row", record: r };
                                editing = false;
                            }}
                            onEditing={(on, focus) => {
                                editing = on;
                                focusField = focus;
                            }}
                            onChanged={refresh}
                            onClose={() => {
                                panelTarget = null;
                                editing = false;
                            }}
                        />
                    {/if}
                </div>
            {:else}
                <div class="placeholder">
                    <Icon name="database-2" size={28} />
                    <h2 class="mono">{db}</h2>
                    <p>Choose a table or a view on the left.</p>
                </div>
            {/if}
        </div>
    {:else}
        <div class="placeholder">
            <Icon name="database-2" size={28} />
            <h2>Data</h2>
            <p>Choose a database on the left.</p>
        </div>
    {/if}
</div>

<style>
    .data-page-layout {
        display: flex;
        height: 100%;
        overflow: hidden;
    }
    .content-pane {
        flex: 1;
        min-width: 0;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }
    .table-header {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 12px 14px 8px;
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
    .toolbar {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 0 14px 10px;
    }
    .pager {
        margin-left: auto;
        display: flex;
        align-items: center;
        gap: 4px;
        white-space: nowrap;
    }
    .range {
        margin-right: 6px;
        color: var(--text-3);
        font-size: 12px;
        font-variant-numeric: tabular-nums;
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
    .fk {
        max-width: 100%;
        overflow: hidden;
        text-overflow: ellipsis;
        padding: 0;
        border: none;
        background: transparent;
        color: var(--accent-text);
        cursor: pointer;
    }
    .fk:hover {
        text-decoration: underline;
        text-underline-offset: 2px;
    }
    .rollup {
        color: var(--accent-text);
    }
    .btn.active {
        background: var(--accent-soft);
        color: var(--accent-text);
    }
    .rollup:hover {
        text-decoration: underline;
        text-underline-offset: 2px;
    }
    .placeholder {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 6px;
        height: 100%;
        color: var(--text-3);
    }
    .placeholder h2 {
        color: var(--text);
        font-size: 15px;
        font-weight: 600;
    }
</style>
