import Router from './router'
import type { RouteDef } from './router'

import Home from './pages/Home.svelte';
import DatabaseForm from './forms/DatabaseForm.svelte';
import RoleForm from './forms/RoleForm.svelte';
import BasicPage from './pages/BasicPage.svelte';
import TableForm from './forms/TableForm.svelte';

export const routeDefs: RouteDef[] = [
    { pattern: '/', redirect: '/databases/' },
    { pattern: '/databases', component: DatabaseForm},
    { pattern: '/databases/:db', redirect: '/databases/:db/tables' },
    { pattern: '/databases/:db/tables', component: TableForm},
    { pattern: '/databases/:db/tables/:table', redirect: '/databases/:db/tables/:table/columns' },
    { pattern: '/databases/:db/tables/:table/columns', component: TableForm },
    { pattern: '/databases/:db/tables/:table/constraints' },
    { pattern: '/databases/:db/tables/:table/policies' },
    { pattern: '/databases/:db/tables/:table/grants'},
    { pattern: '/databases/:db/views'},
    { pattern: '/databases/:db/functions'},
    { pattern: '/databases/:db/schemas'},
    { pattern: '/databases/:db/grants'},
    { pattern: '/roles', component: RoleForm },
];

export const router = new Router(routeDefs, BasicPage)