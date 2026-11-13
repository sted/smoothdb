<script lang="ts">
    import type { Snippet } from "svelte";
    import { formatCell } from "../utils";
    import Icon from "./Icon.svelte";

    // A column of the grid. key is the row key, or the order term for onSort.
    export interface GridColumn {
        key: string;
        label?: string;
        // right-aligned in a tabular font
        numeric?: boolean;
        sortable?: boolean;
        // a column that comes from another table
        related?: boolean;
    }

    interface Props {
        rows: Record<string, any>[];
        // the columns; without them, the keys of the first row
        columns?: GridColumn[];
        loading?: boolean;
        // the PostgREST order of the rows, as "col.asc,other.desc"
        order?: string;
        onSort?: (column: string) => void;
        onEdit?: (row: any) => void;
        onRowClick?: (row: any) => void;
        onCellDoubleClick?: (row: any, key: string) => void;
        isSelected?: (row: any) => boolean;
        // the content of a cell; without it, the formatted value
        cell?: Snippet<[row: Record<string, any>, column: string]>;
    }
    let { rows, columns, loading = false, order = "", onSort, onEdit, onRowClick, onCellDoubleClick, isSelected, cell }: Props =
        $props();

    let cols: GridColumn[] = $derived(
        columns ?? (rows.length > 0 ? Object.keys(rows[0]) : []).map((key) => ({ key, sortable: true })),
    );

    // a click on a link or a button inside a cell is not a click on the row
    function rowClick(ev: MouseEvent, row: Record<string, any>) {
        if ((ev.target as Element).closest("a, button, input, select, textarea")) return;
        onRowClick?.(row);
    }

    // the direction of a column in the order, for the header arrow
    function direction(column: string): "asc" | "desc" | null {
        for (const part of order.split(",")) {
            const [c, d] = part.split(".");
            if (c === column) return d === "desc" ? "desc" : "asc";
        }
        return null;
    }
</script>

<div class="grid-container" class:loading>
    <table>
        <thead>
            <tr>
                {#if onEdit}<th class="edit-column"><span class="sr-only">Edit</span></th>{/if}
                {#each cols as col}
                    {@const dir = direction(col.key)}
                    <th
                        class:num={col.numeric}
                        class:related={col.related}
                        aria-sort={dir === "asc" ? "ascending" : dir === "desc" ? "descending" : undefined}
                    >
                        {#if onSort && col.sortable !== false}
                            <button class="sort" class:sorted={dir} onclick={() => onSort(col.key)}>
                                <span>{col.label ?? col.key}</span>
                                <Icon name={dir === "desc" ? "arrow-down-s" : "arrow-up-s"} size={13} />
                            </button>
                        {:else}
                            <span class="plain">{col.label ?? col.key}</span>
                        {/if}
                    </th>
                {/each}
            </tr>
        </thead>
        <tbody>
            {#each rows as row}
                <tr class:clickable={onRowClick} class:selected={isSelected?.(row)} onclick={(ev) => rowClick(ev, row)}>
                    {#if onEdit}
                        <td class="edit-column">
                            <button
                                class="edit"
                                aria-label="Edit row"
                                onclick={(ev) => {
                                    ev.stopPropagation();
                                    onEdit(row);
                                }}
                            >
                                <Icon name="edit" size={13} />
                            </button>
                        </td>
                    {/if}
                    {#each cols as col}
                        {@const value = row[col.key]}
                        {@const text = formatCell(value)}
                        <td
                            class:null={!cell && (value === null || value === undefined)}
                            class:num={col.numeric}
                            ondblclick={() => onCellDoubleClick?.(row, col.key)}
                        >
                            {#if cell}{@render cell(row, col.key)}{:else}<span title={text}>{text}</span>{/if}
                        </td>
                    {/each}
                </tr>
            {:else}
                <tr>
                    <td class="no-data" colspan={cols.length + (onEdit ? 1 : 0) || 1}>
                        {loading ? "Loading..." : "No rows match."}
                    </td>
                </tr>
            {/each}
        </tbody>
    </table>
</div>

<style>
    .grid-container {
        height: 100%;
        overflow: auto;
    }
    table {
        border-collapse: separate;
        border-spacing: 0;
        min-width: 100%;
        font-size: 12.5px;
    }
    th {
        position: sticky;
        top: 0;
        z-index: 1;
        height: 30px;
        padding: 0;
        border-bottom: 1px solid var(--border);
        border-right: 1px solid var(--border);
        background: var(--bg-subtle);
        color: var(--text-2);
        font-weight: 500;
        text-align: left;
        white-space: nowrap;
    }
    .sort,
    .plain {
        display: flex;
        align-items: center;
        gap: 4px;
        width: 100%;
        height: 100%;
        padding: 0 10px;
    }
    .sort {
        border: none;
        background: transparent;
        color: inherit;
        font-weight: inherit;
        cursor: pointer;
    }
    th.related {
        background: color-mix(in srgb, var(--accent-soft) 60%, var(--bg-subtle));
    }
    th.num .sort,
    th.num .plain {
        justify-content: flex-end;
    }
    .sort :global(.icon) {
        visibility: hidden;
        color: var(--text-3);
    }
    .sort:hover {
        background: var(--bg-hover);
        color: var(--text);
    }
    .sort:hover :global(.icon) {
        visibility: visible;
    }
    .sort.sorted {
        color: var(--text);
    }
    .sort.sorted :global(.icon) {
        visibility: visible;
        color: var(--accent);
    }
    td {
        height: 30px;
        max-width: 320px;
        padding: 0 10px;
        border-bottom: 1px solid var(--border);
        border-right: 1px solid var(--border);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    td.num {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }
    td.null {
        color: var(--text-3);
        font-size: 11.5px;
    }
    tbody tr:hover td {
        background: var(--bg-hover);
    }
    tr.clickable {
        cursor: pointer;
    }
    tr.selected td {
        background: var(--accent-soft) !important;
    }
    .edit-column {
        width: 32px;
        min-width: 32px;
        padding: 0;
        text-align: center;
    }
    .edit {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 22px;
        height: 22px;
        border: none;
        border-radius: var(--radius-sm);
        background: transparent;
        color: var(--text-3);
        cursor: pointer;
        opacity: 0;
    }
    tr:hover .edit,
    .edit:focus-visible {
        opacity: 1;
    }
    .edit:hover {
        background: var(--bg-active);
        color: var(--text);
    }
    .grid-container.loading tbody {
        opacity: 0.55;
    }
    .no-data {
        height: 80px;
        text-align: center;
        color: var(--text-3);
    }
    .sr-only {
        position: absolute;
        width: 1px;
        height: 1px;
        overflow: hidden;
        clip: rect(0 0 0 0);
    }
</style>
