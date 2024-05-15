import Router from './router'
import type { RouteDef } from './router'

import Home from './pages/Home.svelte';
import Databases from './pages/Databases.svelte';
import Tables from './pages/Tables.svelte';
import Views from './pages/Views.svelte';
import Columns from './pages/Columns.svelte';
import Roles from './pages/Roles.svelte';
import Functions from './pages/Functions.svelte';
import Schemas from './pages/Schemas.svelte';
import Policies from './pages/Policies.svelte';
import DbGrants from './pages/DbGrants.svelte';
import TableGrants from './pages/TableGrants.svelte';
import Constraints from './pages/Constraints.svelte';

export const routeDefs: RouteDef[] = [
    // { pattern: '/', redirect: '/databases/' },
    { pattern: '/databases', component: Databases },
    { pattern: '/databases/:db', redirect: '/databases/:db/tables' },
    { pattern: '/databases/:db/tables', component: Tables },
    { pattern: '/databases/:db/tables/:table', redirect: '/databases/:db/tables/:table/columns' },
    { pattern: '/databases/:db/tables/:table/columns', component: Columns },
    { pattern: '/databases/:db/tables/:table/constraints', component: Constraints },
    { pattern: '/databases/:db/tables/:table/policies', component: Policies },
    { pattern: '/databases/:db/tables/:table/grants', component: TableGrants },
    { pattern: '/databases/:db/views', component: Views },
    { pattern: '/databases/:db/functions', component: Functions },
    { pattern: '/databases/:db/schemas', component: Schemas },
    { pattern: '/databases/:db/grants', component: DbGrants },
    { pattern: '/roles', component: Roles },
];

export const router = new Router(routeDefs, Home)