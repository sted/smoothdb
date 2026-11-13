<script lang="ts">
    import { untrack } from "svelte";
    import { makeRequest } from "../api";
    import { tableUrl } from "../config.svelte";
    import { references, tableDef, type Column, type ForeignKey, type TableDef } from "../data.svelte";
    import { columnKind, typeLabel } from "../coltypes";
    import { displayValue, editKind, fromInput, placeholderFor, splitTemporal, toInput } from "../rowvalues";
    import { formatCell } from "../utils";
    import FieldInput from "./FieldInput.svelte";
    import { confirmDialog, dialog } from "../dialog.svelte";
    import Icon from "./Icon.svelte";

    // A row, found by the value of a unique column.
    export interface RecordRef {
        schema: string;
        table: string;
        column: string;
        value: unknown;
    }

    // What the panel shows: a row, or a new row of a table.
    export type PanelTarget = { kind: "row"; record: RecordRef } | { kind: "new"; schema: string; table: string };

    interface Props {
        db: string;
        target: PanelTarget;
        // editing mode; a new row is always edited
        editing: boolean;
        // the field that takes the focus when the editing starts
        focus?: string;
        onOpen: (record: RecordRef) => void;
        onEditing: (editing: boolean, focus?: string) => void;
        // after a save or a delete: the grid reloads
        onChanged: () => void;
        onClose: () => void;
    }
    let { db, target, editing, focus, onOpen, onEditing, onChanged, onClose }: Props = $props();

    interface Group {
        schema: string;
        table: string;
        column: string;
        count: number | null;
        rows: Record<string, any>[];
        def: TableDef | null;
        href: string;
    }

    let def = $state<TableDef | null>(null);
    let row = $state<Record<string, any> | null>(null);
    let groups: Group[] = $state([]);
    let loading = $state(true);

    // the editing state
    let values: Record<string, string> = $state({});
    let nulls: Record<string, boolean> = $state({});
    // the fields as they were when the editing started, and the time zone of
    // each timestamp; a change is a difference from the start
    let initialValues: Record<string, string> = {};
    let initialNulls: Record<string, boolean> = {};
    let zones: Record<string, string> = $state({});
    let errors: Record<string, string> = $state({});
    let error = $state("");
    let busy = $state(false);
    // the editing state belongs to the row on screen; the fields show only then
    let ready = $state(false);

    const preview = 5;
    let isNew = $derived(target.kind === "new");
    let schema = $derived(target.kind === "new" ? target.schema : target.record.schema);
    let table = $derived(target.kind === "new" ? target.table : target.record.table);

    const dataHref = (schema: string, table: string, column: string, value: unknown) =>
        `/ui/data/${encodeURIComponent(db)}/${encodeURIComponent(schema)}/${encodeURIComponent(table)}?${encodeURIComponent(column)}=eq.${encodeURIComponent(String(value))}`;

    const filter = (column: string, value: unknown) =>
        `${encodeURIComponent(column)}=eq.${encodeURIComponent(String(value))}`;

    // the total of a request with Prefer: count=exact, from Content-Range "0-4/42"
    function total(headers?: Headers): number | null {
        const m = headers?.get("Content-Range")?.match(/\/(\d+)$/);
        return m ? Number(m[1]) : null;
    }

    $effect(() => {
        load(target);
    });

    async function load(t: PanelTarget) {
        loading = true;
        groups = [];
        error = "";
        if (t.kind === "new") {
            def = await tableDef(db, t.schema, t.table);
            if (t !== target) return;
            row = {};
            loading = false;
            return;
        }
        const r = t.record;
        const [d, res] = await Promise.all([
            tableDef(db, r.schema, r.table),
            makeRequest(`${tableUrl(db, r.table)}?${filter(r.column, r.value)}&limit=1`, {
                headers: { "Accept-Profile": r.schema },
            }),
        ]);
        if (t !== target) return; // a later target replaced this one
        def = d;
        row = res.success && Array.isArray(res.data) ? (res.data[0] ?? null) : null;
        loading = false;
        if (row) loadReferences(t, row);
    }

    async function loadReferences(t: PanelTarget, current: Record<string, any>) {
        if (t.kind !== "row") return;
        const refs = await references(db, t.record.schema, t.record.table);
        const loaded = await Promise.all(
            refs.map(async (ref) => {
                const refDef = await tableDef(db, ref.schema, ref.table);
                const value = current[ref.refColumn];
                const res = await makeRequest(`${tableUrl(db, ref.table)}?${filter(ref.column, value)}&limit=${preview}`, {
                    headers: { "Accept-Profile": ref.schema, Prefer: "count=exact" },
                });
                return {
                    schema: ref.schema,
                    table: ref.table,
                    column: ref.column,
                    count: total(res.headers),
                    rows: res.success && Array.isArray(res.data) ? res.data : [],
                    def: refDef,
                    href: dataHref(ref.schema, ref.table, ref.column, value),
                };
            }),
        );
        if (t === target) groups = loaded;
    }

    // ---- editing

    let columns: Column[] = $derived(def?.columns ?? []);
    // a table without a primary key, or a view, cannot select one row to update
    let canEdit = $derived(!!def && (isNew || def.primaryKey.length > 0));

    // a key column of an existing row is not edited: the row filter uses it
    const locked = (c: Column) => c.readonly || (!isNew && !!def?.primaryKey.includes(c.name));

    function beginEdit() {
        const r = row ?? {};
        values = Object.fromEntries(columns.map((c) => [c.name, toInput(isNew ? null : r[c.name], editKind(c.type))]));
        // on a new row a column starts unset, so that its default applies
        nulls = Object.fromEntries(columns.map((c) => [c.name, !isNew && r[c.name] === null]));
        initialValues = { ...values };
        initialNulls = { ...nulls };
        zones = Object.fromEntries(columns.map((c) => [c.name, splitTemporal(r[c.name], editKind(c.type))?.zone ?? ""]));
        errors = {};
        error = "";
        ready = true;
    }

    // the editing starts when the parent asks for it, or ends; before the
    // render, so that no field shows without its value. It depends on
    // editing, loading and row only: beginEdit reads the fields it resets.
    $effect.pre(() => {
        if (editing && !loading && row) untrack(beginEdit);
        else ready = false;
    });

    function changedOf(c: Column): boolean {
        if (c.readonly) return false;
        if (isNew) return !nulls[c.name] && values[c.name] !== "";
        return values[c.name] !== initialValues[c.name] || nulls[c.name] !== initialNulls[c.name];
    }

    let changed = $derived(editing && ready ? columns.filter(changedOf).map((c) => c.name) : []);

    function validate(c: Column) {
        try {
            fromInput(values[c.name] ?? "", !!nulls[c.name], editKind(c.type), zones[c.name]);
            delete errors[c.name];
        } catch (e) {
            errors[c.name] = e instanceof Error ? e.message : String(e);
        }
    }

    // confirmLeave asks before the panel drops changes that are not saved
    export async function confirmLeave(): Promise<boolean> {
        if (!editing || changed.length === 0) return true;
        return confirmDialog({
            title: "Discard your changes?",
            message: `${changed.length} ${changed.length === 1 ? "field of this row has" : "fields of this row have"} changes that are not saved.`,
            confirmLabel: "Discard",
            cancelLabel: "Keep editing",
            danger: true,
        });
    }

    async function cancel() {
        if (!(await confirmLeave())) return;
        if (isNew) onClose();
        else onEditing(false);
    }

    const keyFilter = () =>
        (def?.primaryKey ?? []).map((k) => `${encodeURIComponent(k)}=eq.${encodeURIComponent(String(row?.[k]))}`).join("&");

    async function save(ev?: Event) {
        ev?.preventDefault();
        if (!canEdit || busy || !def) return;
        const payload: Record<string, unknown> = {};
        errors = {};
        for (const c of columns) {
            if (!changed.includes(c.name)) continue;
            try {
                payload[c.name] = fromInput(values[c.name] ?? "", !!nulls[c.name], editKind(c.type), zones[c.name]);
            } catch (e) {
                errors[c.name] = e instanceof Error ? e.message : String(e);
            }
        }
        if (Object.keys(errors).length) {
            error = "Some fields are not valid.";
            return;
        }
        if (!isNew && Object.keys(payload).length === 0) {
            onEditing(false);
            return;
        }
        const headers = { "Content-Type": "application/json", "Content-Profile": schema, Prefer: "return=representation" };
        busy = true;
        error = "";
        const res = isNew
            ? await makeRequest(tableUrl(db, table), { method: "POST", headers, body: JSON.stringify(payload), quiet: true })
            : await makeRequest(`${tableUrl(db, table)}?${keyFilter()}`, {
                  method: "PATCH",
                  headers,
                  body: JSON.stringify(payload),
                  quiet: true,
              });
        busy = false;
        if (!res.success) {
            error = res.error || "The server refused the row.";
            return;
        }
        const saved = Array.isArray(res.data) ? res.data[0] : null;
        onChanged();
        const key = def.primaryKey[0];
        if (isNew && saved && key && def.primaryKey.length === 1) {
            onOpen({ schema, table, column: key, value: saved[key] });
        } else if (isNew) {
            onClose();
        } else {
            if (saved) row = saved;
            onEditing(false);
        }
    }

    async function remove() {
        const ok = await confirmDialog({
            title: "Delete this row?",
            message: `The row ${title} of ${table} is deleted from the database. You cannot undo this.`,
            confirmLabel: "Delete row",
            danger: true,
        });
        if (!ok) return;
        busy = true;
        const res = await makeRequest(`${tableUrl(db, table)}?${keyFilter()}`, {
            method: "DELETE",
            headers: { "Content-Profile": schema, Prefer: "return=minimal" },
            quiet: true,
        });
        busy = false;
        if (!res.success) {
            error = res.error || "The server refused to delete the row.";
            return;
        }
        onChanged();
        onClose();
    }

    function onKey(ev: KeyboardEvent) {
        if (!editing || dialog.current) return;
        if (ev.key === "Enter" && (ev.metaKey || ev.ctrlKey)) save(ev);
        else if (ev.key === "Escape") cancel();
    }

    // ---- reading

    // the text that names a row: its label column, else its key
    function rowLabel(d: TableDef | null, r: Record<string, any>): string {
        const key = d?.primaryKey.map((k) => formatCell(r[k])).join(", ");
        const v = d?.labelColumn ? r[d.labelColumn] : null;
        const label = v === null || v === undefined ? "" : formatCell(v);
        return [key, label].filter(Boolean).join(" · ") || formatCell(Object.values(r)[0]);
    }

    function openRow(g: Group, r: Record<string, any>) {
        const k = g.def?.primaryKey[0];
        if (k) onOpen({ schema: g.schema, table: g.table, column: k, value: r[k] });
    }

    let fks = $derived(new Map<string, ForeignKey>((def?.foreignKeys ?? []).map((f) => [f.column, f])));
    let fields: { name: string; type?: string; column?: Column }[] = $derived(
        def ? def.columns.map((c) => ({ name: c.name, type: c.type, column: c })) : row ? Object.keys(row).map((name) => ({ name })) : [],
    );
    let title = $derived(
        target.kind === "new" ? "New row" : `${target.record.column} = ${formatCell(target.record.value)}`,
    );
</script>

<svelte:window onkeydown={onKey} />

<aside class="panel" class:editing aria-label={`${table}: ${title}`}>
    <header>
        <Icon name="table" size={14} />
        <span class="mono table">{table}</span>
        <span class="mono key">{title}</span>
        <span class="tools">
            {#if !editing && canEdit && row}
                <button class="btn btn-sm" onclick={() => onEditing(true)}>
                    <Icon name="edit" size={13} /> Edit
                </button>
            {/if}
            <button class="btn btn-sm btn-ghost btn-icon" aria-label="Close" onclick={async () => {
                if (await confirmLeave()) onClose();
            }}>
                <Icon name="close" size={15} />
            </button>
        </span>
    </header>

    {#if loading}
        <p class="muted pad">Loading...</p>
    {:else if !row}
        <p class="muted pad">The row does not exist, or your role cannot read it.</p>
    {:else}
        <form class="body" onsubmit={save}>
            {#if error}
                <div class="banner banner-danger" role="alert">
                    <Icon name="error-warning" size={15} />
                    {error}
                </div>
            {/if}
            <dl>
                {#each fields as f}
                    {@const c = f.column}
                    {@const kind = columnKind(f.type)}
                    {@const fk = fks.get(f.name)}
                    {@const isChanged = changed.includes(f.name)}
                    <div class="row" class:changed={isChanged} class:has-error={!!errors[f.name]}>
                        <dt>
                            <label for={`rf-${f.name}`} class="name">
                                {#if def?.primaryKey.includes(f.name)}<span class="key-icon" title="Primary key"><Icon name="key-2" size={11} /></span>{/if}
                                <span class="mono">{f.name}</span>
                                {#if editing && c && c.notnull && c.default === null && !locked(c)}<span class="req" title="Required">*</span>{/if}
                            </label>
                            {#if editing && f.type}<span class="type mono">{typeLabel(f.type)}</span>{/if}
                        </dt>
                        <dd
                            class:null={!editing && (row[f.name] === null || row[f.name] === undefined)}
                            ondblclick={() => {
                                if (!editing && canEdit && c && !locked(c)) onEditing(true, f.name);
                            }}
                        >
                            {#if editing && ready && c && !locked(c)}
                                <FieldInput
                                    id={`rf-${f.name}`}
                                    kind={editKind(c.type)}
                                    bind:value={values[f.name]}
                                    bind:isNull={nulls[f.name]}
                                    nullable={!c.notnull}
                                    disabled={busy}
                                    placeholder={placeholderFor(c, editKind(c.type), isNew, !!nulls[f.name])}
                                    invalid={!!errors[f.name]}
                                    autofocus={focus === f.name}
                                    zone={zones[f.name]}
                                    onblur={() => validate(c)}
                                />
                                {#if errors[f.name]}<span class="field-error">{errors[f.name]}</span>{/if}
                            {:else if isNew}
                                <span class="muted">{c?.default?.startsWith("nextval(") ? "Generated" : "Default"}</span>
                            {:else if fk && row[f.name] !== null && !editing}
                                <button
                                    type="button"
                                    class="link"
                                    onclick={() => onOpen({ schema: fk.schema, table: fk.table, column: fk.refColumn, value: row![f.name] })}
                                >
                                    {displayValue(row[f.name], kind)}
                                    <span class="target"><Icon name="arrow-right-up" size={12} />{fk.table}</span>
                                </button>
                            {:else}
                                <span class="value" class:mono={kind === "json" || kind === "array"}>{displayValue(row[f.name], kind)}</span>
                            {/if}
                        </dd>
                    </div>
                {/each}
            </dl>

            {#if !editing && target.kind === "row"}
                <a class="btn btn-sm open" href={dataHref(target.record.schema, target.record.table, target.record.column, target.record.value)}>
                    <Icon name="table" size={14} /> Open in grid
                </a>

                {#if groups.length > 0}
                    <h3>Referenced by</h3>
                    {#each groups as g}
                        <section class="group">
                            <div class="group-head">
                                <span class="mono">{g.table}<span class="muted">.{g.column}</span></span>
                                <span class="badge">{g.count ?? g.rows.length}</span>
                            </div>
                            {#each g.rows as r}
                                <button type="button" class="ref" onclick={() => openRow(g, r)}>{rowLabel(g.def, r)}</button>
                            {/each}
                            {#if (g.count ?? 0) > 0}
                                <a class="all" href={g.href}>Show all {g.count} in grid <Icon name="arrow-right-s" size={13} /></a>
                            {/if}
                        </section>
                    {/each}
                {/if}
            {/if}
        </form>

        {#if editing}
            <footer>
                {#if !isNew}
                    <button type="button" class="btn btn-sm btn-ghost btn-danger" disabled={busy} onclick={remove}>
                        <Icon name="delete-bin-6" size={13} /> Delete
                    </button>
                {/if}
                <span class="spacer"></span>
                <span class="hint">
                    {#if changed.length}{changed.length} {changed.length === 1 ? "change" : "changes"}{:else}<span class="kbd">⌘</span><span class="kbd">↵</span>{/if}
                </span>
                <button type="button" class="btn btn-sm" onclick={cancel}>Cancel</button>
                <button type="button" class="btn btn-sm btn-primary" disabled={busy || (!isNew && changed.length === 0)} onclick={() => save()}>
                    {busy ? "Saving..." : isNew ? "Create" : "Save"}
                </button>
            </footer>
        {/if}
    {/if}
</aside>

<style>
    .panel {
        display: flex;
        flex-direction: column;
        width: 400px;
        flex-shrink: 0;
        border-left: 1px solid var(--border);
        background: var(--bg);
        font-size: 12.5px;
    }
    header {
        display: flex;
        align-items: center;
        gap: 6px;
        height: 40px;
        padding: 0 6px 0 12px;
        border-bottom: 1px solid var(--border);
        color: var(--text-3);
    }
    .table {
        color: var(--text);
        font-weight: 600;
    }
    .key {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: var(--text-3);
    }
    .tools {
        margin-left: auto;
        display: flex;
        align-items: center;
        gap: 4px;
    }
    .pad {
        padding: 16px 12px;
    }
    .body {
        flex: 1;
        overflow: auto;
        padding: 10px 12px 16px;
    }
    .banner {
        margin-bottom: 10px;
    }
    dl {
        border: 1px solid var(--border);
        border-radius: var(--radius);
        overflow: hidden;
    }
    .row {
        display: grid;
        grid-template-columns: minmax(96px, 34%) minmax(0, 1fr);
        border-bottom: 1px solid var(--border);
    }
    .row:last-child {
        border-bottom: none;
    }
    .row.changed dt {
        box-shadow: inset 2px 0 0 var(--accent);
    }
    .row.changed dd {
        background: var(--accent-soft);
    }
    dt {
        display: flex;
        flex-direction: column;
        justify-content: center;
        gap: 1px;
        min-height: 30px;
        padding: 4px 8px;
        border-right: 1px solid var(--border);
        background: var(--bg-subtle);
        color: var(--text-2);
        font-size: 11.5px;
        overflow: hidden;
    }
    .name {
        display: flex;
        align-items: center;
        gap: 4px;
        min-width: 0;
    }
    .name .mono {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .type {
        color: var(--text-3);
        font-size: 10.5px;
    }
    .key-icon {
        display: inline-flex;
        color: var(--warning);
    }
    .req {
        color: var(--danger);
        font-weight: 600;
    }
    dd {
        display: flex;
        flex-direction: column;
        justify-content: center;
        gap: 2px;
        min-height: 30px;
        padding: 2px 8px;
        overflow-wrap: anywhere;
    }
    .editing dd {
        padding: 2px 4px;
    }
    dd.null {
        color: var(--text-3);
        font-size: 11.5px;
    }
    .value.mono {
        font-size: 11.5px;
    }
    .field-error {
        color: var(--danger);
        font-size: 11px;
        padding: 0 2px 2px;
    }
    .link {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 0;
        border: none;
        background: transparent;
        color: var(--accent-text);
        cursor: pointer;
        text-align: left;
    }
    .link:hover {
        text-decoration: underline;
        text-underline-offset: 2px;
    }
    .target {
        display: inline-flex;
        align-items: center;
        gap: 2px;
        color: var(--text-3);
        font-family: var(--font-mono);
        font-size: 11px;
    }
    .open {
        margin-top: 10px;
    }
    h3 {
        margin: 18px 0 8px;
        color: var(--text-3);
        font-size: 11.5px;
        font-weight: 600;
    }
    .group {
        display: flex;
        flex-direction: column;
        margin-bottom: 10px;
        border: 1px solid var(--border);
        border-radius: var(--radius);
        overflow: hidden;
    }
    .group-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 6px 8px;
        background: var(--bg-subtle);
        border-bottom: 1px solid var(--border);
    }
    .ref {
        padding: 5px 8px;
        border: none;
        border-bottom: 1px solid var(--border);
        background: transparent;
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 12px;
        text-align: left;
        cursor: pointer;
    }
    .ref:hover {
        background: var(--bg-hover);
        color: var(--accent-text);
    }
    .all {
        display: flex;
        align-items: center;
        gap: 2px;
        padding: 6px 8px;
        color: var(--accent-text);
        font-size: 12px;
    }
    .all:hover {
        background: var(--bg-hover);
    }
    footer {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 8px 10px;
        border-top: 1px solid var(--border);
        background: var(--bg-subtle);
    }
    .spacer {
        flex: 1;
    }
    .hint {
        display: inline-flex;
        align-items: center;
        gap: 3px;
        color: var(--text-3);
        font-size: 11.5px;
    }
</style>
