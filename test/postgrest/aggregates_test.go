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
			Expected:    `[{"avg":36000}]`,
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
			Description: "two aggregates",
			Query:       "/employees?select=salary.sum(),salary.avg()",
			Headers:     nil,
			Expected:    `[{"sum":108000,"avg":36000}]`,
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
			Expected:    `[{"avg":24000,"occupation":"Author"},{"avg":48000,"occupation":"Marine biologist"},{"avg":36000,"occupation":"Packer"}]`,
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
	}

	test.Execute(t, testConfig, tests)
}
