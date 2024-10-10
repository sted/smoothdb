import type { Component } from 'svelte';

export interface RouteConfig {
    pattern: string
    page?: Component
    component?: any
    redirect?: string
}

interface Route {
    regex: RegExp
    paramNames: string[]
    page?: Component
    component?: Component
    redirect?: string
}

interface RouteParams {
    [key: string]: string;
}

export default class Router {
    path: string = $state("")
    page: Component | undefined = $state()
    component: Component | undefined = $state()
    params: RouteParams = $state({})
    schema: string = $state("")

    private basePath: string = '/ui';
    private routes: Route[]
    private routesMap: Map<string, { regex: RegExp, nextSegments: Set<string> }> = new Map();

    constructor(routeDefs: RouteConfig[], defPage: any) {
        this.page = defPage
        // compile routes and pre-calculate segments
        this.routes = routeDefs.map(({ pattern, page, component, redirect }: RouteConfig): Route => {
            pattern = this.basePath + pattern;
            const { regex, paramNames } = this.compilePattern(pattern);
            this.precalculateNextSegments(pattern);
            return { regex, paramNames, page, component, redirect }
        });

        window.addEventListener('popstate', () => {
            this.navigate(window.location.pathname, this.schema, false);
        });
    }

    navigate(path: string, schema = "", updateHistory = true): boolean {
        const oldPath = this.path;
        if (!path.startsWith(this.basePath)) {
            path = this.basePath + path;
        }
        let paramValues: RouteParams = {};
        let matched = false;

        for (const route of this.routes) {
            const match = route.regex.exec(path);
            if (match) {
                matched = true
                paramValues = route.paramNames.reduce((acc: RouteParams, key, index) => {
                    acc[key] = match[index + 1];
                    return acc;
                }, {});

                if (route.redirect) {
                    const redirectPath = this.basePath + route.redirect.replace(/:([^\/]+)/g, (_, key) => paramValues[key]);
                    return this.navigate(redirectPath, schema);
                }

                this.path = path;
                if (route.page) {
                    this.page = route.page;
                }
                this.component = route.component;
                this.params = paramValues;
                if (schema !== "")
                    this.schema = schema; // if it is "", keep the current schema

                break;
            }
        }

        if (matched) {
            if (updateHistory && oldPath !== this.path) {
                window.history.pushState({}, '', path);
            }
        }
        return matched;
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