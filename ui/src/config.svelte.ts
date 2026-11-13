// Server configuration and UI state.
//
// The server writes window.SMOOTHDB_CONFIG into the page it serves
// (api/ui.go). The Vite dev server does not: in development the UI uses the
// default server settings, and Vite proxies the smoothdb routes (vite.config.js).

const injected = window.SMOOTHDB_CONFIG;

// The UI and the server share the origin, in production and through the dev proxy.
export const serverBase = '';

export const adminDbUrl = serverBase + (injected?.adminURL ?? '/admin');
export const apiDbUrl = serverBase + (injected?.apiURL ?? '/api');
const shortAPIURL = injected?.shortAPIURL ?? false;
export const loginMode = injected?.loginMode ?? 'db';

// The data API URL of a table or view. With ShortAPIURL the server serves a
// single database and the URL has no database segment.
export function dbApiUrl(db: string): string {
    return shortAPIURL ? apiDbUrl : `${apiDbUrl}/${encodeURIComponent(db)}`;
}

export function tableUrl(db: string, table: string): string {
    return `${dbApiUrl(db)}/${encodeURIComponent(table)}`;
}

export const ui = $state({
    // the role can use the admin API: null until the first probe
    admin: null as boolean | null,
    // In development the first 401 sets it.
    authRequired: injected ? !injected.allowAnon : false,
    error:
        injected && injected.loginMode === 'none' && !injected.allowAnon
            ? 'Invalid configuration: LoginMode is "none" and AllowAnon is false, so nobody can use the UI. Set LoginMode to "db" or "gotrue", or set AllowAnon to true.'
            : '',
});
