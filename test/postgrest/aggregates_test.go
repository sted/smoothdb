package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_Aggregates(t *testing.T) {

	tests := []test.Test{
		// Test simple aggregate - sum of all salaries (24000 + 36000 + 48000 = 108000)
		{
			Description: "simple sum aggregate",
			Query:       "/employees?select=salary.sum()",
			Headers:     nil,
			Expected:    `[{"sum":108000}]`,
			Status:      200,
		},
		// Test simple sum aggregate with grouping
		{
			Description: "sum aggregate with grouping by company",
			Query:       "/employees?select=salary.sum(),company&order=company.asc",
			Headers:     nil,
			Expected:    `[{"sum":36000,"company":"Dubrow's Cafeteria"},{"sum":24000,"company":"One-Up Realty"},{"sum":48000,"company":"Pro Garden Management"}]`,
			Status:      200,
		},
		// Test individual aggregates first
		{
			Description: "individual avg aggregate",
			Query:       "/employees?select=salary.avg()",
			Headers:     nil,
			Expected:    `[{"avg":36000.000000000000}]`,
			Status:      200,
		},
		{
			Description: "individual max aggregate",
			Query:       "/employees?select=salary.max()",
			Headers:     nil,
			Expected:    `[{"max":48000}]`,
			Status:      200,
		},
		{
			Description: "individual min aggregate",
			Query:       "/employees?select=salary.min()",
			Headers:     nil,
			Expected:    `[{"min":24000}]`,
			Status:      200,
		},
		{
			Description: "individual count aggregate",
			Query:       "/employees?select=salary.count()",
			Headers:     nil,
			Expected:    `[{"count":3}]`,
			Status:      200,
		},
		{
			Description: "sum and average",
			Query:       "/employees?select=salary.sum(),salary.avg()",
			Headers:     nil,
			Expected:    `[{"sum":108000,"avg":36000.000000000000}]`,
			Status:      200,
		},
		// Test aggregate with custom label
		{
			Description: "aggregate with custom label",
			Query:       "/employees?select=total_salary:salary.sum()",
			Headers:     nil,
			Expected:    `[{"total_salary":108000}]`,
			Status:      200,
		},
		// Test aggregate with cast
		{
			Description: "aggregate with cast",
			Query:       "/employees?select=salary.avg()::int",
			Headers:     nil,
			Expected:    `[{"avg":36000}]`,
			Status:      200,
		},
		// Test aggregate with filtering by occupation
		{
			Description: "aggregate with filtering by occupation",
			Query:       "/employees?select=salary.sum()&occupation=eq.Packer",
			Headers:     nil,
			Expected:    `[{"sum":36000}]`,
			Status:      200,
		},
		// Test count aggregate (3 employees)
		{
			Description: "count aggregate",
			Query:       "/employees?select=first_name.count()",
			Headers:     nil,
			Expected:    `[{"count":3}]`,
			Status:      200,
		},
		// Test aggregate with grouping by occupation
		{
			Description: "aggregate grouped by occupation",
			Query:       "/employees?select=salary.avg(),occupation&order=occupation.asc",
			Headers:     nil,
			Expected:    `[{"avg":24000.000000000000,"occupation":"Author"},{"avg":48000.000000000000,"occupation":"Marine biologist"},{"avg":36000.000000000000,"occupation":"Packer"}]`,
			Status:      200,
		},
		// Test aggregate with ordering by company
		{
			Description: "aggregate with ordering by company",
			Query:       "/employees?select=salary.max(),company&order=company.asc",
			Headers:     nil,
			Expected:    `[{"max":36000,"company":"Dubrow's Cafeteria"},{"max":24000,"company":"One-Up Realty"},{"max":48000,"company":"Pro Garden Management"}]`,
			Status:      200,
		},
		// Test filtering with aggregates (Daniel + Edwin = 36000 + 48000 = 84000)
		{
			Description: "filtering with salary greater than 30000",
			Query:       "/employees?select=salary.sum()&salary=gt.30000",
			Headers:     nil,
			Expected:    `[{"sum":84000}]`,
			Status:      200,
		},

		// === PostgREST compatibility tests ===
		// performing a count without specifying a field
		{
			Description: "returns the count of all rows when no other fields are selected",
			Query:       "/entities?select=count()",
			Headers:     nil,
			Expected:    `[{"count":4}]`,
			Status:      200,
		},
		{
			Description: "allows you to specify an alias for the count",
			Query:       "/entities?select=cnt:count()",
			Headers:     nil,
			Expected:    `[{"cnt":4}]`,
			Status:      200,
		},
		{
			Description: "allows you to cast the result of the count",
			Query:       "/entities?select=count()::text",
			Headers:     nil,
			Expected:    `[{"count":"4"}]`,
			Status:      200,
		},
		{
			Description: "returns the count grouped by all provided fields when other fields are selected",
			Query:       "/projects?select=c:count(),client_id&order=client_id.desc",
			Headers:     nil,
			Expected:    `[{"c":1,"client_id":null},{"c":2,"client_id":2},{"c":2,"client_id":1}]`,
			Status:      200,
		},

		// // performing a count by using it as a column (backwards compat)
		// {
		// 	Description: "returns the count of all rows when no other fields are selected (backwards compat)",
		// 	Query:       "/entities?select=count",
		// 	Headers:     nil,
		// 	Expected:    `[{"count":4}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "returns the embedded count of another resource",
		// 	Query:       "/clients?select=name,projects(count)",
		// 	Headers:     nil,
		// 	Expected:    `[{"name":"Microsoft","projects":[{"count":2}]},{"name":"Apple","projects":[{"count":2}]}]`,
		// 	Status:      200,
		// },

		// performing an aggregation on one or more fields
		{
			Description: "supports sum()",
			Query:       "/project_invoices?select=invoice_total.sum()",
			Headers:     nil,
			Expected:    `[{"sum":8800}]`,
			Status:      200,
		},
		{
			Description: "supports avg()",
			Query:       "/project_invoices?select=invoice_total.avg()",
			Headers:     nil,
			Expected:    `[{"avg":1100.0000000000000000}]`,
			Status:      200,
		},
		{
			Description: "supports min()",
			Query:       "/project_invoices?select=invoice_total.min()",
			Headers:     nil,
			Expected:    `[{"min":100}]`,
			Status:      200,
		},
		{
			Description: "supports max()",
			Query:       "/project_invoices?select=invoice_total.max()",
			Headers:     nil,
			Expected:    `[{"max":4000}]`,
			Status:      200,
		},
		{
			Description: "supports count()",
			Query:       "/project_invoices?select=invoice_total.count()",
			Headers:     nil,
			Expected:    `[{"count":8}]`,
			Status:      200,
		},
		{
			Description: "groups by any fields selected that do not have an aggregate applied",
			Query:       "/project_invoices?select=invoice_total.sum(),invoice_total.max(),invoice_total.min(),project_id&order=project_id.desc",
			Headers:     nil,
			Expected:    `[{"sum":4100,"max":4000,"min":100,"project_id":4},{"sum":3200,"max":2000,"min":1200,"project_id":3},{"sum":1200,"max":700,"min":500,"project_id":2},{"sum":300,"max":200,"min":100,"project_id":1}]`,
			Status:      200,
		},
		{
			Description: "supports the use of aliases on fields that will be used in the group by",
			Query:       "/project_invoices?select=invoice_total.sum(),invoice_total.max(),invoice_total.min(),pid:project_id&order=project_id.desc",
			Headers:     nil,
			Expected:    `[{"sum":4100,"max":4000,"min":100,"pid":4},{"sum":3200,"max":2000,"min":1200,"pid":3},{"sum":1200,"max":700,"min":500,"pid":2},{"sum":300,"max":200,"min":100,"pid":1}]`,
			Status:      200,
		},
		{
			Description: "allows you to specify an alias for the aggregate",
			Query:       "/project_invoices?select=total_charged:invoice_total.sum(),project_id&order=project_id.desc",
			Headers:     nil,
			Expected:    `[{"total_charged":4100,"project_id":4},{"total_charged":3200,"project_id":3},{"total_charged":1200,"project_id":2},{"total_charged":300,"project_id":1}]`,
			Status:      200,
		},
		{
			Description: "allows you to cast the result of the aggregate",
			Query:       "/project_invoices?select=total_charged:invoice_total.sum()::text,project_id&order=project_id.desc",
			Headers:     nil,
			Expected:    `[{"total_charged":"4100","project_id":4},{"total_charged":"3200","project_id":3},{"total_charged":"1200","project_id":2},{"total_charged":"300","project_id":1}]`,
			Status:      200,
		},
		{
			Description: "allows you to cast the input argument of the aggregate",
			Query:       "/trash_details?select=jsonb_col->>key::integer.sum()",
			Headers:     nil,
			Expected:    `[{"sum":24}]`,
			Status:      200,
		},
		{
			Description: "allows the combination of an alias, a before cast, and an after cast",
			Query:       "/trash_details?select=s:jsonb_col->>key::integer.sum()::text",
			Headers:     nil,
			Expected:    `[{"s":"24"}]`,
			Status:      200,
		},
		{
			Description: "supports use of aggregates on RPC functions that return table values",
			Query:       "/rpc/getallprojects?select=id.max()",
			Headers:     nil,
			Expected:    `[{"max":5}]`,
			Status:      200,
		},
		// {
		// 	Description: "allows the use of an JSON-embedded relationship column as part of the group by",
		// 	Query:       "/project_invoices?select=project_id,total:invoice_total.sum(),projects(name)&order=project_id",
		// 	Headers:     nil,
		// 	Expected:    `[{"project_id":1,"total":300,"projects":{"name":"Windows 7"}},{"project_id":2,"total":1200,"projects":{"name":"Windows 10"}},{"project_id":3,"total":3200,"projects":{"name":"IOS"}},{"project_id":4,"total":4100,"projects":{"name":"OSX"}}]`,
		// 	Status:      200,
		// },

		// performing aggregations that involve JSON-embedded relationships
		{
			Description: "supports sum() in embedded relationships",
			Query:       "/projects?select=name,project_invoices(invoice_total.sum())",
			Headers:     nil,
			Expected:    `[{"name":"Windows 7","project_invoices":[{"sum":300}]},{"name":"Windows 10","project_invoices":[{"sum":1200}]},{"name":"IOS","project_invoices":[{"sum":3200}]},{"name":"OSX","project_invoices":[{"sum":4100}]},{"name":"Orphan","project_invoices":[{"sum":null}]}]`,
			Status:      200,
		},
		{
			Description: "supports max() in embedded relationships",
			Query:       "/projects?select=name,project_invoices(invoice_total.max())",
			Headers:     nil,
			Expected:    `[{"name":"Windows 7","project_invoices":[{"max":200}]},{"name":"Windows 10","project_invoices":[{"max":700}]},{"name":"IOS","project_invoices":[{"max":2000}]},{"name":"OSX","project_invoices":[{"max":4000}]},{"name":"Orphan","project_invoices":[{"max":null}]}]`,
			Status:      200,
		},
		{
			Description: "supports avg() in embedded relationships",
			Query:       "/projects?select=name,project_invoices(invoice_total.avg())",
			Headers:     nil,
			Expected:    `[{"name":"Windows 7","project_invoices":[{"avg":150.0000000000000000}]},{"name":"Windows 10","project_invoices":[{"avg":600.0000000000000000}]},{"name":"IOS","project_invoices":[{"avg":1600.0000000000000000}]},{"name":"OSX","project_invoices":[{"avg":2050.0000000000000000}]},{"name":"Orphan","project_invoices":[{"avg":null}]}]`,
			Status:      200,
		},
		{
			Description: "supports min() in embedded relationships",
			Query:       "/projects?select=name,project_invoices(invoice_total.min())",
			Headers:     nil,
			Expected:    `[{"name":"Windows 7","project_invoices":[{"min":100}]},{"name":"Windows 10","project_invoices":[{"min":500}]},{"name":"IOS","project_invoices":[{"min":1200}]},{"name":"OSX","project_invoices":[{"min":100}]},{"name":"Orphan","project_invoices":[{"min":null}]}]`,
			Status:      200,
		},
		{
			Description: "supports all aggregates at once in embedded relationships",
			Query:       "/projects?select=name,project_invoices(invoice_total.max(),invoice_total.min(),invoice_total.avg(),invoice_total.sum(),invoice_total.count())",
			Headers:     nil,
			Expected:    `[{"name":"Windows 7","project_invoices":[{"avg":150.0000000000000000,"max":200,"min":100,"sum":300,"count":2}]},{"name":"Windows 10","project_invoices":[{"avg":600.0000000000000000,"max":700,"min":500,"sum":1200,"count":2}]},{"name":"IOS","project_invoices":[{"avg":1600.0000000000000000,"max":2000,"min":1200,"sum":3200,"count":2}]},{"name":"OSX","project_invoices":[{"avg":2050.0000000000000000,"max":4000,"min":100,"sum":4100,"count":2}]},{"name":"Orphan","project_invoices":[{"avg":null,"max":null,"min":null,"sum":null,"count":0}]}]`,
			Status:      200,
		},

		// performing aggregations on spreaded fields from an embedded resource
		// to-one spread relationships
		// {
		// 	Description: "supports the use of aggregates on spreaded fields",
		// 	Query:       "/budget_expenses?select=total_expenses:expense_amount.sum(),...budget_categories(budget_owner,total_budget:budget_amount.sum())&order=budget_categories(budget_owner)",
		// 	Headers:     nil,
		// 	Expected:    `[{"total_expenses":600.52,"budget_owner":"Brian Smith","total_budget":2000.42},{"total_expenses":100.22,"budget_owner":"Jane Clarkson","total_budget":7000.41},{"total_expenses":900.27,"budget_owner":"Sally Hughes","total_budget":500.23}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports the use of aggregates on spreaded fields when only aggregates are supplied",
		// 	Query:       "/budget_expenses?select=...budget_categories(total_budget:budget_amount.sum())",
		// 	Headers:     nil,
		// 	Expected:    `[{"total_budget":9501.06}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates from a spread relationships grouped by spreaded fields from other relationships",
		// 	Query:       "/processes?select=...process_costs(cost.sum()),...process_categories(name)",
		// 	Headers:     nil,
		// 	Expected:    `[{"sum":400.00,"name":"Batch"},{"sum":350.00,"name":"Mass"}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates from a spread relationships grouped by spreaded fields from other relationships with aliases",
		// 	Query:       "/processes?select=...process_costs(cost_sum:cost.sum()),...process_categories(category:name)",
		// 	Headers:     nil,
		// 	Expected:    `[{"cost_sum":400.00,"category":"Batch"},{"cost_sum":350.00,"category":"Mass"}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates on spreaded fields from nested relationships",
		// 	Query:       "/process_supervisor?select=...processes(factory_id,...process_costs(cost.sum()))",
		// 	Headers:     nil,
		// 	Expected:    `[{"factory_id":3,"sum":110.00},{"factory_id":2,"sum":500.00},{"factory_id":1,"sum":350.00}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates on spreaded fields from nested relationships with aliases",
		// 	Query:       "/process_supervisor?select=...processes(factory_id,...process_costs(cost_sum:cost.sum()))",
		// 	Headers:     nil,
		// 	Expected:    `[{"factory_id":3,"cost_sum":110.00},{"factory_id":2,"cost_sum":500.00},{"factory_id":1,"cost_sum":350.00}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates on spreaded fields from nested relationships, grouped by a regular nested relationship",
		// 	Query:       "/process_supervisor?select=...processes(factories(name),...process_costs(cost.sum()))",
		// 	Headers:     nil,
		// 	Expected:    `[{"factories":{"name":"Factory A"},"sum":350.00},{"factories":{"name":"Factory B"},"sum":500.00},{"factories":{"name":"Factory C"},"sum":110.00}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates on spreaded fields from nested relationships, grouped by a regular nested relationship with aliases",
		// 	Query:       "/process_supervisor?select=...processes(factory:factories(name),...process_costs(cost_sum:cost.sum()))",
		// 	Headers:     nil,
		// 	Expected:    `[{"factory":{"name":"Factory A"},"cost_sum":350.00},{"factory":{"name":"Factory B"},"cost_sum":500.00},{"factory":{"name":"Factory C"},"cost_sum":110.00}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates on spreaded fields from nested relationships, grouped by spreaded fields from other nested relationships",
		// 	Query:       "/process_supervisor?select=supervisor_id,...processes(...process_costs(cost.sum()),...process_categories(name))&order=supervisor_id",
		// 	Headers:     nil,
		// 	Expected:    `[{"supervisor_id":1,"sum":220.00,"name":"Batch"},{"supervisor_id":2,"sum":70.00,"name":"Batch"},{"supervisor_id":2,"sum":200.00,"name":"Mass"},{"supervisor_id":3,"sum":180.00,"name":"Batch"},{"supervisor_id":3,"sum":110.00,"name":"Mass"},{"supervisor_id":4,"sum":180.00,"name":"Batch"}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates on spreaded fields from nested relationships, grouped by spreaded fields from other nested relationships with aliases",
		// 	Query:       "/process_supervisor?select=supervisor_id,...processes(...process_costs(cost_sum:cost.sum()),...process_categories(category:name))&order=supervisor_id",
		// 	Headers:     nil,
		// 	Expected:    `[{"supervisor_id":1,"cost_sum":220.00,"category":"Batch"},{"supervisor_id":2,"cost_sum":70.00,"category":"Batch"},{"supervisor_id":2,"cost_sum":200.00,"category":"Mass"},{"supervisor_id":3,"cost_sum":180.00,"category":"Batch"},{"supervisor_id":3,"cost_sum":110.00,"category":"Mass"},{"supervisor_id":4,"cost_sum":180.00,"category":"Batch"}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates on spreaded fields from nested relationships, grouped by spreaded fields from other nested relationships, using a nested relationship as top parent",
		// 	Query:       "/supervisors?select=name,process_supervisor(...processes(...process_costs(cost.sum()),...process_categories(name)))",
		// 	Headers:     nil,
		// 	Expected:    `[{"name":"Mary","process_supervisor":[{"name":"Batch","sum":220.00}]},{"name":"John","process_supervisor":[{"name":"Batch","sum":70.00},{"name":"Mass","sum":200.00}]},{"name":"Peter","process_supervisor":[{"name":"Batch","sum":180.00},{"name":"Mass","sum":110.00}]},{"name":"Sarah","process_supervisor":[{"name":"Batch","sum":180.00}]},{"name":"Jane","process_supervisor":[]}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "supports aggregates on spreaded fields from nested relationships, grouped by spreaded fields from other nested relationships, using a nested relationship as top parent with aliases",
		// 	Query:       "/supervisors?select=name,process_supervisor(...processes(...process_costs(cost_sum:cost.sum()),...process_categories(category:name)))",
		// 	Headers:     nil,
		// 	Expected:    `[{"name":"Mary","process_supervisor":[{"category":"Batch","cost_sum":220.00}]},{"name":"John","process_supervisor":[{"category":"Batch","cost_sum":70.00},{"category":"Mass","cost_sum":200.00}]},{"name":"Peter","process_supervisor":[{"category":"Batch","cost_sum":180.00},{"category":"Mass","cost_sum":110.00}]},{"name":"Sarah","process_supervisor":[{"category":"Batch","cost_sum":180.00}]},{"name":"Jane","process_supervisor":[]}]`,
		// 	Status:      200,
		// },

		// supports count() aggregate without specifying a field
		// {
		// 	Description: "works by itself in the embedded resource",
		// 	Query:       "/process_supervisor?select=supervisor_id,...processes(count())&order=supervisor_id",
		// 	Headers:     nil,
		// 	Expected:    `[{"supervisor_id":1,"count":2},{"supervisor_id":2,"count":2},{"supervisor_id":3,"count":3},{"supervisor_id":4,"count":1}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "works by itself in the embedded resource with alias",
		// 	Query:       "/process_supervisor?select=supervisor_id,...processes(processes_count:count())&order=supervisor_id",
		// 	Headers:     nil,
		// 	Expected:    `[{"supervisor_id":1,"processes_count":2},{"supervisor_id":2,"processes_count":2},{"supervisor_id":3,"processes_count":3},{"supervisor_id":4,"processes_count":1}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "works alongside other columns in the embedded resource",
		// 	Query:       "/process_supervisor?select=...supervisors(id,count())&order=supervisors(id)",
		// 	Headers:     nil,
		// 	Expected:    `[{"id":1,"count":2},{"id":2,"count":2},{"id":3,"count":3},{"id":4,"count":1}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "works alongside other columns in the embedded resource with aliases",
		// 	Query:       "/process_supervisor?select=...supervisors(supervisor:id,supervisor_count:count())&order=supervisors(supervisor)",
		// 	Headers:     nil,
		// 	Expected:    `[{"supervisor":1,"supervisor_count":2},{"supervisor":2,"supervisor_count":2},{"supervisor":3,"supervisor_count":3},{"supervisor":4,"supervisor_count":1}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "works on nested resources",
		// 	Query:       "/process_supervisor?select=supervisor_id,...processes(...process_costs(count()))&order=supervisor_id",
		// 	Headers:     nil,
		// 	Expected:    `[{"supervisor_id":1,"count":2},{"supervisor_id":2,"count":2},{"supervisor_id":3,"count":3},{"supervisor_id":4,"count":1}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "works on nested resources with aliases",
		// 	Query:       "/process_supervisor?select=supervisor:supervisor_id,...processes(...process_costs(process_costs_count:count()))&order=supervisor_id",
		// 	Headers:     nil,
		// 	Expected:    `[{"supervisor":1,"process_costs_count":2},{"supervisor":2,"process_costs_count":2},{"supervisor":3,"process_costs_count":3},{"supervisor":4,"process_costs_count":1}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "works on nested resources grouped by spreaded fields",
		// 	Query:       "/process_supervisor?select=...processes(factory_id,...process_costs(count()))&order=processes(factory_id)",
		// 	Headers:     nil,
		// 	Expected:    `[{"factory_id":1,"count":2},{"factory_id":2,"count":4},{"factory_id":3,"count":2}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "works on nested resources grouped by spreaded fields with aliases",
		// 	Query:       "/process_supervisor?select=...processes(factory:factory_id,...process_costs(process_costs_count:count()))&order=processes(factory)",
		// 	Headers:     nil,
		// 	Expected:    `[{"factory":1,"process_costs_count":2},{"factory":2,"process_costs_count":4},{"factory":3,"process_costs_count":2}]`,
		// 	Status:      200,
		// },
		// {
		// 	Description: "works on different levels of the nested resources at the same time",
		// 	Query:       "/process_supervisor?select=...processes(factory:factory_id,processes_count:count(),...process_costs(process_costs_count:count()))&order=processes(factory)",
		// 	Headers:     nil,
		// 	Expected:    `[{"factory":1,"processes_count":2,"process_costs_count":2},{"factory":2,"processes_count":4,"process_costs_count":4},{"factory":3,"processes_count":2,"process_costs_count":2}]`,
		// 	Status:      200,
		// },

		// to-many spread relationships
		// {
		// 	Description: "does not support the use of aggregates on to-many spreads",
		// 	Query:       "/factories?select=name,...factory_buildings(type,size.sum())",
		// 	Headers:     nil,
		// 	Expected:    `{"code":"PGRST127","message":"Feature not implemented","details":"Aggregates are not implemented for one-to-many or many-to-many spreads.","hint":null}`,
		// 	Status:      400,
		// },
	}

	test.Execute(t, testConfig, tests)
}

func TestPostgREST_AggregatesDisallowed(t *testing.T) {

	tests := []test.Test{
		// attempting to use an aggregate when aggregate functions are disallowed
		// {
		// 	Description: "prevents the use of aggregates",
		// 	Query:       "/project_invoices?select=invoice_total.sum()",
		// 	Headers:     nil,
		// 	Expected:    `{"hint":null,"details":null,"code":"PGRST123","message":"Use of aggregate functions is not allowed"}`,
		// 	Status:      400,
		// },
		// {
		// 	Description: "prevents the use of aggregates on embedded relationships",
		// 	Query:       "/projects?select=name,project_invoices(invoice_total.sum())",
		// 	Headers:     nil,
		// 	Expected:    `{"hint":null,"details":null,"code":"PGRST123","message":"Use of aggregate functions is not allowed"}`,
		// 	Status:      400,
		// },
		// {
		// 	Description: "prevents the use of aggregates on to-one spread embeds",
		// 	Query:       "/project_invoices?select=...projects(id.count())",
		// 	Headers:     nil,
		// 	Expected:    `{"hint":null,"details":null,"code":"PGRST123","message":"Use of aggregate functions is not allowed"}`,
		// 	Status:      400,
		// },
		// {
		// 	Description: "prevents the use of aggregates on to-many spread embeds",
		// 	Query:       "/factories?select=...processes(id.count())",
		// 	Headers:     nil,
		// 	Expected:    `{"hint":null,"details":null,"code":"PGRST123","message":"Use of aggregate functions is not allowed"}`,
		// 	Status:      400,
		// },
	}

	test.Execute(t, testConfig, tests)
}
