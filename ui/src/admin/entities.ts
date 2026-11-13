// The kinds of object of the admin area. A descriptor says where a kind
// lives in the admin API, which fields a form shows, which of them change
// after the creation, and how the list shows the objects. The admin page and
// the entity panel read only descriptors: no kind has a form of its own.

import { adminDbUrl } from '../config.svelte';
import { typeLabel } from '../coltypes';

export type Item = Record<string, any>;

// Where the admin page is: the database, and the table with its schema.
export interface AdminContext {
    db: string;
    schema: string;
    table: string;
}

export type FieldKind =
    | 'text' // one line
    | 'sql' // an SQL fragment: a default, a check, a view definition
    | 'bool'
    | 'role' // one role of the server
    | 'roles' // several roles, PUBLIC included
    | 'schema' // one schema of the database
    | 'type' // a column type, with its modifier: numeric(10,2)
    | 'select' // one of options
    | 'columns' // several columns of the table of the page
    | 'table' // one table of the database, as schema.table
    | 'refcolumns' // several columns of the table that the field `from` names
    | 'multi' // several of options
    | 'args'; // the arguments of a function: "a int4, out total numeric"

export interface FieldDef {
    key: string;
    label: string;
    kind: FieldKind;
    // the value cannot change after the creation
    createOnly?: boolean;
    // the value cannot change after the creation for some objects only
    createOnlyWhen?: (item: Item) => boolean;
    // the value is shown, never sent
    readonly?: boolean;
    required?: boolean;
    options?: string[];
    placeholder?: string;
    help?: string;
    // a multi-line field
    multiline?: boolean;
    // the field shows only when this holds for the values of the form
    when?: (values: Item) => boolean;
    // for refcolumns: the field that names the table
    from?: string;
    // for role: the label of the empty choice
    emptyLabel?: string;
    // the value of the field for an object, when it is not item[key] as is
    read?: (item: Item) => any;
}

export interface ListColumn {
    key: string;
    label: string;
    // the text of the cell; without it, the value as is
    text?: (item: Item) => string;
    // a boolean shown as a badge when true
    flag?: boolean;
    mono?: boolean;
    // the cell links to the object's page (open)
    link?: boolean;
}

export interface Request {
    url: string;
    headers?: Record<string, string>;
    body?: unknown;
}

export interface EntityDef {
    kind: string;
    singular: string;
    plural: string;
    icon: string;
    list: (ctx: AdminContext) => Request;
    key: (item: Item) => string;
    title: (item: Item) => string;
    columns: ListColumn[];
    fields: FieldDef[];
    // the initial values of a new object
    initial?: (ctx: AdminContext) => Item;
    create?: (ctx: AdminContext, values: Item) => Request;
    // the changed fields only; the key of the object comes from item. Some
    // changes take two requests in order (a view definition, then a rename).
    update?: (ctx: AdminContext, item: Item, changed: Item) => Request | Request[];
    remove?: (ctx: AdminContext, item: Item) => Request;
    // what the confirmation of a drop says
    removeWarning?: (item: Item) => string;
    // the verb of the removal: Drop, or Revoke for a grant
    removeVerb?: string;
    // the admin page of the object's content
    open?: (ctx: AdminContext, item: Item) => string;
    // the object in the Data area
    data?: (ctx: AdminContext, item: Item) => string;
}

// ---- URLs

const enc = encodeURIComponent;
export const uiPath = (...segments: string[]) => '/ui/' + segments.map(enc).join('/');
const api = (...segments: string[]) => adminDbUrl + '/' + segments.map(enc).join('/');
const dbApi = (ctx: AdminContext, ...segments: string[]) => api('databases', ctx.db, ...segments);
const profile = (schema: string): Record<string, string> =>
    schema ? { 'Content-Profile': schema, 'Accept-Profile': schema } : {};

// Only the keys present in values, so that an unchanged field is not sent.
function pick(values: Item, keys: string[]): Item {
    const out: Item = {};
    for (const k of keys) if (k in values) out[k] = values[k];
    return out;
}

const list = (v: unknown) => (Array.isArray(v) ? v.join(', ') : '');

// ---- descriptors

const databases: EntityDef = {
    kind: 'databases',
    singular: 'database',
    plural: 'Databases',
    icon: 'database-2',
    list: () => ({ url: api('databases') }),
    key: (d) => d.name,
    title: (d) => d.name,
    columns: [
        { key: 'name', label: 'name', mono: true, link: true },
        { key: 'owner', label: 'owner', mono: true },
    ],
    fields: [
        { key: 'name', label: 'Name', kind: 'text', required: true },
        { key: 'owner', label: 'Owner', kind: 'role' },
    ],
    create: (_, v) => ({ url: api('databases'), body: pick(v, ['name', 'owner']) }),
    update: (_, d, c) => ({ url: api('databases', d.name), body: pick(c, ['name', 'owner']) }),
    remove: (_, d) => ({ url: api('databases', d.name) }),
    removeWarning: (d) =>
        `The database ${d.name} and all its data are removed. Open connections to it are closed. You cannot undo this.`,
    open: (_, d) => uiPath('databases', d.name, 'tables'),
    data: (_, d) => uiPath('data', d.name),
};

const roleFlags: [string, string][] = [
    ['canlogin', 'Can log in'],
    ['issuperuser', 'Superuser'],
    ['cancreatedatabases', 'Can create databases'],
    ['cancreateroles', 'Can create roles'],
    ['canbypassrls', 'Bypasses row level security'],
    ['noinherit', 'Does not inherit privileges'],
];

const roles: EntityDef = {
    kind: 'roles',
    singular: 'role',
    plural: 'Roles',
    icon: 'group',
    list: () => ({ url: api('roles') }),
    key: (r) => r.name,
    title: (r) => r.name,
    columns: [
        { key: 'name', label: 'name', mono: true },
        { key: 'canlogin', label: 'login', flag: true },
        { key: 'issuperuser', label: 'superuser', flag: true },
        { key: 'cancreatedatabases', label: 'create db', flag: true },
        { key: 'cancreateroles', label: 'create role', flag: true },
        { key: 'canbypassrls', label: 'bypass rls', flag: true },
        { key: 'memberof', label: 'member of', text: (r) => list(r.memberof), mono: true },
    ],
    fields: [
        { key: 'name', label: 'Name', kind: 'text', required: true },
        ...roleFlags.map(([key, label]): FieldDef => ({ key, label, kind: 'bool' })),
        { key: 'memberof', label: 'Member of', kind: 'roles', readonly: true },
    ],
    initial: () => Object.fromEntries(roleFlags.map(([k]) => [k, false])),
    create: (_, v) => ({ url: api('roles'), body: pick(v, ['name', ...roleFlags.map(([k]) => k)]) }),
    update: (_, r, c) => ({ url: api('roles', r.name), body: pick(c, ['name', ...roleFlags.map(([k]) => k)]) }),
    remove: (_, r) => ({ url: api('roles', r.name) }),
    removeWarning: (r) => `The role ${r.name} is removed. It must own no object and hold no privilege. You cannot undo this.`,
};

const schemas: EntityDef = {
    kind: 'schemas',
    singular: 'schema',
    plural: 'Schemas',
    icon: 'stack',
    list: (ctx) => ({ url: dbApi(ctx, 'schemas') }),
    key: (s) => s.name,
    title: (s) => s.name,
    columns: [
        { key: 'name', label: 'name', mono: true },
        { key: 'owner', label: 'owner', mono: true },
    ],
    fields: [
        { key: 'name', label: 'Name', kind: 'text', required: true },
        { key: 'owner', label: 'Owner', kind: 'role' },
    ],
    create: (ctx, v) => ({ url: dbApi(ctx, 'schemas'), body: pick(v, ['name', 'owner']) }),
    update: (ctx, s, c) => ({ url: dbApi(ctx, 'schemas', s.name), body: pick(c, ['name', 'owner']) }),
    remove: (ctx, s) => ({ url: dbApi(ctx, 'schemas', s.name) }),
    removeWarning: (s) =>
        `The schema ${s.name} is dropped with CASCADE: every table, view and function in it is removed too. You cannot undo this.`,
};

const tables: EntityDef = {
    kind: 'tables',
    singular: 'table',
    plural: 'Tables',
    icon: 'table',
    list: (ctx) => ({ url: dbApi(ctx, 'tables') }),
    key: (t) => `${t.schema}.${t.name}`,
    title: (t) => t.name,
    columns: [
        { key: 'name', label: 'name', mono: true, link: true },
        { key: 'schema', label: 'schema', mono: true },
        { key: 'owner', label: 'owner', mono: true },
        { key: 'rowsecurity', label: 'rls', flag: true },
        { key: 'comment', label: 'comment' },
    ],
    fields: [
        { key: 'name', label: 'Name', kind: 'text', required: true },
        { key: 'schema', label: 'Schema', kind: 'schema' },
        { key: 'owner', label: 'Owner', kind: 'role' },
        { key: 'rowsecurity', label: 'Row level security', kind: 'bool' },
        { key: 'comment', label: 'Comment', kind: 'text' },
        {
            key: 'withId',
            label: 'Add an id column',
            kind: 'bool',
            createOnly: true,
            help: 'bigint, generated by default as identity, primary key',
        },
    ],
    initial: (ctx) => ({ schema: ctx.schema || 'public', rowsecurity: false, withId: true }),
    create: (ctx, v) => {
        const body = pick(v, ['name', 'schema', 'owner', 'rowsecurity', 'comment']);
        if (v.withId) {
            body.columns = [{ name: 'id', type: 'bigint generated by default as identity', notnull: true }];
            body.constraints = ['PRIMARY KEY (id)'];
        }
        return { url: dbApi(ctx, 'tables'), headers: profile(v.schema), body };
    },
    update: (ctx, t, c) => ({
        url: dbApi(ctx, 'tables', t.name),
        headers: profile(t.schema),
        body: pick(c, ['name', 'schema', 'owner', 'rowsecurity', 'comment']),
    }),
    remove: (ctx, t) => ({ url: dbApi(ctx, 'tables', t.name), headers: profile(t.schema) }),
    removeWarning: (t) => `The table ${t.schema}.${t.name} and all its rows are removed. You cannot undo this.`,
    open: (ctx, t) => uiPath('databases', ctx.db, 'tables', t.schema, t.name, 'columns'),
    data: (ctx, t) => uiPath('data', ctx.db, t.schema, t.name),
};

const columns: EntityDef = {
    kind: 'columns',
    singular: 'column',
    plural: 'Columns',
    icon: 'braces',
    list: (ctx) => ({ url: dbApi(ctx, 'tables', ctx.table, 'columns'), headers: profile(ctx.schema) }),
    key: (c) => c.name,
    title: (c) => c.name,
    columns: [
        { key: 'name', label: 'name', mono: true },
        { key: 'type', label: 'type', mono: true, text: (c) => c.typedef ?? typeLabel(c.type ?? '') },
        { key: 'notnull', label: 'not null', flag: true },
        { key: 'default', label: 'default', mono: true },
        { key: 'constraints', label: 'constraints', mono: true, text: (c) => list(c.constraints) },
        { key: 'comment', label: 'comment' },
    ],
    fields: [
        { key: 'name', label: 'Name', kind: 'text', required: true },
        { key: 'type', label: 'Type', kind: 'type', required: true },
        { key: 'notnull', label: 'Not null', kind: 'bool' },
        { key: 'default', label: 'Default', kind: 'sql', placeholder: 'none', help: 'An SQL expression: 0, now(), gen_random_uuid()' },
        { key: 'comment', label: 'Comment', kind: 'text' },
        { key: 'constraints', label: 'Constraints', kind: 'text', readonly: true },
    ],
    initial: () => ({ type: 'text', notnull: false }),
    create: (ctx, v) => {
        const body = pick(v, ['name', 'type', 'notnull', 'comment']);
        if (v.default) body.default = v.default;
        return { url: dbApi(ctx, 'tables', ctx.table, 'columns'), headers: profile(ctx.schema), body };
    },
    // an empty default drops it
    update: (ctx, c, ch) => ({
        url: dbApi(ctx, 'tables', ctx.table, 'columns', c.name),
        headers: profile(ctx.schema),
        body: pick(ch, ['name', 'type', 'notnull', 'default', 'comment']),
    }),
    remove: (ctx, c) => ({ url: dbApi(ctx, 'tables', ctx.table, 'columns', c.name), headers: profile(ctx.schema) }),
    removeWarning: (c) => `The column ${c.name} and its values are removed from every row. You cannot undo this.`,
};

// An SQL identifier: a plain lower-case name as is, any other name quoted.
export const ident = (name: string) => (/^[a-z_][a-z0-9_]*$/.test(name) ? name : `"${name.replace(/"/g, '""')}"`);
// "schema.table" from the table field, quoted part by part
const qualified = (st: string) => {
    const dot = st.indexOf('.');
    return dot < 0 ? ident(st) : `${ident(st.slice(0, dot))}.${ident(st.slice(dot + 1))}`;
};
const idents = (cols: unknown) => (Array.isArray(cols) ? cols.map(ident).join(', ') : '');

const fkActions = ['NO ACTION', 'CASCADE', 'SET NULL', 'SET DEFAULT', 'RESTRICT'];
const isType = (...types: string[]) => (v: Item) => types.includes(v.type);

// The definition of a constraint from the guided fields.
export function constraintDefinition(v: Item): string {
    switch (v.type) {
        case 'PRIMARY KEY':
        case 'UNIQUE':
            return `${v.type} (${idents(v.columns)})`;
        case 'CHECK':
            return `CHECK (${v.expression})`;
        case 'FOREIGN KEY': {
            let def = `FOREIGN KEY (${idents(v.columns)}) REFERENCES ${qualified(v.reftable)} (${idents(v.refcolumns)})`;
            if (v.ondelete && v.ondelete !== 'NO ACTION') def += ` ON DELETE ${v.ondelete}`;
            if (v.onupdate && v.onupdate !== 'NO ACTION') def += ` ON UPDATE ${v.onupdate}`;
            return def;
        }
    }
    return '';
}

const constraints: EntityDef = {
    kind: 'constraints',
    singular: 'constraint',
    plural: 'Constraints',
    icon: 'key-2',
    list: (ctx) => ({ url: dbApi(ctx, 'tables', ctx.table, 'constraints'), headers: profile(ctx.schema) }),
    key: (c) => c.name,
    title: (c) => c.name,
    columns: [
        { key: 'name', label: 'name', mono: true },
        { key: 'type', label: 'type' },
        { key: 'columns', label: 'columns', mono: true, text: (c) => list(c.columns) },
        { key: 'definition', label: 'definition', mono: true },
    ],
    fields: [
        { key: 'type', label: 'Type', kind: 'select', createOnly: true, required: true, options: ['PRIMARY KEY', 'UNIQUE', 'CHECK', 'FOREIGN KEY'] },
        { key: 'name', label: 'Name', kind: 'text', createOnly: true, placeholder: 'Generated' },
        { key: 'columns', label: 'Columns', kind: 'columns', createOnly: true, required: true, when: (v) => v.type !== 'CHECK' },
        {
            key: 'expression',
            label: 'Check',
            kind: 'sql',
            createOnly: true,
            required: true,
            multiline: true,
            placeholder: 'price >= 0',
            help: 'A boolean expression on the columns of the row',
            when: isType('CHECK'),
        },
        { key: 'reftable', label: 'References', kind: 'table', createOnly: true, required: true, when: isType('FOREIGN KEY') },
        { key: 'refcolumns', label: 'Columns of it', kind: 'refcolumns', from: 'reftable', createOnly: true, required: true, when: isType('FOREIGN KEY') },
        { key: 'ondelete', label: 'On delete', kind: 'select', createOnly: true, options: fkActions, when: isType('FOREIGN KEY') },
        { key: 'onupdate', label: 'On update', kind: 'select', createOnly: true, options: fkActions, when: isType('FOREIGN KEY') },
        { key: 'definition', label: 'Definition', kind: 'sql', readonly: true },
    ],
    initial: () => ({ type: 'PRIMARY KEY', columns: [], refcolumns: [], ondelete: 'NO ACTION', onupdate: 'NO ACTION' }),
    create: (ctx, v) => ({
        url: dbApi(ctx, 'tables', ctx.table, 'constraints'),
        headers: profile(ctx.schema),
        body: { ...(v.name ? { name: v.name } : {}), definition: constraintDefinition(v) },
    }),
    remove: (ctx, c) => ({ url: dbApi(ctx, 'tables', ctx.table, 'constraints', c.name), headers: profile(ctx.schema) }),
    removeWarning: (c) => `The constraint ${c.name} is dropped. The rows are not checked against it any more.`,
};

const policies: EntityDef = {
    kind: 'policies',
    singular: 'policy',
    plural: 'Policies',
    icon: 'shield-keyhole',
    list: (ctx) => ({ url: dbApi(ctx, 'tables', ctx.table, 'policies'), headers: profile(ctx.schema) }),
    key: (p) => p.name,
    title: (p) => p.name,
    columns: [
        { key: 'name', label: 'name', mono: true },
        { key: 'command', label: 'command' },
        { key: 'roles', label: 'roles', mono: true, text: (p) => list(p.roles) },
        { key: 'using', label: 'using', mono: true },
        { key: 'check', label: 'with check', mono: true },
        { key: 'deny', label: 'restrictive', flag: true },
    ],
    fields: [
        { key: 'name', label: 'Name', kind: 'text', required: true },
        { key: 'command', label: 'Command', kind: 'select', createOnly: true, options: ['ALL', 'SELECT', 'INSERT', 'UPDATE', 'DELETE'] },
        { key: 'deny', label: 'Restrictive', kind: 'bool', createOnly: true, help: 'A restrictive policy must pass together with the permissive ones' },
        { key: 'roles', label: 'Roles', kind: 'roles', help: 'Empty means PUBLIC' },
        { key: 'using', label: 'Using', kind: 'sql', multiline: true, placeholder: 'owner = current_user', help: 'The rows the command sees' },
        { key: 'check', label: 'With check', kind: 'sql', multiline: true, placeholder: 'owner = current_user', help: 'The rows the command may write' },
    ],
    initial: () => ({ command: 'ALL', deny: false, roles: [] }),
    create: (ctx, v) => {
        const body = pick(v, ['name', 'command', 'deny', 'roles']);
        if (v.using) body.using = v.using;
        if (v.check) body.check = v.check;
        return { url: dbApi(ctx, 'tables', ctx.table, 'policies'), headers: profile(ctx.schema), body };
    },
    update: (ctx, p, c) => ({
        url: dbApi(ctx, 'tables', ctx.table, 'policies', p.name),
        headers: profile(ctx.schema),
        body: pick(c, ['name', 'roles', 'using', 'check']),
    }),
    remove: (ctx, p) => ({ url: dbApi(ctx, 'tables', ctx.table, 'policies', p.name), headers: profile(ctx.schema) }),
    removeWarning: (p) => `The policy ${p.name} is dropped. The rows it allowed may become invisible or writable.`,
};

const views: EntityDef = {
    kind: 'views',
    singular: 'view',
    plural: 'Views',
    icon: 'eye',
    list: (ctx) => ({ url: dbApi(ctx, 'views') }),
    key: (v) => `${v.schema}.${v.name}`,
    title: (v) => v.name,
    columns: [
        { key: 'name', label: 'name', mono: true },
        { key: 'schema', label: 'schema', mono: true },
        { key: 'owner', label: 'owner', mono: true },
        { key: 'materialized', label: 'materialized', flag: true },
        { key: 'definition', label: 'definition', mono: true, text: (v) => (v.definition ?? '').replace(/\s+/g, ' ').trim() },
    ],
    fields: [
        { key: 'name', label: 'Name', kind: 'text', required: true },
        { key: 'schema', label: 'Schema', kind: 'schema' },
        { key: 'owner', label: 'Owner', kind: 'role' },
        { key: 'materialized', label: 'Materialized', kind: 'bool', createOnly: true },
        {
            key: 'definition',
            label: 'Definition',
            kind: 'sql',
            multiline: true,
            required: true,
            placeholder: 'SELECT ...',
            help: 'A new definition must keep the columns of the old one, in order; it may add columns at the end',
            // a materialized view has no CREATE OR REPLACE
            createOnlyWhen: (v) => !!v.materialized,
        },
    ],
    initial: (ctx) => ({ schema: ctx.schema || 'public', materialized: false }),
    create: (ctx, v) => ({ url: dbApi(ctx, 'views'), headers: profile(v.schema), body: pick(v, ['name', 'schema', 'definition', 'materialized']) }),
    update: (ctx, v, c) => {
        const out: Request[] = [];
        // the definition first, under the old name and schema
        if ('definition' in c) {
            out.push({
                url: dbApi(ctx, 'views'),
                headers: profile(v.schema),
                body: { name: v.name, schema: v.schema, definition: c.definition, replace: true },
            });
        }
        const rest = pick(c, ['name', 'owner', 'schema']);
        if (Object.keys(rest).length) out.push({ url: dbApi(ctx, 'views', v.name), headers: profile(v.schema), body: rest });
        return out;
    },
    remove: (ctx, v) => ({ url: dbApi(ctx, 'views', v.name), headers: profile(v.schema) }),
    removeWarning: (v) => `The ${v.materialized ? 'materialized view' : 'view'} ${v.schema}.${v.name} is dropped. Its data stay in the tables it reads.`,
    data: (ctx, v) => uiPath('data', ctx.db, v.schema, v.name),
};

// The arguments of a function as text, "a int4, out total numeric", and back.
const argModes = ['in', 'out', 'inout', 'variadic'];

export function formatArgs(args: Item[] | undefined): string {
    return (args ?? [])
        .filter((a) => a && a.type && a.mode !== 'table')
        .map((a) => [a.mode && a.mode !== 'in' ? a.mode : '', a.name, a.type].filter(Boolean).join(' '))
        .join(', ');
}

export function parseArgs(text: string): Item[] {
    const out: Item[] = [];
    let depth = 0;
    let cur = '';
    const parts: string[] = [];
    for (const ch of text) {
        if (ch === '(') depth++;
        if (ch === ')') depth--;
        if (ch === ',' && depth === 0) {
            parts.push(cur);
            cur = '';
        } else cur += ch;
    }
    parts.push(cur);
    for (const part of parts.map((p) => p.trim()).filter(Boolean)) {
        const words = part.split(/\s+/);
        let mode = 'in';
        if (argModes.includes(words[0].toLowerCase()) && words.length > 1) mode = words.shift()!.toLowerCase();
        // "name type ..." when there are two words or more; a lone word is a type
        const [name, type] = words.length > 1 ? [words[0], words.slice(1).join(' ')] : ['', words[0]];
        out.push({ name, type, mode });
    }
    return out;
}

// the types of the input arguments: the signature that addresses an overload
const argTypes = (f: Item) =>
    (f.arguments ?? [])
        .filter((a: Item) => a && a.type && a.mode !== 'out' && a.mode !== 'table')
        .map((a: Item) => a.type)
        .join(',');

const volatilityWords: Record<string, string> = { i: 'immutable', s: 'stable', v: 'volatile' };
const volatility = (f: Item) => volatilityWords[f.volatility] ?? f.volatility ?? '';

const functionFields = ['name', 'schema', 'returns', 'language', 'definition', 'volatility', 'strict', 'security'];

function functionBody(v: Item): Item {
    const body = pick(v, functionFields);
    body.arguments = parseArgs(v.arguments ?? '');
    return body;
}

const functions: EntityDef = {
    kind: 'functions',
    singular: 'function',
    plural: 'Functions',
    icon: 'function',
    list: (ctx) => ({ url: dbApi(ctx, 'functions') }),
    key: (f) => `${f.schema}.${f.name}(${typeof f.arguments === 'string' ? parseArgs(f.arguments).map((a) => a.type).join(',') : argTypes(f)})`,
    title: (f) => `${f.name}(${argTypes(f)})`,
    columns: [
        { key: 'name', label: 'name', mono: true },
        { key: 'schema', label: 'schema', mono: true },
        { key: 'arguments', label: 'arguments', mono: true, text: (f) => formatArgs(f.arguments) },
        { key: 'returns', label: 'returns', mono: true },
        { key: 'language', label: 'language' },
        { key: 'volatility', label: 'volatility', text: volatility },
    ],
    fields: [
        { key: 'name', label: 'Name', kind: 'text', required: true, createOnly: true },
        { key: 'schema', label: 'Schema', kind: 'schema', createOnly: true },
        {
            key: 'arguments',
            label: 'Arguments',
            kind: 'args',
            createOnly: true,
            placeholder: 'a int4, b text',
            help: 'Comma-separated: [in|out|inout|variadic] name type',
            read: (f) => formatArgs(f.arguments),
        },
        { key: 'returns', label: 'Returns', kind: 'type', required: true, createOnly: true, placeholder: 'setof t, table (a int, b text)', options: ['void', 'integer', 'bigint', 'numeric', 'text', 'boolean', 'date', 'timestamptz', 'jsonb', 'trigger', 'setof record'] },
        { key: 'language', label: 'Language', kind: 'select', options: ['sql', 'plpgsql'] },
        { key: 'volatility', label: 'Volatility', kind: 'select', options: ['volatile', 'stable', 'immutable'], read: volatility },
        { key: 'strict', label: 'Strict', kind: 'bool', help: 'Returns NULL when an argument is NULL, without a call' },
        { key: 'security', label: 'Security', kind: 'select', options: ['invoker', 'definer'] },
        { key: 'definition', label: 'Body', kind: 'sql', multiline: true, required: true, placeholder: 'SELECT a + b' },
    ],
    initial: (ctx) => ({ schema: ctx.schema || 'public', returns: 'void', language: 'sql', volatility: 'volatile', security: 'invoker', strict: false, arguments: '' }),
    create: (ctx, v) => ({ url: dbApi(ctx, 'functions'), headers: profile(v.schema), body: functionBody(v) }),
    // CREATE OR REPLACE with the same signature: the whole function goes again
    update: (ctx, f, c) => {
        const v = { ...f, arguments: formatArgs(f.arguments), volatility: volatility(f), ...c };
        return { url: dbApi(ctx, 'functions'), headers: profile(f.schema), body: { ...functionBody(v), replace: true } };
    },
    remove: (ctx, f) => ({
        url: `${dbApi(ctx, 'functions', f.name)}?args=${encodeURIComponent(argTypes(f))}`,
        headers: profile(f.schema),
    }),
    removeWarning: (f) => `The function ${f.schema}.${f.name}(${argTypes(f)}) is dropped. Code that calls it fails.`,
};

// Grants: a grant has no name; its key is the grantee with the privileges.
// A grant does not change: a new grant adds privileges, a revoke removes them.
const grantColumns: ListColumn[] = [
    { key: 'grantee', label: 'grantee', mono: true, text: (g) => g.grantee || 'PUBLIC' },
    { key: 'types', label: 'privileges', text: (g) => list(g.types) },
    { key: 'columns', label: 'columns', mono: true, text: (g) => list(g.columns) },
    { key: 'grantor', label: 'grantor', mono: true },
];

const grantee = (g: Item) => (g.grantee ? g.grantee : 'public');

// the grant routes address a table as schema.table
const tableTarget = (ctx: AdminContext) => (ctx.schema ? `${ctx.schema}.${ctx.table}` : ctx.table);

const tableGrants: EntityDef = {
    kind: 'tablegrants',
    singular: 'grant',
    plural: 'Grants',
    icon: 'shield-keyhole',
    list: (ctx) => ({ url: dbApi(ctx, 'table', tableTarget(ctx), 'grants') }),
    key: (g) => `${grantee(g)}|${list(g.types)}|${list(g.columns)}`,
    title: (g) => g.grantee || 'PUBLIC',
    columns: grantColumns,
    fields: [
        { key: 'grantee', label: 'Grantee', kind: 'role', createOnly: true, emptyLabel: 'PUBLIC (all roles)' },
        {
            key: 'types',
            label: 'Privileges',
            kind: 'multi',
            createOnly: true,
            required: true,
            options: ['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'TRUNCATE', 'REFERENCES', 'TRIGGER'],
        },
        {
            key: 'columns',
            label: 'Columns',
            kind: 'columns',
            createOnly: true,
            help: 'Only SELECT, INSERT, UPDATE and REFERENCES take columns; empty means the whole table',
        },
        { key: 'grantor', label: 'Grantor', kind: 'text', readonly: true },
    ],
    initial: () => ({ grantee: '', types: ['SELECT'], columns: [] }),
    create: (ctx, v) => ({
        url: api('grants', ctx.db, 'table', tableTarget(ctx)),
        body: { grantee: v.grantee || 'public', types: v.types, ...(v.columns?.length ? { columns: v.columns } : {}) },
    }),
    remove: (ctx, g) => ({
        url: api('grants', ctx.db, 'table', tableTarget(ctx)),
        body: { grantee: grantee(g), types: g.types, ...(g.columns?.length ? { columns: g.columns } : {}) },
    }),
    removeWarning: (g) =>
        `${g.grantee || 'PUBLIC'} loses ${list(g.types)} on ${g.columns?.length ? `the columns ${list(g.columns)} of ` : ''}this table.`,
    removeVerb: 'Revoke',
};

const dbGrants: EntityDef = {
    kind: 'dbgrants',
    singular: 'grant',
    plural: 'Grants',
    icon: 'shield-keyhole',
    list: (ctx) => ({ url: dbApi(ctx, 'grants') }),
    key: (g) => `${grantee(g)}|${list(g.types)}`,
    title: (g) => g.grantee || 'PUBLIC',
    columns: grantColumns.filter((c) => c.key !== 'columns'),
    fields: [
        { key: 'grantee', label: 'Grantee', kind: 'role', createOnly: true, emptyLabel: 'PUBLIC (all roles)' },
        { key: 'types', label: 'Privileges', kind: 'multi', createOnly: true, required: true, options: ['CONNECT', 'CREATE', 'TEMPORARY'] },
        { key: 'grantor', label: 'Grantor', kind: 'text', readonly: true },
    ],
    initial: () => ({ grantee: '', types: ['CONNECT'] }),
    create: (ctx, v) => ({ url: api('grants', ctx.db), body: { targettype: 'database', targetname: ctx.db, grantee: v.grantee || 'public', types: v.types } }),
    remove: (ctx, g) => ({ url: api('grants', ctx.db), body: { targettype: 'database', targetname: ctx.db, grantee: grantee(g), types: g.types } }),
    removeWarning: (g) => `${g.grantee || 'PUBLIC'} loses ${list(g.types)} on this database.`,
    removeVerb: 'Revoke',
};

export const entities: Record<string, EntityDef> = {
    databases,
    roles,
    schemas,
    tables,
    columns,
    constraints,
    policies,
    views,
    functions,
    tablegrants: tableGrants,
    dbgrants: dbGrants,
};

// The sections of a database and of a table, as tabs of the admin page.
export const dbSections = [
    ['tables', 'Tables'],
    ['views', 'Views'],
    ['functions', 'Functions'],
    ['schemas', 'Schemas'],
    ['grants', 'Grants'],
] as const;

export const tableSections = [
    ['columns', 'Columns'],
    ['constraints', 'Constraints'],
    ['policies', 'Policies'],
    ['grants', 'Grants'],
] as const;

