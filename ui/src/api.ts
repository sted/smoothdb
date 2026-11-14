import { authStore, type User } from './auth.svelte';
import { serverBase, ui } from './config.svelte';
import { showError } from './toast.svelte';

export interface Data {
	name: string;
	schema?: string;
	[key: string]: any;
}

export interface ApiResponse<T = any> {
	success: boolean;
	data?: T;
	error?: string;
	status?: number;
	headers?: Headers;
}

export interface AccessTokenResponse {
	access_token: string;
	token_type: string;
	expires_in: number;
	expires_at: number;
	refresh_token?: string;
	user?: User;
}

export interface RequestOptions extends RequestInit {
	// do not show an error toast: the caller reports the error itself
	quiet?: boolean;
}

// The error body of smoothdb: PostgREST errors carry code, message, details
// and hint, on every route; a non-JSON body is shown as text.
function describeError(status: number, statusText: string, text: string): { message: string; detail?: string } {
	try {
		const e = JSON.parse(text);
		const message = e.message || e.error || `HTTP ${status}`;
		const detail = [e.code, e.details, e.hint].filter((v) => v && typeof v === 'string').join('\n');
		return { message, detail: detail || undefined };
	} catch {
		return { message: text || `HTTP ${status}: ${statusText}` };
	}
}

// keepNumberText keeps a JSON number as text when a JavaScript number would
// change it: the scale of a numeric (210.30) or the digits of a big integer.
// It needs the source text access of JSON.parse; without it numbers stay numbers.
function keepNumberText(_key: string, value: unknown, context?: { source?: string }): unknown {
	if (typeof value === 'number' && context?.source !== undefined && context.source !== String(value)) {
		return context.source;
	}
	return value;
}

export const parseJSON = (text: string) => JSON.parse(text, keepNumberText as any);

// makeRequest sends a request with the auth token. A relative endpoint is
// resolved against the smoothdb server. It never throws: it reports a failure
// in the result and, unless quiet, in an error toast.
export async function makeRequest<T = any>(endpoint: string, options: RequestOptions = {}): Promise<ApiResponse<T>> {
	const { quiet, ...init } = options;
	const url = /^https?:\/\//.test(endpoint) ? endpoint : `${serverBase}${endpoint}`;
	let response: Response;
	try {
		response = await fetch(url, { ...init, headers: { ...authStore.getAuthHeaders(), ...init.headers } });
	} catch (error) {
		const message = error instanceof Error ? error.message : 'Network error';
		if (!quiet) showError(`Cannot reach the server: ${message}`);
		return { success: false, error: message, status: 0 };
	}

	if (!response.ok) {
		const { message, detail } = describeError(response.status, response.statusText, await response.text());
		if (response.status === 401) {
			// the token is missing or expired: the app shows the login page
			authStore.clearAuth();
			ui.authRequired = true;
		} else if (!quiet) {
			showError(message, detail);
		}
		return { success: false, error: message, status: response.status };
	}

	const text = await response.text();
	if (text === '') return { success: true, status: response.status, headers: response.headers };
	// the media type alone: a parameter can name json too (a text plan has
	// for="application/json")
	if ((response.headers.get('content-type') ?? '').split(';')[0].includes('json')) {
		try {
			return { success: true, data: parseJSON(text), status: response.status, headers: response.headers };
		} catch (e) {
			const message = `Invalid JSON response: ${e instanceof Error ? e.message : e}`;
			if (!quiet) showError(message);
			return { success: false, error: message, status: response.status };
		}
	}
	return { success: true, data: text as T, status: response.status, headers: response.headers };
}

// login asks /token for a token. In db mode the server takes the role name
// in the email field.
export async function login(username: string, password: string): Promise<ApiResponse<AccessTokenResponse>> {
	const response = await makeRequest<AccessTokenResponse>('/token', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ email: username, password }),
		quiet: true,
	});
	if (response.success && response.data) {
		const user: User = response.data.user || { aud: '', role: username, username, email: '' };
		authStore.setAuth(response.data.access_token, user);
	}
	return response;
}
