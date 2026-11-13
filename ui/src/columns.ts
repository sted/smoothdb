// The columns of the data grid: the columns of the table, columns of the rows
// that its foreign keys point to (lookups), and aggregates of the rows that
// point to it (rollups).
//
// The UI keeps them in its query parameter cols, one spec for each column:
//   name                              a column of the table
//   customer_id>email                 email of the row that customer_id points to
//   orders.customer_id:count          the number of orders whose customer_id points here
//   orders.customer_id:sum(amount)    the sum of their amount
// Without cols the grid shows every column of the table and nothing else.

import type { ForeignKey, Reference, TableDef } from './data.svelte';

export type RollupFn = 'count' | 'sum' | 'avg' | 'min' | 'max';

export type ColumnSpec =
    | { kind: 'own'; name: string }
    | { kind: 'lookup'; fk: string; target: string }
    | { kind: 'rollup'; table: string; column: string; fn: RollupFn; arg?: string };

export function specText(s: ColumnSpec): string {
    if (s.kind === 'own') return s.name;
    if (s.kind === 'lookup') return `${s.fk}>${s.target}`;
    return `${s.table}.${s.column}:${s.fn}${s.arg ? `(${s.arg})` : ''}`;
}

export function parseSpec(text: string): ColumnSpec | null {
    let m = text.match(/^([^>:.]+)>(.+)$/);
    if (m) return { kind: 'lookup', fk: m[1], target: m[2] };
    m = text.match(/^([^.:]+)\.([^:]+):(count|sum|avg|min|max)(?:\((.+)\))?$/);
    if (m) return { kind: 'rollup', table: m[1], column: m[2], fn: m[3] as RollupFn, arg: m[4] };
    return text ? { kind: 'own', name: text } : null;
}

export const fkAlias = (column: string) => `_fk_${column}`;

const ident = (s: string) => s.replace(/\W/g, '_');

function rollupAlias(s: Extract<ColumnSpec, { kind: 'rollup' }>): string {
    return `_ag_${ident(s.table)}_${ident(s.column)}_${s.fn}${s.arg ? `_${ident(s.arg)}` : ''}`;
}

// The grid key of a column. For a lookup it is also its PostgREST order term.
export function columnKey(s: ColumnSpec): string {
    if (s.kind === 'own') return s.name;
    if (s.kind === 'lookup') return `${fkAlias(s.fk)}(${s.target})`;
    return rollupAlias(s);
}

export function columnLabel(s: ColumnSpec, fks: Map<string, ForeignKey>): string {
    if (s.kind === 'own') return s.name;
    if (s.kind === 'lookup') return `${fks.get(s.fk)?.table ?? s.fk}.${s.target}`;
    const words: Record<RollupFn, string> = { count: 'count', sum: 'sum of', avg: 'average', min: 'earliest', max: 'latest' };
    return s.fn === 'count' ? `${s.table} · count` : `${s.table} · ${words[s.fn]} ${s.arg}`;
}

// readColumns returns the specs of the query parameters; without cols, every column of the table.
export function readColumns(params: URLSearchParams, def: TableDef | null): ColumnSpec[] {
    const text = params.get('cols');
    if (text === null) return (def?.columns ?? []).map((c) => ({ kind: 'own', name: c.name }));
    return text
        .split(',')
        .map(parseSpec)
        .filter((s): s is ColumnSpec => s !== null);
}

export function writeColumns(params: URLSearchParams, specs: ColumnSpec[] | null) {
    if (specs === null) params.delete('cols');
    else params.set('cols', specs.map(specText).join(','));
}

// The value of a lookup or rollup cell in a row of the response.
export function cellValue(row: Record<string, any>, s: ColumnSpec): unknown {
    if (s.kind === 'own') return row[s.name];
    if (s.kind === 'lookup') return row[fkAlias(s.fk)]?.[s.target] ?? null;
    const agg = row[rollupAlias(s)];
    return Array.isArray(agg) ? (agg[0]?.[s.fn] ?? null) : null;
}

// buildSelect returns the PostgREST select of the grid: every column of the
// table (the form, the links and the key need them), one embed for each
// foreign key with its label and its lookups, one embed for each rollup.
export function buildSelect(def: TableDef | null, specs: ColumnSpec[], fkLabels: Record<string, string>): string {
    const parts = ['*'];
    const byFk = new Map<string, Set<string>>();
    for (const [fk, label] of Object.entries(fkLabels)) byFk.set(fk, new Set([label]));
    for (const s of specs) {
        if (s.kind !== 'lookup') continue;
        if (!byFk.has(s.fk)) byFk.set(s.fk, new Set());
        byFk.get(s.fk)!.add(s.target);
    }
    for (const [fk, cols] of byFk) {
        const f = def?.foreignKeys.find((x) => x.column === fk);
        if (f && cols.size) parts.push(`${fkAlias(fk)}:${f.table}!${fk}(${[...cols].join(',')})`);
    }
    for (const s of specs) {
        if (s.kind !== 'rollup') continue;
        const body = s.fn === 'count' ? 'count()' : `${s.arg}.${s.fn}()`;
        parts.push(`${rollupAlias(s)}:${s.table}!${s.column}(${body})`);
    }
    return parts.join(',');
}

// The rows a rollup counts, as a filtered grid of the other table.
export function rollupHref(db: string, ref: Reference, row: Record<string, any>): string {
    const value = encodeURIComponent(String(row[ref.refColumn]));
    return `/ui/data/${encodeURIComponent(db)}/${encodeURIComponent(ref.schema)}/${encodeURIComponent(ref.table)}?${encodeURIComponent(ref.column)}=eq.${value}`;
}
