import type { Component } from 'svelte';

export interface RouteConfig {
    pattern: string
    page?: Component
    component?: any
    redirect?: string
    // data for the page of the route, such as the kind of object it lists
    meta?: Record<string, string>
}

interface Route {
    regex: RegExp
    paramNames: string[]
    page?: Component
    component?: Component
    redirect?: string
    meta?: Record<string, string>
}

interface RouteParams {
    [key: string]: string;
}

export default class Router {
    path: string = $state("")
    // the query string of the route, without "?"
    query: string = $state("")
    page: Component | undefined = $state()
    component: Component | undefined = $state()
    params: RouteParams = $state({})
    meta: Record<string, string> = $state({})

    private basePath: string = '/ui';
    private routes: Route[]
    private routesMap: Map<string, { regex: RegExp, nextSegments: Set<string> }> = new Map();

    constructor(routeDefs: RouteConfig[], defPage: any) {
        this.page = defPage
        // compile routes and pre-calculate segments
        this.routes = routeDefs.map(({ pattern, page, component, redirect, meta }: RouteConfig): Route => {
            pattern = this.basePath + pattern;
            const { regex, paramNames } = this.compilePattern(pattern);
            this.precalculateNextSegments(pattern);
            return { regex, paramNames, page, component, redirect, meta }
        });

        window.addEventListener('popstate', () => {
            this.navigate(window.location.pathname + window.location.search, false);
        });
        // Same-origin links under the base path change the route without a
        // page load. Modified clicks (new tab, new window) keep the browser default.
        document.addEventListener('click', (ev) => {
            if (ev.defaultPrevented || ev.button !== 0 || ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey) return;
            const a = (ev.target as Element | null)?.closest?.('a');
            if (!a || a.target || a.hasAttribute('download')) return;
            const url = new URL(a.href, window.location.href);
            if (url.origin !== window.location.origin || !url.pathname.startsWith(this.basePath)) return;
            if (this.navigate(url.pathname + url.search)) ev.preventDefault();
        });
    }

    navigate(path: string, updateHistory = true): boolean {
        let query = "";
        const q = path.indexOf("?");
        if (q >= 0) {
            query = path.slice(q + 1);
            path = path.slice(0, q);
        }
        if (!path.startsWith(this.basePath)) {
            path = this.basePath + path;
        }
        if (path.length > 1 && path.endsWith('/')) {
            path = path.slice(0, -1);
        }
        let paramValues: RouteParams = {};
        let matched = false;

        for (const route of this.routes) {
            const match = route.regex.exec(path);
            if (match) {
                matched = true
                paramValues = route.paramNames.reduce((acc: RouteParams, key, index) => {
                    acc[key] = decodeURIComponent(match[index + 1]);
                    return acc;
                }, {});

                if (route.redirect) {
                    const redirectPath = this.basePath + route.redirect.replace(/:([^\/]+)/g, (_, key) => paramValues[key]);
                    return this.navigate(redirectPath);
                }

                this.path = path;
                this.query = query;
                if (route.page) {
                    this.page = route.page;
                }
                this.component = route.component;
                this.params = paramValues;
                this.meta = route.meta ?? {};

                break;
            }
        }

        if (matched) {
            const url = this.path + (this.query ? "?" + this.query : "");
            const current = window.location.pathname + window.location.search;
            if (updateHistory && current !== url) {
                window.history.pushState({}, '', url);
            } else if (current !== url) {
                window.history.replaceState({}, '', url);
            }
        }
        return matched;
    }

    // setQuery changes the query string of the current route.
    setQuery(params: URLSearchParams, updateHistory = true): void {
        this.navigate(this.path + (params.size ? "?" + params.toString() : ""), updateHistory);
    }

    getAltRoutes(path: string): string[] {
        const segments = path.split('/').filter(Boolean).slice(0, -1);
        const pathWithoutLastSegment = segments.join('/');

        for (const { regex, nextSegments } of this.routesMap.values()) {
            if (regex.test('/' + pathWithoutLastSegment)) {
                return Array.from(nextSegments);
            }
        }
        return [];
    }

    private compilePattern(pattern: string): { regex: RegExp, paramNames: string[] } {
        const regex = /:([^\/]+)/g;
        const paramNames: string[] = [];
        const replacedPattern = pattern.replace(regex, (_, key) => {
            paramNames.push(key);
            return '([^\\/]+)';
        });
        return { regex: new RegExp(`^${replacedPattern}$`), paramNames };
    }

    private precalculateNextSegments(pattern: string): void {
        const segments = pattern.split('/').filter(Boolean);
        let currentPattern = '';

        segments.forEach((segment, index) => {
            currentPattern += '/' + segment;
            const regexPattern = `^${currentPattern.replace(/:[^\s/]+/g, '[^/]+')}$`;
            const regex = new RegExp(regexPattern);

            const nextSegment = segments[index + 1];
            if (nextSegment) {
                if (!this.routesMap.has(regexPattern)) {
                    this.routesMap.set(regexPattern, { regex, nextSegments: new Set() });
                }
                this.routesMap.get(regexPattern)?.nextSegments.add(nextSegment);
            }
        });
    }
}