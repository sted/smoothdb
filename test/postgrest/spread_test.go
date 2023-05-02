package postgrest

import (
	"testing"

	"github.com/smoothdb/smoothdb/test"
)

func TestPostgREST_Spread(t *testing.T) {

	tests := []test.Test{

		// describe "spread embeds" $ do
		// it "works on a many-to-one relationship" $ do
		//   get "/projects?select=id,...clients(client_name:name)" `shouldRespondWith`
		//     [json|[
		//       {"id":1,"client_name":"Microsoft"},
		//       {"id":2,"client_name":"Microsoft"},
		//       {"id":3,"client_name":"Apple"},
		//       {"id":4,"client_name":"Apple"},
		//       {"id":5,"client_name":null}
		//     ]|]
		//     { matchStatus  = 200
		//     , matchHeaders = [matchContentTypeJson]
		//     }
		{
			Description: "works on a many-to-one relationship",
			Method:      "GET",
			Query:       "/projects?select=id,...clients(client_name:name)",
			Expected: `[
				       {"id":1,"client_name":"Microsoft"},
				       {"id":2,"client_name":"Microsoft"},
				       {"id":3,"client_name":"Apple"},
				       {"id":4,"client_name":"Apple"},
				       {"id":5,"client_name":null}
				     ]`,
			Status: 200,
		},

		//   get "/grandchild_entities?select=name,...child_entities(parent_name:name,...entities(grandparent_name:name))&limit=3" `shouldRespondWith`
		//     [json|[
		//       {"name":"grandchild entity 1","parent_name":"child entity 1","grandparent_name":"entity 1"},
		//       {"name":"grandchild entity 2","parent_name":"child entity 1","grandparent_name":"entity 1"},
		//       {"name":"grandchild entity 3","parent_name":"child entity 2","grandparent_name":"entity 1"}
		//     ]|]
		//     { matchStatus  = 200
		//     , matchHeaders = [matchContentTypeJson]
		//     }
		{
			Description: "works on a many-to-one relationship",
			Method:      "GET",
			Query:       "/grandchild_entities?select=name,...child_entities(parent_name:name,...entities(grandparent_name:name))&limit=3",
			Expected: `[
				       {"name":"grandchild entity 1","parent_name":"child entity 1","grandparent_name":"entity 1"},
				       {"name":"grandchild entity 2","parent_name":"child entity 1","grandparent_name":"entity 1"},
				       {"name":"grandchild entity 3","parent_name":"child entity 2","grandparent_name":"entity 1"}
				     ]`,
			Status: 200,
		},
		//   get "/videogames?select=name,...computed_designers(designer_name:name)" `shouldRespondWith`
		//     [json|[
		//       {"name":"Civilization I","designer_name":"Sid Meier"},
		//       {"name":"Civilization II","designer_name":"Sid Meier"},
		//       {"name":"Final Fantasy I","designer_name":"Hironobu Sakaguchi"},
		//       {"name":"Final Fantasy II","designer_name":"Hironobu Sakaguchi"}
		//     ]|]
		//     { matchStatus  = 200
		//     , matchHeaders = [matchContentTypeJson]
		//     }

		// @@ computed table

		// {
		// 	Description: "works on a many-to-one relationship",
		// 	Method:      "GET",
		// 	Query:       "/videogames?select=name,...computed_designers(designer_name:name)",
		// 	Expected: `[
		// 		       {"name":"Civilization I","designer_name":"Sid Meier"},
		// 		       {"name":"Civilization II","designer_name":"Sid Meier"},
		// 		       {"name":"Final Fantasy I","designer_name":"Hironobu Sakaguchi"},
		// 		       {"name":"Final Fantasy II","designer_name":"Hironobu Sakaguchi"}
		// 		     ]`,
		// 	Status: 200,
		// },

		// @@ computed table

		// {
		// 	Description: "works on a many-to-one relationship",
		// 	Method:      "GET",
		// 	Query:       "/videogames?select=name,...computed_designers(designer_name:name)",
		// 	Expected: `[
		// 		       {"name":"Civilization I","designer_name":"Sid Meier"},
		// 		       {"name":"Civilization II","designer_name":"Sid Meier"},
		// 		       {"name":"Final Fantasy I","designer_name":"Hironobu Sakaguchi"},
		// 		       {"name":"Final Fantasy II","designer_name":"Hironobu Sakaguchi"}
		// 		     ]`,
		// 	Status: 200,
		// },

		// it "works inside a normal embed" $
		//   get "/grandchild_entities?select=name,child_entity:child_entities(name,...entities(parent_name:name))&limit=1" `shouldRespondWith`
		//     [json|[
		//       {"name":"grandchild entity 1","child_entity":{"name":"child entity 1","parent_name":"entity 1"}}
		//     ]|]
		//     { matchStatus  = 200
		//     , matchHeaders = [matchContentTypeJson]
		//     }
		{
			Description: "works inside a normal embed",
			Method:      "GET",
			Query:       "/grandchild_entities?select=name,child_entity:child_entities(name,...entities(parent_name:name))&limit=1",
			Expected: `[
				       {"name":"grandchild entity 1","child_entity":{"name":"child entity 1","parent_name":"entity 1"}}
				     ]`,
			Status: 200,
		},
		// it "works on a one-to-one relationship" $
		//   get "/country?select=name,...capital(capital:name)" `shouldRespondWith`
		//     [json|[
		//       {"name":"Afghanistan","capital":"Kabul"},
		//       {"name":"Algeria","capital":"Algiers"}
		//     ]|]
		//     { matchStatus  = 200
		//     , matchHeaders = [matchContentTypeJson]
		//     }
		{
			Description: "works on a one-to-one relationship",
			Method:      "GET",
			Query:       "/country?select=name,...capital(capital:name)",
			Expected: `[
				       {"name":"Afghanistan","capital":"Kabul"},
				       {"name":"Algeria","capital":"Algiers"}
				     ]`,
			Status: 200,
		},
		// it "fails when is not a to-one relationship" $ do
		//   get "/clients?select=*,...projects(*)" `shouldRespondWith`
		//     [json|{
		//       "code":"PGRST119",
		//       "details":"'clients' and 'projects' do not form a many-to-one or one-to-one relationship",
		//       "hint":null,
		//       "message":"A spread operation on 'projects' is not possible"
		//     }|]
		//     { matchStatus  = 400
		//     , matchHeaders = [matchContentTypeJson]
		//     }
		{
			Description: "fails when is not a to-one relationship",
			Method:      "GET",
			Query:       "/clients?select=*,...projects(*)",
			Expected:    ``,
			Status:      400,
		},
		//   get "/designers?select=*,...computed_videogames(*)" `shouldRespondWith`
		//     [json|{
		//       "code":"PGRST119",
		//       "details":"'designers' and 'computed_videogames' do not form a many-to-one or one-to-one relationship",
		//       "hint":null,
		//       "message":"A spread operation on 'computed_videogames' is not possible"
		//     }|]
		//     { matchStatus  = 400
		//     , matchHeaders = [matchContentTypeJson]
		//     }

		// it "can include or exclude attributes of the junction on a m2m" $ do
		//   get "/users?select=*,tasks:users_tasks(*,...tasks(*))&limit=1" `shouldRespondWith`
		//     [json|[{
		//       "id":1,"name":"Angela Martin",
		//       "tasks": [
		//         {"user_id":1,"task_id":1,"id":1,"name":"Design w7","project_id":1},
		//         {"user_id":1,"task_id":2,"id":2,"name":"Code w7","project_id":1},
		//         {"user_id":1,"task_id":3,"id":3,"name":"Design w10","project_id":2},
		//         {"user_id":1,"task_id":4,"id":4,"name":"Code w10","project_id":2}
		//       ]
		//     }]|]
		//     { matchStatus  = 200
		//     , matchHeaders = [matchContentTypeJson]
		//     }
		{
			Description: "can include or exclude attributes of the junction on a m2m",
			Method:      "GET",
			Query:       "/users?select=*,tasks:users_tasks(*,...tasks(*))&limit=1",
			Expected: `[{
				      "id":1,"name":"Angela Martin",
				      "tasks": [
				        {"user_id":1,"task_id":1,"id":1,"name":"Design w7","project_id":1},
				        {"user_id":1,"task_id":2,"id":2,"name":"Code w7","project_id":1},
				        {"user_id":1,"task_id":3,"id":3,"name":"Design w10","project_id":2},
				        {"user_id":1,"task_id":4,"id":4,"name":"Code w10","project_id":2}
				      ]
				    }]`,
			Status: 200,
		},
		//   get "/users?select=*,tasks:users_tasks(...tasks(*))&limit=1" `shouldRespondWith`
		//     [json|[{
		//       "id":1,"name":"Angela Martin",
		//       "tasks":[
		//         {"id":1,"name":"Design w7","project_id":1},
		//         {"id":2,"name":"Code w7","project_id":1},
		//         {"id":3,"name":"Design w10","project_id":2},
		//         {"id":4,"name":"Code w10","project_id":2}
		//       ]
		//     }]|]
		//     { matchStatus  = 200
		//     , matchHeaders = [matchContentTypeJson]
		{
			Description: "can include or exclude attributes of the junction on a m2m",
			Method:      "GET",
			Query:       "/users?select=*,tasks:users_tasks(...tasks(*))&limit=1",
			Expected: `[{
				      "id":1,"name":"Angela Martin",
				      "tasks":[
				        {"id":1,"name":"Design w7","project_id":1},
				        {"id":2,"name":"Code w7","project_id":1},
				        {"id":3,"name":"Design w10","project_id":2},
				        {"id":4,"name":"Code w10","project_id":2}
				      ]
				    }]`,
			Status: 200,
		},
	}

	test.Execute(t, testConfig, tests)
}
