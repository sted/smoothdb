// Cached lists that the admin forms offer in their selects: the roles of the
// server and the schemas of a database. A change of roles or schemas drops them.

import { makeRequest } from '../api';
import { adminDbUrl } from '../config.svelte';

let roles: Promise<string[]> | null = null;
const schemas = new Map<string, Promise<string[]>>();

const names = (r: { success: boolean; data?: any }) =>
    r.success && Array.isArray(r.data) ? r.data.map((x: { name: string }) => x.name) : [];

export function roleNames(): Promise<string[]> {
    roles ??= makeRequest(`${adminDbUrl}/roles`, { quiet: true }).then(names);
    return roles;
}

export function schemaNames(db: string): Promise<string[]> {
    let p = schemas.get(db);
    if (!p) {
        p = makeRequest(`${adminDbUrl}/databases/${encodeURIComponent(db)}/schemas`, { quiet: true }).then(names);
        schemas.set(db, p);
    }
    return p;
}

export function forgetRefs(kind: string, db?: string) {
    if (kind === 'roles') roles = null;
    if (kind === 'schemas' && db) schemas.delete(db);
}

// "schema.table" for every table of a database, for a foreign key target
const tables = new Map<string, Promise<string[]>>();
export function tableNames(db: string): Promise<string[]> {
    let p = tables.get(db);
    if (!p) {
        p = makeRequest(`${adminDbUrl}/databases/${encodeURIComponent(db)}/tables`, { quiet: true }).then((r) =>
            r.success && Array.isArray(r.data) ? r.data.map((t: { schema: string; name: string }) => `${t.schema}.${t.name}`) : [],
        );
        tables.set(db, p);
    }
    return p;
}

// the columns of a table, for the columns of a constraint or of a grant
const columns = new Map<string, Promise<string[]>>();
export function columnNames(db: string, schema: string, table: string): Promise<string[]> {
    const k = `${db}\u0000${schema}\u0000${table}`;
    let p = columns.get(k);
    if (!p) {
        p = makeRequest(`${adminDbUrl}/databases/${encodeURIComponent(db)}/tables/${encodeURIComponent(table)}/columns`, {
            headers: schema ? { 'Accept-Profile': schema } : {},
            quiet: true,
        }).then(names);
        columns.set(k, p);
    }
    return p;
}

// a DDL change in a database drops its cached tables and columns
export function forgetTables(db: string) {
    tables.delete(db);
    for (const k of columns.keys()) if (k.startsWith(db + '\u0000')) columns.delete(k);
}
