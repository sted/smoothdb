import type { Writable } from 'svelte/store';
import { writable, get } from 'svelte/store';
import type { ComponentType } from 'svelte';


export interface RouteDef {
    pattern: string;
    component?: ComponentType;
    redirect?: string;
}

interface LocationState {
    component: ComponentType;
}

interface Route {
    regex: RegExp
    paramNames: string[]
    component?: ComponentType
    redirect?: string
}

interface RouteParams {
    [key: string]: string;
}

export default class Router {
    private basePath: string = '/ui'; 
    private defaultState: LocationState
    private location: Writable<LocationState>
    private routes: Route[]
    private routesMap: Map<string, { regex: RegExp, nextSegments: Set<string> }> = new Map();

    constructor(routeDefs: RouteDef[], defComponent: ComponentType) {
        this.defaultState = {component: defComponent}
        this.location = writable<LocationState>(this.defaultState);
        // compile routes and pre-calculate segments
        this.routes = routeDefs.map(({ pattern, component, redirect }) => {
            pattern = this.basePath + pattern;
            const {regex, paramNames} = this.compilePattern(pattern);
            this.precalculateNextSegments(pattern);
            return {regex, paramNames, component, redirect}
          });

        window.addEventListener('popstate', () => {
            this.navigate(window.location.pathname, false);
        });
    }

    subscribe(run: (value: LocationState) => void): () => void {
        return this.location.subscribe(run)
    }

    navigate(path: string, updateHistory = true) {
        if (!path.startsWith(this.basePath)) {
            path = this.basePath + path;
        }
        const currentState = get(this.location);
        let newState: LocationState = { ...this.defaultState };
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
                    this.navigate(redirectPath);
                    return;
                }
                if (route.component) {
                    newState.component = route.component;
                }
                break;
            }
        }

        if (updateHistory && newState.component !== currentState.component) {
            window.history.pushState({}, '', path);
        }
        this.location.set(newState);
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
     
    
    private compilePattern(pattern: string): {regex: RegExp, paramNames: string[]} {
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