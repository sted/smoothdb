// One representation of a column value for the whole Data area: the grid, the
// record panel in reading mode and the fields in editing mode.

import { checkNumber, columnKind, type Kind } from './coltypes';
import type { Column } from './data.svelte';

// The text of a value. A JSON or array column shows JSON, so that the JSON
// string "[123]" and the array [123] look different; any other text shows as is.
export function displayValue(value: unknown, kind: Kind): string {
    if (value === null || value === undefined) return 'NULL';
    if (kind === 'json' || kind === 'array') return JSON.stringify(value);
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
}

// How a column is edited: arrays are edited as JSON text.
export type EditKind = 'int' | 'float' | 'numeric' | 'bool' | 'json' | 'date' | 'timestamp' | 'time' | 'text';

export function editKind(type: string): EditKind {
    const k = columnKind(type);
    if (k === 'array') return 'json';
    if (k === 'uuid' || k === 'any') return 'text';
    return k;
}

// A timestamp or a time as the native picker edits it, and the time zone
// that the picker cannot hold. The picker keeps milliseconds: a value with
// more digits loses them only when the user changes the field.
export interface Temporal {
    local: string;
    // "Z", "+01", "+05:30" or ""
    zone: string;
}

const timestampRe = /^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})(:\d{2})?(\.\d+)?(Z|[+-]\d{2}(?::?\d{2})?)?$/;
const timeRe = /^(\d{2}:\d{2})(:\d{2})?(\.\d+)?(Z|[+-]\d{2}(?::?\d{2})?)?$/;

export function splitTemporal(value: unknown, ek: EditKind): Temporal | null {
    if (typeof value !== 'string') return null;
    if (ek === 'timestamp') {
        const m = value.match(timestampRe);
        if (!m) return null;
        return { local: `${m[1]}T${m[2]}${m[3] ?? ':00'}${m[4] ? m[4].slice(0, 4) : ''}`, zone: m[5] ?? '' };
    }
    if (ek === 'time') {
        const m = value.match(timeRe);
        if (!m) return null;
        return { local: `${m[1]}${m[2] ?? ':00'}${m[3] ? m[3].slice(0, 4) : ''}`, zone: m[4] ?? '' };
    }
    return null;
}

// The label of a time zone next to a picker.
export const zoneLabel = (zone: string) => (zone === 'Z' || zone === '+00' || zone === '+00:00' ? 'UTC' : zone);

// The text of a field for a stored value.
export function toInput(value: unknown, ek: EditKind): string {
    if (value === null || value === undefined) return '';
    if (ek === 'bool') return value ? 'true' : 'false';
    if (ek === 'timestamp' || ek === 'time') return splitTemporal(value, ek)?.local ?? String(value);
    // compact JSON: it reads on the one line of a field without the focus
    if (ek === 'json') return JSON.stringify(value);
    return String(value);
}

// The value to store for the text of a field. Throws an Error with a message
// for the user when the text is not valid for the column.
export function fromInput(text: string, isNull: boolean, ek: EditKind, zone = ''): unknown {
    if (isNull) return null;
    const v = text.trim();
    switch (ek) {
        case 'bool':
            return v === '' ? null : v === 'true';
        case 'int': {
            if (v === '') return null;
            const bad = checkNumber('int', v);
            if (bad) throw new Error(bad);
            // a value beyond 2^53 stays a string, so that no digit is lost
            return Number.isSafeInteger(Number(v)) ? Number(v) : v;
        }
        case 'float':
            if (v === '') return null;
            if (Number.isNaN(Number(v))) throw new Error('Enter a number.');
            return Number(v);
        case 'numeric':
            if (v === '') return null;
            if (checkNumber('numeric', v)) throw new Error('Enter a number.');
            // numeric keeps every digit as a string; PostgreSQL casts it
            return v;
        case 'json':
            if (v === '') return null;
            try {
                return JSON.parse(text);
            } catch (e) {
                throw new Error(`Invalid JSON: ${e instanceof Error ? e.message : e}`);
            }
        case 'date':
            return v === '' ? null : v;
        case 'timestamp':
        case 'time':
            // the zone of the stored value goes back with it
            return v === '' ? null : v + zone;
        default:
            return text;
    }
}

export const sameValue = (a: unknown, b: unknown) => JSON.stringify(a ?? null) === JSON.stringify(b ?? null);

// The placeholder of a field: what the database does with an unset column.
export function placeholderFor(c: Column, ek: EditKind, isNew: boolean, isNull: boolean): string {
    if (isNull) return 'NULL';
    if (isNew && c.default !== null) return c.default.startsWith('nextval(') ? 'Generated' : `Default: ${c.default}`;
    if (ek === 'json') return c.type.startsWith('_') ? '["a", "b"]' : '{"key": "value"}';
    return c.notnull ? 'Required' : 'Empty';
}
