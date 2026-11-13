<script lang="ts">
    import { columnKind, isNumeric } from "../coltypes";
    import { specText, type ColumnSpec, type RollupFn } from "../columns";
    import type { Reference, TableDef } from "../data.svelte";
    import Icon from "./Icon.svelte";

    interface Props {
        def: TableDef;
        specs: ColumnSpec[];
        // the definition of the table each foreign key points to, by foreign key column
        targets: Record<string, TableDef | null>;
        // the tables whose foreign keys point to this one
        refs: { ref: Reference; def: TableDef | null }[];
        // null goes back to the default columns
        onChange: (specs: ColumnSpec[] | null) => void;
    }
    let { def, specs, targets, refs, onChange }: Props = $props();

    let open = $state(false);
    let root: HTMLElement | undefined = $state();
    // the menu opens to the right of the button, or to its left when the
    // pane has no room on the right
    let alignRight = $state(false);
    const menuWidth = 260;

    function toggleOpen() {
        if (!open && root) {
            const pane = root.closest(".content-pane") ?? document.body;
            const r = root.getBoundingClientRect();
            alignRight = r.left + menuWidth > pane.getBoundingClientRect().right - 8;
        }
        open = !open;
    }
    let chosen = $derived(new Set(specs.map(specText)));

    interface Rollup {
        fn: RollupFn;
        arg?: string;
        label: string;
    }

    // the aggregates offered for a referencing table: count, the sum of a
    // number, the latest of a date
    function rollups(r: { ref: Reference; def: TableDef | null }): Rollup[] {
        const out: Rollup[] = [{ fn: "count", label: "count" }];
        for (const c of r.def?.columns ?? []) {
            if (r.def?.primaryKey.includes(c.name) || r.def?.foreignKeys.some((f) => f.column === c.name)) continue;
            const k = columnKind(c.type);
            if (isNumeric(k)) out.push({ fn: "sum", arg: c.name, label: `sum of ${c.name}` });
            else if (k === "date" || k === "timestamp") out.push({ fn: "max", arg: c.name, label: `latest ${c.name}` });
        }
        return out;
    }

    // The specs in display order: the columns of the table, each followed by
    // its lookups, then the rollups.
    function ordered(set: Set<string>): ColumnSpec[] {
        const out: ColumnSpec[] = [];
        for (const c of def.columns) {
            if (set.has(c.name)) out.push({ kind: "own", name: c.name });
            const target = targets[c.name];
            for (const t of target?.columns ?? []) {
                if (set.has(`${c.name}>${t.name}`)) out.push({ kind: "lookup", fk: c.name, target: t.name });
            }
        }
        for (const r of refs) {
            for (const x of rollups(r)) {
                const s: ColumnSpec = { kind: "rollup", table: r.ref.table, column: r.ref.column, fn: x.fn, arg: x.arg };
                if (set.has(specText(s))) out.push(s);
            }
        }
        return out;
    }

    function toggle(s: ColumnSpec) {
        const set = new Set(chosen);
        const t = specText(s);
        if (set.has(t)) set.delete(t);
        else set.add(t);
        onChange(ordered(set));
    }

    let extras = $derived(specs.filter((s) => s.kind !== "own").length);
    let hidden = $derived(def.columns.length - specs.filter((s) => s.kind === "own").length);

    function onWindowClick(ev: MouseEvent) {
        if (open && !(ev.target as Element).closest?.(".column-picker")) open = false;
    }
    function onKey(ev: KeyboardEvent) {
        if (open && ev.key === "Escape") open = false;
    }
</script>

<svelte:window onclick={onWindowClick} onkeydown={onKey} />

<div class="column-picker" bind:this={root}>
    <button class="btn btn-sm btn-ghost" aria-expanded={open} onclick={toggleOpen}>
        <Icon name="table" size={14} /> Columns
        {#if extras}<span class="badge badge-accent">+{extras}</span>{/if}
        {#if hidden > 0}<span class="badge">{hidden} hidden</span>{/if}
    </button>

    {#if open}
        <div class="menu" class:right={alignRight}>
            <div class="scroll">
                <h4>This table</h4>
                {#each def.columns as c}
                    {@const s = { kind: "own", name: c.name } as ColumnSpec}
                    <label class="item">
                        <input type="checkbox" checked={chosen.has(c.name)} onchange={() => toggle(s)} />
                        <span class="mono">{c.name}</span>
                    </label>
                {/each}

                {#each def.foreignKeys as fk}
                    {@const target = targets[fk.column]}
                    {#if target}
                        <h4>
                            <Icon name="arrow-right-up" size={12} />
                            <span class="mono">{fk.column}</span> → <span class="mono">{fk.table}</span>
                        </h4>
                        {#each target.columns.filter((t) => t.name !== fk.refColumn) as t}
                            {@const s = { kind: "lookup", fk: fk.column, target: t.name } as ColumnSpec}
                            <label class="item">
                                <input type="checkbox" checked={chosen.has(specText(s))} onchange={() => toggle(s)} />
                                <span class="mono">{t.name}</span>
                            </label>
                        {/each}
                    {/if}
                {/each}

                {#each refs as r}
                    <h4>
                        <Icon name="arrow-left-s" size={12} />
                        <span class="mono">{r.ref.table}</span> by <span class="mono">{r.ref.column}</span>
                    </h4>
                    {#each rollups(r) as x}
                        {@const s = { kind: "rollup", table: r.ref.table, column: r.ref.column, fn: x.fn, arg: x.arg } as ColumnSpec}
                        <label class="item">
                            <input type="checkbox" checked={chosen.has(specText(s))} onchange={() => toggle(s)} />
                            <span>{x.label}</span>
                        </label>
                    {/each}
                {/each}
            </div>
            <div class="foot">
                <button class="btn btn-sm btn-ghost" onclick={() => onChange(null)}>Reset</button>
            </div>
        </div>
    {/if}
</div>

<style>
    .column-picker {
        position: relative;
    }
    .menu {
        position: absolute;
        top: calc(100% + 6px);
        left: 0;
        z-index: 40;
        display: flex;
        flex-direction: column;
        width: 260px;
        max-height: 420px;
        border: 1px solid var(--border);
        border-radius: var(--radius-lg);
        background: var(--bg);
        box-shadow: var(--shadow-pop);
    }
    .menu.right {
        left: auto;
        right: 0;
    }
    .scroll {
        overflow-y: auto;
        padding: 6px;
    }
    h4 {
        display: flex;
        align-items: center;
        gap: 4px;
        margin: 8px 6px 4px;
        color: var(--text-3);
        font-size: 11px;
        font-weight: 600;
    }
    h4:first-child {
        margin-top: 2px;
    }
    h4 .mono {
        font-weight: 400;
        color: var(--text-2);
    }
    .item {
        display: flex;
        align-items: center;
        gap: 8px;
        height: 26px;
        padding: 0 6px;
        border-radius: var(--radius-sm);
        cursor: pointer;
        font-size: 12.5px;
    }
    .item:hover {
        background: var(--bg-hover);
    }
    .foot {
        display: flex;
        justify-content: flex-end;
        padding: 6px;
        border-top: 1px solid var(--border);
    }
</style>
