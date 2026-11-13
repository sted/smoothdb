// Typed filters of the Data area, and their PostgREST form.
//
// A filter is a column, an operator chosen from the operators of the column
// kind, and a value as the user writes it. In the URL of the UI the filters
// are the PostgREST query parameters themselves: column=op.value for "all",
// or=(column.op.value,...) for "any". A parameter the UI cannot read back is
// kept as a raw filter and shown as written.

import { checkNumber, isNumeric, type Kind } from './coltypes';

export type OpId =
    | 'eq'
    | 'neq'
    | 'gt'
    | 'gte'
    | 'lt'
    | 'lte'
    | 'contains'
    | 'not_contains'
    | 'starts'
    | 'ends'
    | 'in'
    | 'is_null'
    | 'not_null'
    | 'is_true'
    | 'is_false'
    | 'has'
    | 'overlaps'
    | 'json_contains'
    | 'raw';

export interface Filter {
    column: string;
    op: OpId;
    // the value as the user writes it; a list is "a, b, c"; for raw, the whole PostgREST expression
    value: string;
}

export interface FilterState {
    mode: 'and' | 'or';
    filters: Filter[];
}

// none: no value; scalar: one value; list: comma-separated values; json: a JSON document
export type ValueShape = 'none' | 'scalar' | 'list' | 'json';

const shapes: Record<OpId, ValueShape> = {
    eq: 'scalar',
    neq: 'scalar',
    gt: 'scalar',
    gte: 'scalar',
    lt: 'scalar',
    lte: 'scalar',
    contains: 'scalar',
    not_contains: 'scalar',
    starts: 'scalar',
    ends: 'scalar',
    in: 'list',
    is_null: 'none',
    not_null: 'none',
    is_true: 'none',
    is_false: 'none',
    has: 'list',
    overlaps: 'list',
    json_contains: 'json',
    raw: 'scalar',
};

export const valueShape = (op: OpId) => shapes[op];

const timeKinds = new Set<Kind>(['date', 'timestamp', 'time']);

// The operators of a column kind, in the order the menu shows them.
export function operatorsFor(kind: Kind, nullable = true): OpId[] {
    let ops: OpId[];
    if (isNumeric(kind)) ops = ['eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'in'];
    else if (timeKinds.has(kind)) ops = ['eq', 'neq', 'gt', 'gte', 'lt', 'lte'];
    else if (kind === 'bool') ops = ['is_true', 'is_false'];
    else if (kind === 'json') ops = ['json_contains'];
    else if (kind === 'array') ops = ['has', 'overlaps'];
    else if (kind === 'uuid') ops = ['eq', 'neq', 'in'];
    else if (kind === 'text') ops = ['contains', 'not_contains', 'starts', 'ends', 'eq', 'neq', 'in'];
    else ops = ['eq', 'neq', 'contains', 'starts', 'ends', 'gt', 'gte', 'lt', 'lte', 'in'];
    return nullable ? [...ops, 'is_null', 'not_null'] : ops;
}

const labels: Record<OpId, string> = {
    eq: '=',
    neq: '≠',
    gt: '>',
    gte: '≥',
    lt: '<',
    lte: '≤',
    contains: 'contains',
    not_contains: 'does not contain',
    starts: 'starts with',
    ends: 'ends with',
    in: 'is one of',
    is_null: 'is null',
    not_null: 'is not null',
    is_true: 'is true',
    is_false: 'is false',
    has: 'contains all of',
    overlaps: 'contains any of',
    json_contains: 'contains',
    raw: '',
};

const timeLabels: Partial<Record<OpId, string>> = {
    eq: 'is',
    neq: 'is not',
    gt: 'is after',
    gte: 'is on or after',
    lt: 'is before',
    lte: 'is on or before',
};

export function operatorLabel(op: OpId, kind: Kind): string {
    return (timeKinds.has(kind) && timeLabels[op]) || labels[op];
}

// ---- values in the PostgREST syntax

// A value inside or=(...) or in.(...) is quoted when it holds a character
// that the grammar reserves.
const reserved = /[,()"\\:]|^\s|\s$|^$/;

function quote(v: string): string {
    return `"${v.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`;
}

const quoteIf = (v: string) => (reserved.test(v) ? quote(v) : v);

function unquote(v: string): string {
    if (v.length >= 2 && v.startsWith('"') && v.endsWith('"')) return v.slice(1, -1).replace(/\\(.)/g, '$1');
    return v;
}

// An element of a PostgreSQL array literal, {a,"b c"}.
const arrayItem = (v: string) => (/[,{}"\\\s]|^$|^null$/i.test(v) ? quote(v) : v);

// splitTop splits at the commas that are outside quotes, parentheses and braces.
function splitTop(s: string): string[] {
    const out: string[] = [];
    let depth = 0;
    let inQuote = false;
    let cur = '';
    for (let i = 0; i < s.length; i++) {
        const c = s[i];
        if (inQuote) {
            cur += c;
            if (c === '\\' && i + 1 < s.length) cur += s[++i];
            else if (c === '"') inQuote = false;
            continue;
        }
        if (c === '"') inQuote = true;
        else if (c === '(' || c === '{') depth++;
        else if (c === ')' || c === '}') depth--;
        else if (c === ',' && depth === 0) {
            out.push(cur);
            cur = '';
            continue;
        }
        cur += c;
    }
    out.push(cur);
    return out;
}

export const splitList = (v: string) =>
    v
        .split(',')
        .map((s) => s.trim())
        .filter((s) => s !== '');

// expression returns the PostgREST expression of a filter, "gte.10". inLogic
// quotes the value for a place inside or=(...).
export function expression(f: Filter, inLogic = false): string {
    const val = (v: string) => (inLogic ? quoteIf(v) : v);
    switch (f.op) {
        case 'eq':
        case 'neq':
        case 'gt':
        case 'gte':
        case 'lt':
        case 'lte':
            return `${f.op}.${val(f.value)}`;
        case 'contains':
            return `ilike.${val(`*${f.value}*`)}`;
        case 'not_contains':
            return `not.ilike.${val(`*${f.value}*`)}`;
        case 'starts':
            return `ilike.${val(`${f.value}*`)}`;
        case 'ends':
            return `ilike.${val(`*${f.value}`)}`;
        case 'in':
            return `in.(${splitList(f.value).map(quoteIf).join(',')})`;
        case 'is_null':
            return 'is.null';
        case 'not_null':
            return 'not.is.null';
        case 'is_true':
            return 'is.true';
        case 'is_false':
            return 'is.false';
        case 'has':
        case 'overlaps': {
            const literal = `{${splitList(f.value).map(arrayItem).join(',')}}`;
            return `${f.op === 'has' ? 'cs' : 'ov'}.${val(literal)}`;
        }
        case 'json_contains':
            return `cs.${val(JSON.stringify(JSON.parse(f.value)))}`;
        case 'raw':
            return f.value;
    }
}

// parseExpression reads a PostgREST expression back into an operator and a value.
export function parseExpression(expr: string, kind: Kind): { op: OpId; value: string } {
    const raw = { op: 'raw' as OpId, value: expr };
    let e = expr;
    let negated = false;
    if (e.startsWith('not.')) {
        negated = true;
        e = e.slice(4);
    }
    const dot = e.indexOf('.');
    if (dot < 0) return raw;
    const op = e.slice(0, dot);
    const v = unquote(e.slice(dot + 1));
    if (op === 'is') {
        if (v === 'null') return { op: negated ? 'not_null' : 'is_null', value: '' };
        if (!negated && v === 'true') return { op: 'is_true', value: '' };
        if (!negated && v === 'false') return { op: 'is_false', value: '' };
        return raw;
    }
    if (op === 'ilike') {
        let m = v.match(/^\*(.*)\*$/);
        if (m) return { op: negated ? 'not_contains' : 'contains', value: m[1] };
        if (negated) return raw;
        if ((m = v.match(/^\*(.*)$/))) return { op: 'ends', value: m[1] };
        if ((m = v.match(/^(.*)\*$/))) return { op: 'starts', value: m[1] };
        return raw;
    }
    if (negated) return raw;
    if (['eq', 'neq', 'gt', 'gte', 'lt', 'lte'].includes(op)) return { op: op as OpId, value: v };
    if (op === 'in') {
        const m = v.match(/^\((.*)\)$/);
        return m ? { op: 'in', value: splitTop(m[1]).map(unquote).join(', ') } : raw;
    }
    if (op === 'cs' && kind === 'json') return { op: 'json_contains', value: v };
    if (op === 'cs' || op === 'ov') {
        const m = v.match(/^\{(.*)\}$/);
        return m ? { op: op === 'cs' ? 'has' : 'overlaps', value: splitTop(m[1]).map(unquote).join(', ') } : raw;
    }
    return raw;
}

// checkValue returns an error message for a value the operator cannot use.
export function checkValue(f: Filter, kind: Kind): string | null {
    const shape = valueShape(f.op);
    if (shape === 'none' || f.op === 'raw') return null;
    if (shape === 'json') {
        try {
            JSON.parse(f.value);
            return null;
        } catch {
            return 'Enter a JSON document.';
        }
    }
    if (shape === 'list') {
        const items = splitList(f.value);
        if (items.length === 0) return 'Enter one or more values, separated by commas.';
        if (isNumeric(kind)) for (const i of items) if (checkNumber(kind, i)) return checkNumber(kind, i);
        return null;
    }
    if (f.value.trim() === '' && kind !== 'text' && kind !== 'any') return 'Enter a value.';
    if (isNumeric(kind) && f.op !== 'contains') return checkNumber(kind, f.value);
    return null;
}

// ---- the URL

// the query parameters of the UI and of the request; every other one is a filter
export const reservedParams = new Set(['page', 'order', 'select', 'limit', 'offset', 'cols']);

// readFilters reads the filters from the query parameters of the UI.
export function readFilters(params: URLSearchParams, kindOf: (column: string) => Kind): FilterState {
    const plain: Filter[] = [];
    const groups: [string, string][] = [];
    for (const [k, v] of params) {
        if (reservedParams.has(k)) continue;
        if (k === 'or' || k === 'and' || k.startsWith('not.')) groups.push([k, v]);
        else plain.push({ column: k, ...parseExpression(v, kindOf(k)) });
    }
    // one or=(...) group of simple conditions, and nothing else: the "any" mode
    if (groups.length === 1 && groups[0][0] === 'or' && plain.length === 0) {
        const m = groups[0][1].match(/^\((.*)\)$/);
        const items = m ? splitTop(m[1]) : [];
        const filters: Filter[] = [];
        for (const item of items) {
            const dot = item.indexOf('.');
            if (dot <= 0 || /^(and|or|not)\(/.test(item)) break;
            const column = item.slice(0, dot);
            filters.push({ column, ...parseExpression(item.slice(dot + 1), kindOf(column)) });
        }
        if (filters.length === items.length && filters.length > 0) return { mode: 'or', filters };
    }
    // any other group stays as written
    for (const [k, v] of groups) plain.push({ column: k, op: 'raw', value: v });
    return { mode: 'and', filters: plain };
}

// writeFilters replaces the filters in the query parameters of the UI.
export function writeFilters(params: URLSearchParams, state: FilterState) {
    for (const k of [...params.keys()]) if (!reservedParams.has(k)) params.delete(k);
    params.delete('page');
    if (state.mode === 'or' && state.filters.length > 1) {
        params.set('or', `(${state.filters.map((f) => `${f.column}.${expression(f, true)}`).join(',')})`);
    } else {
        for (const f of state.filters) params.append(f.column, expression(f));
    }
}

// a group written by hand cannot go inside or=(...)
export const canUseAny = (state: FilterState) =>
    state.filters.every((f) => !(f.op === 'raw' && (f.column === 'or' || f.column === 'and' || f.column.startsWith('not.'))));
