import Router from './router.svelte'
import type { RouteConfig } from './router.svelte'

import BasicPage from './pages/BasicPage.svelte';
import DataPage from './pages/DataPage.svelte';
import LoginPage from './pages/LoginPage.svelte';
import AdminPage from './admin/AdminPage.svelte';

// An admin route names the kind of object it lists in meta.entity (admin/entities.ts).
const admin = (pattern: string, entity: string): RouteConfig => ({ pattern, component: AdminPage, meta: { entity } });

export const routeDefs: RouteConfig[] = [
    { pattern: '', redirect: '/databases' },
    { pattern: '/login', component: LoginPage },

    // Admin routes (DDL/schema management)
    admin('/databases', 'databases'),
    { pattern: '/databases/:db', redirect: '/databases/:db/tables' },
    admin('/databases/:db/tables', 'tables'),
    { pattern: '/databases/:db/tables/:schema', redirect: '/databases/:db/tables' },
    { pattern: '/databases/:db/tables/:schema/:table', redirect: '/databases/:db/tables/:schema/:table/columns' },
    admin('/databases/:db/tables/:schema/:table/columns', 'columns'),
    admin('/databases/:db/tables/:schema/:table/constraints', 'constraints'),
    admin('/databases/:db/tables/:schema/:table/policies', 'policies'),
    admin('/databases/:db/tables/:schema/:table/grants', 'tablegrants'),
    admin('/databases/:db/views', 'views'),
    admin('/databases/:db/functions', 'functions'),
    admin('/databases/:db/schemas', 'schemas'),
    admin('/databases/:db/grants', 'dbgrants'),
    admin('/roles', 'roles'),

    // Data routes (actual table content)
    { pattern: '/data', component: DataPage },
    { pattern: '/data/:db', component: DataPage },
    { pattern: '/data/:db/:schema', redirect: '/data/:db' },
    { pattern: '/data/:db/:schema/:table', component: DataPage },
];

export const router = new Router(routeDefs, BasicPage)
