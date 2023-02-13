
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



WITH pgrst_source AS (
    WITH pgrst_payload AS (
        SELECT $1 AS json_data
    ), 
    pgrst_body AS ( 
        SELECT 
            CASE WHEN json_typeof(json_data) = 'array' 
            THEN json_data 
            ELSE json_build_array(json_data) END AS val 
        FROM pgrst_payload
    ) 
    UPDATE "test"."items" SET "id" = _."id" FROM (
        SELECT * FROM json_populate_recordset (null::"test"."items" , (SELECT val FROM pgrst_body))
    ) _  
    WHERE  "test"."items"."id" = $2 RETURNING "test"."items"."always_true", "test"."items"."id"
) 
SELECT '' AS total_result_set, pg_catalog.count(_postgrest_t) AS page_total, array[]::text[] AS header, coalesce(json_agg(_postgrest_t), '[]')::character varying AS body, nullif(current_setting('response.headers', true), '') AS response_headers, nullif(current_setting('response.status', true), '') AS response_status FROM (
    SELECT "items"."id", "items"."always_true" FROM "pgrst_source" AS "items"    ) _postgrest_t



UPDATE "test"."items" SET "id" = _."id" WHERE  "test"."items"."id" = $2 RETURNING "test"."items"."always_true", "test"."items"."id"


WITH pgrst_source AS (
    WITH pgrst_payload AS (
        SELECT $1 AS json_data
    ), 
    pgrst_body AS ( 
        SELECT CASE WHEN json_typeof(json_data) = 'array' THEN json_data ELSE json_build_array(json_data) END AS val 
        FROM pgrst_payload
    ) 
    UPDATE "test"."students" SET "id" = _."id" FROM (
        SELECT * FROM json_populate_recordset (null::"test"."students" , (SELECT val FROM pgrst_body) )
    ) _  
    WHERE  "test"."students"."id" = $2 RETURNING "test"."students"."code", "test"."students"."id", "test"."students"."name"
) 
SELECT '' AS total_result_set, pg_catalog.count(_postgrest_t) AS page_total, array[]::text[] AS header, coalesce(json_agg(_postgrest_t), '[]')::character varying AS body, nullif(current_setting('response.headers', true), '') AS response_headers, nullif(current_setting('response.status', true), '') AS response_status 
FROM (
    SELECT "students"."name", row_to_json("students_students_info_1".*) AS "students_info" 
    FROM "pgrst_source" AS "students" 
    LEFT JOIN LATERAL ( 
        SELECT "students_info_1"."address" FROM "test"."students_info" AS "students_info_1"  
        WHERE "students_info_1"."id" = "students"."id" AND "students_info_1"."code" = "students"."code"   
    ) AS "students_students_info_1" ON TRUE   
) _postgrest_t

UPDATE "test"."students" SET "id" = 1 
FROM "test"."students_info" 
WHERE  "test"."students"."id" = 1 and "test"."students"."id" = "test"."students_info"."id" 
RETURNING "test"."students"."code", "test"."students"."id", "test"."students"."name"


WITH pgrst_source AS (
    WITH pgrst_payload AS (
        SELECT $1 AS json_data
    ), pgrst_body AS (
        SELECT CASE WHEN json_typeof(json_data) = 'array' THEN json_data ELSE json_build_array(json_data) END AS val 
        FROM pgrst_payload
    ) 
    UPDATE "test"."articles" SET "body" = _."body", "id" = _."id" 
    FROM (
        SELECT * FROM json_populate_recordset (null::"test"."articles" , (SELECT val FROM pgrst_body))
    ) _  
    RETURNING "test"."articles".*
) 
SELECT '' AS total_result_set, pg_catalog.count(_postgrest_t) AS page_total, array[]::text[] AS header, coalesce(json_agg(_postgrest_t), '[]')::character varying AS body, nullif(current_setting('response.headers', true), '') AS response_headers, nullif(current_setting('response.status', true), '') AS response_status FROM (SELECT "articles".* FROM "pgrst_source" AS "articles"    ) _postgrest_t



WITH pgrst_payload AS (
    SELECT $1 AS json_data
), 
pgrst_body AS (
        SELECT CASE WHEN json_typeof(json_data) = 'array' THEN json_data ELSE json_build_array(json_data) END AS val 
        FROM pgrst_payload
) 
    UPDATE "test"."articles" SET "body" = _."body", "id" = _."id" 
    FROM (
        SELECT * FROM json_populate_recordset (null::"test"."articles" , (SELECT val FROM pgrst_body))
    ) _  
    RETURNING "test"."articles".*



