package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// see https://github.com/PostgREST/postgrest/commit/78d45b4e32e1129f25c2b6739dbd587ef873822c

func TestPostgREST_RelatedOrders(t *testing.T) {

	tests := []test.Test{
		// related orders
		// it "works on a many-to-one relationship"
		{
			Description: "related orders works on a many-to-one relationship",
			Query:       "/projects?select=id,clients(name)&order=clients(name).nullsfirst",
			Expected: `[
				{"id":5,"clients":null},
				{"id":3,"clients":{"name":"Apple"}},
				{"id":4,"clients":{"name":"Apple"}},
				{"id":1,"clients":{"name":"Microsoft"}},
				{"id":2,"clients":{"name":"Microsoft"}} ]`,
			Status: 200,
		},
		{
			Description: "related orders works with alias on a many-to-one relationship",
			Query:       "/projects?select=id,client:clients(name)&order=client(name).asc",
			Expected: `[
				{"id":3,"client":{"name":"Apple"}},
				{"id":4,"client":{"name":"Apple"}},
				{"id":1,"client":{"name":"Microsoft"}},
				{"id":2,"client":{"name":"Microsoft"}},
				{"id":5,"client":null} ]`,
			Status: 200,
		},

		// it "works on a one-to-one relationship and jsonb column"
		{
			Description: "related orders works on a one-to-one relationship with jsonb asc",
			Query:       "/trash?select=id,trash_details(id,jsonb_col)&order=trash_details(jsonb_col->key).asc",
			Expected: `[
				{"id":2,"trash_details":{"id":2,"jsonb_col":{"key": 6}}},
				{"id":3,"trash_details":{"id":3,"jsonb_col":{"key": 8}}},
				{"id":1,"trash_details":{"id":1,"jsonb_col":{"key": 10}}}
			]`,
			Status: 200,
		},
		{
			Description: "related orders works on a one-to-one relationship with jsonb desc",
			Query:       "/trash?select=id,trash_details(id,jsonb_col)&order=trash_details(jsonb_col->key).desc",
			Expected: `[
				{"id":1,"trash_details":{"id":1,"jsonb_col":{"key": 10}}},
				{"id":3,"trash_details":{"id":3,"jsonb_col":{"key": 8}}},
				{"id":2,"trash_details":{"id":2,"jsonb_col":{"key": 6}}}
			]`,
			Status: 200,
		},

		// it "works on an embedded resource" (nested related order)
		{
			Description: "nested related order desc",
			Query:       "/users?select=name,tasks(id,name,projects(id,name))&tasks.order=projects(id).desc&limit=1",
			Expected: `[{
				"name":"Angela Martin",
				"tasks":[
					{"id": 3, "name":"Design w10","projects":{"id":2,"name":"Windows 10"}},
					{"id": 4, "name":"Code w10","projects":{"id":2,"name":"Windows 10"}},
					{"id": 1, "name":"Design w7","projects":{"id":1,"name":"Windows 7"}},
					{"id": 2, "name":"Code w7","projects":{"id":1,"name":"Windows 7"}}
				]
			}]`,
			Status: 200,
		},
		{
			Description: "nested related order desc with secondary sort",
			Query:       "/users?select=name,tasks(id,name,projects(id,name))&tasks.order=projects(id).desc,name&limit=1",
			Expected: `[{
				"name":"Angela Martin",
				"tasks":[
					{"id": 4, "name":"Code w10","projects":{"id":2,"name":"Windows 10"}},
					{"id": 3, "name":"Design w10","projects":{"id":2,"name":"Windows 10"}},
					{"id": 2, "name":"Code w7","projects":{"id":1,"name":"Windows 7"}},
					{"id": 1, "name":"Design w7","projects":{"id":1,"name":"Windows 7"}}
				]
			}]`,
			Status: 200,
		},
		{
			Description: "nested related order asc",
			Query:       "/users?select=name,tasks(id,name,projects(id,name))&tasks.order=projects(id).asc&limit=1",
			Expected: `[{
				"name":"Angela Martin",
				"tasks":[
					{"id":1,"name":"Design w7","projects":{"id":1,"name":"Windows 7"}},
					{"id":2,"name":"Code w7","projects":{"id":1,"name":"Windows 7"}},
					{"id":3,"name":"Design w10","projects":{"id":2,"name":"Windows 10"}},
					{"id":4,"name":"Code w10","projects":{"id":2,"name":"Windows 10"}}
				]
			}]`,
			Status: 200,
		},

		// it "fails when is not a to-one relationship"
		{
			Description: "related order fails on to-many relationship",
			Query:       "/clients?select=*,projects(*)&order=projects(id)",
			Status:      400,
		},
		{
			Description: "related order fails on to-many relationship with alias",
			Query:       "/clients?select=*,pros:projects(*)&order=pros(id)",
			Status:      400,
		},
		{
			Description: "related order fails on to-many computed relationship",
			Query:       "/designers?select=id,computed_videogames(id)&order=computed_videogames(id).desc",
			Status:      400,
		},

		// it "fails when the resource is not embedded"
		{
			Description: "related order fails when resource is not embedded",
			Query:       "/projects?select=id,clients(name)&order=clientsx(name).nullsfirst",
			Status:      400,
		},
	}

	test.Execute(t, testConfig, tests)
}

// related orders - computed relationship (to-one)
// computed_designers returns SETOF so it's to-many in smoothdb;
// PostgREST may handle this differently. Use computed_designers_noset if available.
// {
// 	Description: "related orders works on a computed to-one relationship",
// 	Query:       "/videogames?select=id,computed_designers(id)&order=computed_designers(id).desc",
// 	Expected: `[
// 		{"id":3,"computed_designers":{"id":2}},
// 		{"id":4,"computed_designers":{"id":2}},
// 		{"id":1,"computed_designers":{"id":1}},
// 		{"id":2,"computed_designers":{"id":1}}
// 	]`,
// 	Status: 200,
// },

// context "related conditions through null operator on embed"

// it "works on a many-to-one relationship"
// {
// 	Description: "null operator on embed works on many-to-one not null",
// 	Query:       "/projects?select=name,clients()&clients=not.is.null",
// 	Expected: `[
// 		{"name":"Windows 7"},
// 		{"name":"Windows 10"},
// 		{"name":"IOS"},
// 		{"name":"OSX"}
// 	]`,
// 	Status: 200,
// },
// {
// 	Description: "null operator on embed works on many-to-one is null",
// 	Query:       "/projects?select=name,clients()&clients=is.null",
// 	Expected:    `[{"name":"Orphan"}]`,
// 	Status:      200,
// },
// {
// 	Description: "null operator on embed works on computed many-to-one is null",
// 	Query:       "/projects?select=name,computed_clients()&computed_clients=is.null",
// 	Expected:    `[{"name":"Orphan"}]`,
// 	Status:      200,
// },

// it "works on a one-to-many relationship"
// {
// 	Description: "null operator on embed works on one-to-many not null",
// 	Query:       "/entities?select=name,child_entities()&child_entities=not.is.null",
// 	Expected: `[
// 		{"name":"entity 1"},
// 		{"name":"entity 2"}
// 	]`,
// 	Status: 200,
// },
// {
// 	Description: "null operator on embed works on one-to-many is null",
// 	Query:       "/entities?select=name,child_entities()&child_entities=is.null",
// 	Expected: `[
// 		{"name":"entity 3"},
// 		{"name":null}
// 	]`,
// 	Status: 200,
// },
// {
// 	Description: "null operator on embed works on one-to-many is null with alias",
// 	Query:       "/entities?select=name,childs:child_entities()&childs=is.null",
// 	Expected: `[
// 		{"name":"entity 3"},
// 		{"name":null}
// 	]`,
// 	Status: 200,
// },

// it "works on a many-to-many relationship"
// {
// 	Description: "null operator on embed works on many-to-many not null",
// 	Query:       "/users?select=name,tasks()&tasks.id=eq.1&tasks=not.is.null",
// 	Expected: `[
// 		{"name":"Angela Martin"},
// 		{"name":"Dwight Schrute"}
// 	]`,
// 	Status: 200,
// },
// {
// 	Description: "null operator on embed works on many-to-many is null",
// 	Query:       "/users?select=name,tasks()&tasks.id=eq.1&tasks=is.null",
// 	Expected:    `[{"name":"Michael Scott"}]`,
// 	Status:      200,
// },

// it "works on nested embeds"
// {
// 	Description: "null operator on embed works on nested embeds",
// 	Query:       "/entities?select=name,child_entities(name,grandchild_entities())&child_entities.grandchild_entities=not.is.null&child_entities=not.is.null",
// 	Expected:    `[{"name":"entity 1","child_entities":[{"name":"child entity 1"}, {"name":"child entity 2"}]}]`,
// 	Status:      200,
// },

// it "can do an or across embeds"
// {
// 	Description: "null operator on embed can do an or across embeds",
// 	Query:       "/client?select=*,clientinfo(),contact()&clientinfo.other=ilike.*main*&contact.name=ilike.*tabby*&or=(clientinfo.not.is.null,contact.not.is.null)",
// 	Expected: `[
// 		{"id":1,"name":"Walmart"},
// 		{"id":2,"name":"Target"}
// 	]`,
// 	Status: 200,
// },

// it "only works with is null or is not null operators"
// {
// 	Description: "null operator on embed only works with is null or not is null",
// 	Query:       "/projects?select=name,clients(*)&clients=eq.3",
// 	Status:      400,
// },

// it "doesn't interfere filtering when embedding using the column name"
// {
// 	Description: "null operator on embed doesn't interfere with column name filtering",
// 	Query:       "/projects?select=name,client_id,client:client_id(name)&client_id=eq.2",
// 	Expected: `[
// 		{"name":"IOS","client_id":2,"client":{"name":"Apple"}},
// 		{"name":"OSX","client_id":2,"client":{"name":"Apple"}}
// 	]`,
// 	Status: 200,
// },
