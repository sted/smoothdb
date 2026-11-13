import Router from './router.svelte'
import type { RouteConfig } from './router.svelte'

import BasicPage from './pages/BasicPage.svelte';
import DataPage from './pages/DataPage.svelte';
import LoginPage from './pages/LoginPage.svelte';
import DatabaseForm from './forms/DatabaseForm.svelte';
import RoleForm from './forms/RoleForm.svelte';
import TableForm from './forms/TableForm.svelte';
import SchemaForm from './forms/SchemaForm.svelte';
import ColumnForm from './forms/ColumnForm.svelte';
import ConstraintForm from './forms/ConstraintForm.svelte';

export const routeDefs: RouteConfig[] = [
    { pattern: '', redirect: '/databases' },
    { pattern: '/login', component: LoginPage },
    
    // Admin routes (DDL/schema management)
    { pattern: '/databases', component: DatabaseForm },
    { pattern: '/databases/:db', redirect: '/databases/:db/tables' },
    { pattern: '/databases/:db/tables', component: TableForm },
    { pattern: '/databases/:db/tables/:table', redirect: '/databases/:db/tables/:table/columns' },
    { pattern: '/databases/:db/tables/:table/columns', component: ColumnForm },
    { pattern: '/databases/:db/tables/:table/constraints', component: ConstraintForm },
    { pattern: '/databases/:db/tables/:table/policies' },
    { pattern: '/databases/:db/tables/:table/grants' },
    { pattern: '/databases/:db/views' },
    { pattern: '/databases/:db/functions' },
    { pattern: '/databases/:db/schemas', component: SchemaForm },
    { pattern: '/databases/:db/grants' },
    { pattern: '/roles', component: RoleForm },
    
    // Data routes (actual table content)
    { pattern: '/data', component: DataPage },
    { pattern: '/data/:db', component: DataPage },
    { pattern: '/data/:db/:schema', redirect: '/data/:db' },
    { pattern: '/data/:db/:schema/:table', component: DataPage },
];

export const router = new Router(routeDefs, BasicPage)