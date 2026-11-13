// The kinds of PostgreSQL column types that the UI edits and filters in
// different ways. The type names are those of pg_type (int4, _text, ...), as
// $info and the admin API report them.

export type Kind =
    | 'int'
    | 'float'
    | 'numeric'
    | 'bool'
    | 'json'
    | 'array'
    | 'date'
    | 'timestamp'
    | 'time'
    | 'uuid'
    | 'text'
    // a column with no known type: the columns of a view
    | 'any';

const kinds: Record<string, Kind> = {
    int2: 'int',
    int4: 'int',
    int8: 'int',
    oid: 'int',
    float4: 'float',
    float8: 'float',
    numeric: 'numeric',
    money: 'numeric',
    bool: 'bool',
    json: 'json',
    jsonb: 'json',
    date: 'date',
    timestamp: 'timestamp',
    timestamptz: 'timestamp',
    time: 'time',
    timetz: 'time',
    uuid: 'uuid',
};

export function columnKind(type: string | undefined): Kind {
    if (!type) return 'any';
    if (type.startsWith('_')) return 'array';
    return kinds[type] ?? 'text';
}

export const isNumeric = (k: Kind) => k === 'int' || k === 'float' || k === 'numeric';

// The type as people write it: _int4 is int4[].
export const typeLabel = (type: string) => (type.startsWith('_') ? type.slice(1) + '[]' : type);

// checkNumber returns an error message for a value that is not a number of the kind.
export function checkNumber(kind: Kind, v: string): string | null {
    const s = v.trim();
    if (kind === 'int' && !/^-?\d+$/.test(s)) return 'Enter a whole number.';
    if ((kind === 'float' || kind === 'numeric') && !/^-?(\d+\.?\d*|\.\d+)(e[+-]?\d+)?$/i.test(s)) return 'Enter a number.';
    return null;
}

const dateRe = /^\d{4}-\d{2}-\d{2}$/;
const timestampRe = /^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}(:?\d{2})?)?$/;
const timeRe = /^\d{2}:\d{2}(:\d{2}(\.\d+)?)?([+-]\d{2}(:?\d{2})?)?$/;
const uuidRe = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
// a numeric whose text the UI keeps ("210.30"), see parseJSON in api.ts
const numberTextRe = /^-?\d+(\.\d+)?(e[+-]?\d+)?$/i;

// inferKind guesses the kind of a column with no known type, a column of a
// view, from its values in the rows on screen. Every value that is not null
// must agree; otherwise the kind stays 'any'.
export function inferKind(values: unknown[]): Kind {
    const seen = values.filter((v) => v !== null && v !== undefined);
    if (seen.length === 0) return 'any';
    const all = (test: (v: unknown) => boolean) => seen.every(test);
    if (all((v) => typeof v === 'boolean')) return 'bool';
    if (all((v) => typeof v === 'number' && Number.isInteger(v))) return 'int';
    if (all((v) => typeof v === 'number' || (typeof v === 'string' && numberTextRe.test(v)))) return 'numeric';
    if (all((v) => Array.isArray(v))) return 'array';
    if (all((v) => typeof v === 'object')) return 'json';
    if (all((v) => typeof v === 'string' && dateRe.test(v))) return 'date';
    if (all((v) => typeof v === 'string' && timestampRe.test(v))) return 'timestamp';
    if (all((v) => typeof v === 'string' && timeRe.test(v))) return 'time';
    if (all((v) => typeof v === 'string' && uuidRe.test(v))) return 'uuid';
    if (all((v) => typeof v === 'string')) return 'text';
    return 'any';
}
