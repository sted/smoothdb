// What the Data area knows about the tables of a database. It reads the data
// API only (/api and /token), so it works for a role without access to /admin.

import { makeRequest } from './api';
import { adminDbUrl, dbApiUrl, ui } from './config.svelte';
import { parsePrimaryKey } from './utils';

// A column as $info and the admin API return it.
export interface Column {
    name: string;
    type: string;
    notnull: boolean;
    default: string | null;
    readonly: boolean;
    schema: string;
    constraints?: string[] | null;
}

// A single-column foreign key.
export interface ForeignKey {
    column: string;
    schema: string;
    table: string;
    refColumn: string;
}

export interface Relation {
    name: string;
    schema: string;
}

// The definition of a table. A view has none: the UI shows it read-only.
export interface TableDef {
    columns: Column[];
    primaryKey: string[];
    foreignKeys: ForeignKey[];
    // the column that names a row, for links to it
    labelColumn?: string;
}

// A column of another table that references a table.
export interface Reference {
    schema: string;
    table: string;
    column: string;
    refColumn: string;
}

const unquote = (n: string) => (n.startsWith('"') ? n.slice(1, -1).replace(/""/g, '"') : n);

// "FOREIGN KEY (customer_id) REFERENCES customers(id)", optionally with a schema.
export function parseForeignKey(def: string, schema: string): ForeignKey | null {
    const m = def.match(/^FOREIGN KEY \((.+?)\) REFERENCES (.+?)\((.+?)\)/);
    if (!m || m[1].includes(',')) return null;
    const target = m[2].match(/^(?:("(?:[^"]|"")+"|[^".]+)\.)?("(?:[^"]|"")+"|[^".]+)$/);
    if (!target) return null;
    return {
        column: unquote(m[1]),
        schema: target[1] ? unquote(target[1]) : schema,
        table: unquote(target[2]),
        refColumn: unquote(m[3]),
    };
}

const labelNames = ['name', 'title', 'label', 'display_name', 'username', 'email', 'code', 'slug'];
const textTypes = new Set(['text', 'varchar', 'bpchar', 'citext']);

function labelColumn(columns: Column[], primaryKey: string[]): string | undefined {
    for (const n of labelNames) if (columns.some((c) => c.name === n)) return n;
    return columns.find((c) => textTypes.has(c.type) && !primaryKey.includes(c.name))?.name;
}

const key = (...parts: string[]) => parts.join('\u0000');

// A failed request is not an answer: a 401 before the login, a network error.
// The caches keep only answers, so the next call asks again.
class Retry extends Error {}

function cached<T>(cache: Map<string, Promise<T>>, k: string, load: () => Promise<T>): Promise<T> {
    let p = cache.get(k);
    if (!p) {
        p = load();
        cache.set(k, p);
        p.catch(() => cache.delete(k));
    }
    return p;
}
const defs = new Map<string, Promise<TableDef | null>>();
const relations = new Map<string, Promise<{ tables: Relation[]; views: Relation[] }>>();

// tableDef returns the definition of a table, or null for a view.
export function tableDef(db: string, schema: string, table: string): Promise<TableDef | null> {
    return cached(defs, key(db, schema, table), () =>
        makeRequest(`${dbApiUrl(db)}/$info/${encodeURIComponent(table)}`, {
            headers: { 'Accept-Profile': schema },
            quiet: true,
        }).then((r) => {
            // $info answers 404 for a view
            if (r.status === 404) return null;
            if (!r.success || !r.data?.columns) throw new Retry(r.error);
            const columns: Column[] = r.data.columns;
            const primaryKey = parsePrimaryKey(r.data.constraints);
            const foreignKeys = columns
                .flatMap((c) => c.constraints ?? [])
                .map((d) => parseForeignKey(d, schema))
                .filter((f): f is ForeignKey => f !== null);
            return { columns, primaryKey, foreignKeys, labelColumn: labelColumn(columns, primaryKey) };
        }),
    ).catch(() => null);
}

// listRelations returns the tables of a database from the data API. The data
// API does not list views: they come from the admin API when the role can use it.
export function listRelations(db: string): Promise<{ tables: Relation[]; views: Relation[] }> {
    return cached(relations, db, () =>
        Promise.all([
            makeRequest(`${dbApiUrl(db)}/`),
            ui.admin === false
                ? Promise.resolve({ success: true, data: [] })
                : makeRequest(`${adminDbUrl}/databases/${encodeURIComponent(db)}/views`, { quiet: true }),
        ]).then(([t, v]) => {
            if (!t.success) throw new Retry(t.error);
            return {
                tables: Array.isArray(t.data) ? t.data : [],
                views: v.success && Array.isArray(v.data) ? v.data : [],
            };
        }),
    ).catch(() => ({ tables: [], views: [] }));
}

// references returns the columns of other tables that reference a table.
export async function references(db: string, schema: string, table: string): Promise<Reference[]> {
    const { tables } = await listRelations(db);
    const all = await Promise.all(tables.map(async (t) => ({ t, def: await tableDef(db, t.schema, t.name) })));
    return all.flatMap(({ t, def }) =>
        (def?.foreignKeys ?? [])
            .filter((f) => f.schema === schema && f.table === table)
            .map((f) => ({ schema: t.schema, table: t.name, column: f.column, refColumn: f.refColumn })),
    );
}

// forget drops the cached definitions of a database, after a schema change.
export function forget(db: string) {
    relations.delete(db);
    for (const k of defs.keys()) if (k.startsWith(db + '\u0000')) defs.delete(k);
}

// listDatabases tells whether the role can use the admin API (ui.admin) and
// returns the databases it lists. The data API has no database list.
const databases = new Map<string, Promise<{ name: string }[]>>();
export function listDatabases(): Promise<{ name: string }[]> {
    return cached(databases, '', () =>
        makeRequest(`${adminDbUrl}/databases`, { quiet: true }).then((r) => {
            if (r.status === 401 || r.status === 0) throw new Retry(r.error);
            ui.admin = r.success;
            return r.success && Array.isArray(r.data) ? r.data : [];
        }),
    ).catch(() => []);
}

// forgetAll drops every cache, after a login or a logout.
export function forgetAll() {
    defs.clear();
    viewCols.clear();
    relations.clear();
    databases.clear();
    ui.admin = null;
}

// viewColumns returns the columns of a view with their types. The data API
// does not describe views ($info answers 404), so they come from the admin
// API when the role can use it; null otherwise.
const viewCols = new Map<string, Promise<Column[] | null>>();
export function viewColumns(db: string, schema: string, view: string): Promise<Column[] | null> {
    if (ui.admin === false) return Promise.resolve(null);
    return cached(viewCols, key(db, schema, view), () =>
        makeRequest(`${adminDbUrl}/databases/${encodeURIComponent(db)}/tables/${encodeURIComponent(view)}/columns`, {
            headers: { 'Accept-Profile': schema },
            quiet: true,
        }).then((r) => {
            if (r.status === 401 || r.status === 0) throw new Retry(r.error);
            return r.success && Array.isArray(r.data) ? (r.data as Column[]) : null;
        }),
    ).catch(() => null);
}
