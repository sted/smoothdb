
SELECT c.conrelid::regclass AS table_from,
       c.conname,
       c.contype,
       pg_get_constraintdef(c.oid),
       a.attname
       FROM pg_constraint c
            CROSS JOIN LATERAL unnest(c.conkey) ak(k)
            INNER JOIN pg_attribute a
                       ON a.attrelid = c.conrelid
                          AND a.attnum = ak.k
       WHERE c.conrelid::regclass::text = 'test'
       ORDER BY c.contype;

SELECT c.conrelid::regclass AS table_from,
       c.conname,
       c.contype,
       pg_get_constraintdef(c.oid),
       a.attname
       FROM pg_constraint c
            CROSS JOIN LATERAL unnest(c.conkey) ak(k)
            INNER JOIN pg_attribute a
                       ON a.attrelid = c.conrelid
                          AND a.attnum = ak.k
       WHERE c.conrelid::regclass::text = 'test'
       GROUP BY c.contype
       ORDER BY c.contype;

SELECT 
    "test"."projects"."id", 
    "test"."projects"."name", 
    row_to_json("projects_clients".*) AS "clients", 
    COALESCE( "projects_tasks"."_projects_tasks", '[]') AS "tasks" 
FROM "test"."projects" 
LEFT JOIN LATERAL ( 
    SELECT "test"."clients".* 
    FROM "test"."clients"  
    WHERE "test"."clients"."id" = "test"."projects"."client_id"   
) AS "projects_clients" ON TRUE 
LEFT JOIN LATERAL ( 
    SELECT json_agg("_projects_tasks") AS "_projects_tasks"
    FROM (
        SELECT "test"."tasks"."id", "test"."tasks"."name" 
        FROM "test"."tasks"  
        WHERE "test"."tasks"."project_id" = "test"."projects"."id"   
    ) AS "_projects_tasks" 
) AS "projects_tasks" ON TRUE 
WHERE  "test"."projects"."id" = 1  

SELECT 
    "test"."projects"."id", 
    "test"."projects"."name", 
    row_to_json("projects_clients".*) AS "clients", 
    COALESCE( "projects_tasks"."_projects_tasks", '[]') AS "tasks" 
FROM "test"."projects" 
LEFT JOIN LATERAL ( 
    SELECT * 
    FROM "test"."clients"  
    WHERE "test"."clients"."id" = "test"."projects"."client_id"   
) AS "projects_clients" ON TRUE 
LEFT JOIN LATERAL ( 
    SELECT json_agg("_projects_tasks") AS "_projects_tasks"
    FROM (
        SELECT "id", "name" 
        FROM "test"."tasks"  
        WHERE "test"."tasks"."project_id" = "test"."projects"."id"   
    ) AS "_projects_tasks" 
) AS "projects_tasks" ON TRUE 
WHERE  "test"."projects"."id" = 1  


SELECT 
    "test"."clients"."id", 
    COALESCE( "clients_projects"."_clients_projects", '[]') AS "projects"
FROM "test"."clients" 
LEFT JOIN LATERAL ( 
    SELECT json_agg("_clients_projects") AS "_clients_projects"
    FROM (
        SELECT "test"."projects"."id", 
        COALESCE( "projects_tasks"."_projects_tasks", '[]') AS "tasks" 
        FROM "test"."projects" 
        LEFT JOIN LATERAL ( 
            SELECT json_agg("_projects_tasks") AS "_projects_tasks"
            FROM (
                SELECT "test"."tasks"."id", "test"."tasks"."name" 
                FROM "test"."tasks"  
                WHERE  "test"."tasks"."name" like $1 AND "test"."tasks"."project_id" = "test"."projects"."id"   
            ) AS "_projects_tasks" 
        ) AS "projects_tasks" ON TRUE 
        WHERE "test"."projects"."client_id" = "test"."clients"."id"   
    ) AS "_clients_projects" 
) AS "clients_projects" ON TRUE  




SELECT "test"."projects"."id", "test"."projects"."name",  
row_to_json("projects_clients".*) AS "clients",  COALESCE("projects_tasks"."_projects_tasks", '[]') AS "tasks" FROM "test"."projects"  
LEFT JOIN LATERAL ( SELECT "test"."clients".* FROM test.clients WHERE "test"."clients"."id" = "test"."projects"."client_id") AS "projects_clients" ON TRUE LEFT JOIN LATERAL ( 
    SELECT json_agg("_projects_tasks") AS "_projects_tasks" FROM ( 
        SELECT "test"."tasks"."id", "test"."tasks"."name" FROM test.tasks WHERE "test"."tasks"."project_id" = "test"."projects"."id" AND "test"."tasks"."name" = 'Code w7') AS "_projects_tasks") AS "projects_tasks" ON TRUE WHERE "test"."projects"."id" = '1'