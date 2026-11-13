<script lang="ts">
    import { columnKind, typeLabel, type Kind } from "../coltypes";
    import {
        canUseAny,
        checkValue,
        operatorLabel,
        operatorsFor,
        valueShape,
        type Filter,
        type FilterState,
        type OpId,
    } from "../filters";
    import Icon from "./Icon.svelte";

    // A column the filters can use: a table column has its type; a view
    // column has the kind guessed from its values (inferKind).
    export interface FilterColumn {
        name: string;
        type?: string;
        kind?: Kind;
        notnull?: boolean;
    }

    interface Props {
        filterState: FilterState;
        columns: FilterColumn[];
        onChange: (next: FilterState) => void;
    }
    let { filterState, columns, onChange }: Props = $props();

    function kindOf(name: string): Kind {
        const c = columns.find((x) => x.name === name);
        return c?.kind ?? columnKind(c?.type);
    }
    const nullable = (name: string) => !columns.find((c) => c.name === name)?.notnull;

    // the editor: index of the filter it edits, -1 for a new one, null when closed
    let editing: number | null = $state(null);
    let column = $state("");
    let op: OpId = $state("eq");
    let value = $state("");
    let error = $state("");

    let kind = $derived(kindOf(column));
    let ops = $derived(operatorsFor(kind, nullable(column)));
    let shape = $derived(valueShape(op));

    function openEditor(i: number) {
        const f = i >= 0 ? filterState.filters[i] : null;
        column = f?.column ?? columns[0]?.name ?? "";
        op = f?.op ?? operatorsFor(kindOf(column), nullable(column))[0];
        value = f?.value ?? "";
        error = "";
        editing = i;
    }

    // a new column keeps the operator when the column has it
    function pickColumn(name: string) {
        column = name;
        const next = operatorsFor(kindOf(name), nullable(name));
        if (!next.includes(op)) op = next[0];
        error = "";
    }

    function apply(ev: Event) {
        ev.preventDefault();
        const f: Filter = { column, op, value: valueShape(op) === "none" ? "" : value };
        error = checkValue(f, kind) ?? "";
        if (error) return;
        const filters = [...filterState.filters];
        if (editing !== null && editing >= 0) filters[editing] = f;
        else filters.push(f);
        editing = null;
        onChange({ ...filterState, filters });
    }

    function remove(i: number) {
        onChange({ ...filterState, filters: filterState.filters.filter((_, j) => j !== i) });
    }

    function onWindowClick(ev: MouseEvent) {
        if (editing !== null && !(ev.target as Element).closest?.(".filter-zone")) editing = null;
    }

    function onKey(ev: KeyboardEvent) {
        if (editing !== null && ev.key === "Escape") editing = null;
    }

    // the text of a chip value: a list keeps its commas, a text gets quotes
    function shown(f: Filter): string {
        if (valueShape(f.op) === "none") return "";
        if (valueShape(f.op) === "list") return f.value;
        const k = kindOf(f.column);
        return k === "text" || k === "any" || k === "uuid" ? `“${f.value}”` : f.value;
    }

    const inputType = (k: Kind) =>
        k === "date" ? "date" : k === "timestamp" ? "datetime-local" : k === "time" ? "time" : "text";
    const placeholder = (k: Kind, o: OpId) =>
        valueShape(o) === "list"
            ? k === "int" || k === "float" || k === "numeric"
                ? "1, 2, 3"
                : "a, b, c"
            : k === "uuid"
              ? "00000000-0000-0000-0000-000000000000"
              : k === "int" || k === "float" || k === "numeric"
                ? "0"
                : "Value";
</script>

<svelte:window onclick={onWindowClick} onkeydown={onKey} />

<div class="filters filter-zone">
    {#if filterState.filters.length > 1}
        <div class="mode" role="radiogroup" aria-label="Rows that match">
            <span class="mode-label">Match</span>
            {#each [["and", "all"], ["or", "any"]] as [m, label]}
                <button
                    role="radio"
                    aria-checked={filterState.mode === m}
                    class:on={filterState.mode === m}
                    disabled={m === "or" && !canUseAny(filterState)}
                    title={m === "or" && !canUseAny(filterState) ? "A filter written by hand cannot go in an 'any' group" : undefined}
                    onclick={() => onChange({ ...filterState, mode: m as "and" | "or" })}>{label}</button
                >
            {/each}
        </div>
    {/if}

    {#each filterState.filters as f, i}
        {#if i > 0}<span class="joiner">{filterState.mode === "or" ? "or" : "and"}</span>{/if}
        <span class="chip" class:active={editing === i}>
            <button class="chip-body" onclick={() => openEditor(i)} title="Edit filter">
                {#if f.op === "raw"}
                    <span class="col">{f.column}</span><span class="raw">={f.value}</span>
                {:else}
                    <span class="col">{f.column}</span>
                    <span class="op">{operatorLabel(f.op, kindOf(f.column))}</span>
                    {#if shown(f)}<span class="val">{shown(f)}</span>{/if}
                {/if}
            </button>
            <button class="chip-x" aria-label={`Remove filter on ${f.column}`} onclick={() => remove(i)}>
                <Icon name="close" size={12} />
            </button>
        </span>
    {/each}

    <button class="btn btn-sm btn-ghost add" onclick={() => openEditor(-1)} aria-expanded={editing === -1}>
        <Icon name="filter-3" size={14} />
        {filterState.filters.length ? "Add" : "Filter"}
    </button>

    {#if editing !== null}
        <form class="editor" onsubmit={apply}>
            <div class="row">
                <label class="lbl" for="flt-col">Column</label>
                <select id="flt-col" class="input input-sm mono" value={column} onchange={(e) => pickColumn(e.currentTarget.value)}>
                    {#each columns as c}
                        <option value={c.name}>{c.name}{c.type ? `  ·  ${typeLabel(c.type)}` : c.kind && c.kind !== "any" ? `  ·  ${c.kind}` : ""}</option>
                    {/each}
                </select>
            </div>
            <div class="row">
                <label class="lbl" for="flt-op">Condition</label>
                {#if op === "raw"}
                    <span class="mono raw-note">written by hand</span>
                {:else}
                    <select id="flt-op" class="input input-sm" bind:value={op} onchange={() => (error = "")}>
                        {#each ops as o}<option value={o}>{operatorLabel(o, kind)}</option>{/each}
                    </select>
                {/if}
            </div>
            {#if shape !== "none"}
                <div class="row">
                    <label class="lbl" for="flt-val">{shape === "list" ? "Values" : "Value"}</label>
                    {#if op === "raw"}
                        <input id="flt-val" class="input input-sm mono" bind:value autocomplete="off" />
                    {:else if shape === "json"}
                        <textarea id="flt-val" class="input mono" rows="3" bind:value placeholder={'{"key": "value"}'}></textarea>
                    {:else}
                        <!-- svelte-ignore a11y_autofocus -->
                        <input
                            id="flt-val"
                            class="input input-sm"
                            class:mono={kind !== "text" && kind !== "any"}
                            type={shape === "list" ? "text" : inputType(kind)}
                            step={kind === "timestamp" || kind === "time" ? "1" : undefined}
                            inputmode={kind === "int" ? "numeric" : kind === "float" || kind === "numeric" ? "decimal" : undefined}
                            placeholder={placeholder(kind, op)}
                            bind:value
                            oninput={() => (error = "")}
                            autocomplete="off"
                            autofocus
                        />
                    {/if}
                </div>
            {/if}
            {#if shape === "list"}<p class="hint">Separate the values with commas.</p>{/if}
            {#if error}<p class="err">{error}</p>{/if}
            <div class="actions">
                <button type="button" class="btn btn-sm btn-ghost" onclick={() => (editing = null)}>Cancel</button>
                <button type="submit" class="btn btn-sm btn-primary">{editing === -1 ? "Add filter" : "Apply"}</button>
            </div>
        </form>
    {/if}
</div>

<style>
    .filters {
        position: relative;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 6px;
        min-height: var(--h-control-sm);
    }
    .mode {
        display: inline-flex;
        align-items: center;
        height: var(--h-control-sm);
        padding: 2px;
        border: 1px solid var(--border-strong);
        border-radius: var(--radius);
        background: var(--bg-subtle);
        font-size: 11.5px;
    }
    .mode-label {
        padding: 0 6px 0 4px;
        color: var(--text-3);
    }
    .mode button {
        height: 100%;
        padding: 0 8px;
        border: none;
        border-radius: var(--radius-sm);
        background: transparent;
        color: var(--text-2);
        cursor: pointer;
    }
    .mode button.on {
        background: var(--bg);
        color: var(--text);
        font-weight: 500;
        box-shadow: 0 1px 2px rgba(0, 0, 0, 0.12);
    }
    .mode button:disabled {
        opacity: 0.4;
        cursor: default;
    }
    .joiner {
        color: var(--text-3);
        font-size: 11px;
        font-family: var(--font-mono);
    }
    .chip {
        display: inline-flex;
        align-items: stretch;
        height: var(--h-control-sm);
        border: 1px solid var(--border-strong);
        border-radius: var(--radius);
        background: var(--bg-subtle);
        font-size: 12px;
        overflow: hidden;
    }
    .chip.active {
        border-color: var(--accent);
        box-shadow: var(--focus-ring);
    }
    .chip-body {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        padding: 0 6px 0 8px;
        border: none;
        background: transparent;
        cursor: pointer;
        max-width: 360px;
    }
    .chip-body:hover {
        background: var(--bg-hover);
    }
    .col {
        font-family: var(--font-mono);
        font-size: 11.5px;
        color: var(--text);
    }
    .op {
        color: var(--text-3);
    }
    .val,
    .raw {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: var(--accent-text);
        font-weight: 500;
    }
    .raw {
        font-family: var(--font-mono);
        font-size: 11.5px;
        font-weight: 400;
    }
    .chip-x {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 20px;
        border: none;
        border-left: 1px solid var(--border);
        background: transparent;
        color: var(--text-3);
        cursor: pointer;
    }
    .chip-x:hover {
        color: var(--danger);
        background: var(--bg-hover);
    }
    .editor {
        position: absolute;
        top: calc(100% + 6px);
        left: 0;
        z-index: 40;
        display: flex;
        flex-direction: column;
        gap: 8px;
        width: 340px;
        padding: 12px;
        border: 1px solid var(--border);
        border-radius: var(--radius-lg);
        background: var(--bg);
        box-shadow: var(--shadow-pop);
    }
    .row {
        display: grid;
        grid-template-columns: 76px minmax(0, 1fr);
        align-items: center;
        gap: 8px;
    }
    .row:has(textarea) {
        align-items: start;
    }
    .lbl {
        color: var(--text-2);
        font-size: 12px;
    }
    .raw-note {
        color: var(--text-3);
        font-size: 11.5px;
    }
    .hint,
    .err {
        margin-left: 84px;
        font-size: 11.5px;
    }
    .hint {
        color: var(--text-3);
    }
    .err {
        color: var(--danger);
    }
    .actions {
        display: flex;
        justify-content: flex-end;
        gap: 6px;
        margin-top: 4px;
    }
</style>
