import { makeRequest } from './api';

// getData returns the JSON at url, or throws after the error toast.
export async function getData(url: string): Promise<any> {
	const response = await makeRequest(url);
	if (!response.success) throw new Error(response.error);
	return response.data;
}

// "PRIMARY KEY (a, "B c")" -> ["a", "B c"]
export function parsePrimaryKey(constraints: string[] | null | undefined): string[] {
	const def = (constraints ?? []).find((c) => c.startsWith('PRIMARY KEY'));
	const m = def?.match(/^PRIMARY KEY \((.*)\)$/);
	if (!m) return [];
	return m[1].split(/,\s*/).map((n) => (n.startsWith('"') ? n.slice(1, -1).replace(/""/g, '"') : n));
}

// The text of a cell: JSON for objects and arrays.
export function formatCell(value: unknown): string {
	if (value === null || value === undefined) return 'NULL';
	if (typeof value === 'object') return JSON.stringify(value);
	return String(value);
}
