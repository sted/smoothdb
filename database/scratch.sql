
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






WITH pgrst_source AS (
    WITH pgrst_payload AS (SELECT $1 AS json_data), 
    pgrst_body AS (
         SELECT CASE WHEN json_typeof(json_data) = 'array' THEN json_data ELSE json_build_array(json_data) END AS val FROM pgrst_payload
    ) 
    INSERT INTO "test"."child_entities"("id", "name", "parent_id") 
    SELECT "id", "name", "parent_id" 
    FROM json_populate_recordset (null::"test"."child_entities", (SELECT val FROM pgrst_body)) _  
    RETURNING "test"."child_entities"."id", "test"."child_entities"."parent_id"
) 
SELECT '' AS total_result_set, pg_catalog.count(_postgrest_t) AS page_total, array[]::text[] AS header, coalesce(json_agg(_postgrest_t), '[]')::character varying AS body, nullif(current_setting('response.headers', true), '') AS response_headers, nullif(current_setting('response.status', true), '') AS response_status 
FROM (
    SELECT "child_entities"."id", row_to_json("child_entities_entities_1".*) AS "entities" 
    FROM "pgrst_source" AS "child_entities" 
    LEFT JOIN LATERAL ( 
        SELECT "entities_1"."id" FROM "test"."entities" AS "entities_1"  WHERE  ( "entities_1"."id" = $2 OR  "entities_1"."id" = $3) AND "entities_1"."id" = "child_entities"."parent_id"   
    ) AS "child_entities_entities_1" ON TRUE   
) _postgrest_t

SELECT "test"."entities"."id",  
    COALESCE("entities_child_entities"."_entities_child_entities", '[]') AS "child_entities" 
FROM "test"."entities"  
LEFT JOIN LATERAL ( 
    SELECT json_agg("_entities_child_entities") AS "_entities_child_entities" 
    FROM ( 
        SELECT "test"."child_entities"."id" 
        FROM test.child_entities 
        WHERE "test"."child_entities"."parent_id" = "test"."entities"."id" AND ("test"."child_entities"."id" = '1' OR "test"."child_entities"."name" = 'child entity 2')     
    ) AS "_entities_child_entities"
) AS "entities_child_entities" ON TRUE

WITH child_entities  AS (
    INSERT INTO child_entities (id, name, parent_id) 
    VALUES 
        ()
        ()
        ()
    RETURNING id
)
SELECT 

WITH _source AS (
    INSERT INTO "test"."child_entities" (id, name, parent_id) 
    VALUES ($1, $2, $3), ($4, $5, $6), ($7, $8, $9) 
    RETURNING "test"."child_entities"."id", "test"."child_entities"."parent_id"
) 
SELECT "_source"."id",  row_to_json("child_entities_entities".*) AS "entities" 
FROM _source  
LEFT JOIN LATERAL ( 
    SELECT "test"."entities"."id" 
    FROM test.entities 
    WHERE "test"."entities"."id" = "_source"."parent_id" AND ("test"."entities"."id" = '2' OR "test"."entities"."id" = '3') 
    ORDER BY "test"."entities"."id"
) AS "child_entities_entities" ON TRUE

WITH pgrst_source AS (
    WITH pgrst_payload AS (
        SELECT $1 AS json_data
    ), 
    pgrst_body AS (
        SELECT CASE WHEN json_typeof(json_data) = 'array' THEN json_data ELSE json_build_array(json_data) END AS val FROM pgrst_payload
    ) 
    UPDATE "test"."web_content" 
    SET "name" = _."name" 
    FROM (
        SELECT * 
        FROM json_populate_recordset (null::"test"."web_content" , (SELECT val FROM pgrst_body) )
    ) _  
    WHERE  "test"."web_content"."id" = $2 
    RETURNING "test"."web_content"."id", "test"."web_content"."name"
) 
SELECT '' AS total_result_set, pg_catalog.count(_postgrest_t) AS page_total, array[]::text[] AS header, coalesce(json_agg(_postgrest_t), '[]')::character varying AS body, nullif(current_setting('response.headers', true), '') AS response_headers, nullif(current_setting('response.status', true), '') AS response_status 
FROM (
    SELECT "web_content"."id", "web_content"."name", COALESCE( "web_content_web_content_1"."web_content_web_content_1", '[]') AS "web_content" 
    FROM "pgrst_source" AS "web_content" 
    LEFT JOIN LATERAL ( 
        SELECT json_agg("web_content_web_content_1") AS "web_content_web_content_1"
        FROM (
            SELECT "web_content_1"."name" 
            FROM "test"."web_content" AS "web_content_1"  
            WHERE "web_content_1"."p_web_id" = "web_content"."id"   
        ) AS "web_content_web_content_1" 
    ) AS "web_content_web_content_1" ON TRUE   
) _postgrest_t

WITH _source AS (
    UPDATE "test"."web_content" 
    SET name = $1 
    WHERE "test"."web_content"."id" = '0' 
    RETURNING "test"."web_content"."id", "test"."web_content"."name"
) 
SELECT  FROM _source

WITH _source AS (
    UPDATE \"test\".\"web_content\" 
    SET name = $1 
    WHERE \"test\".\"web_content\".\"id\" = '0'
    RETURNING \"test\".\"web_content\".\"id\", \"test\".\"web_content\".\"name\", \"test\".\"web_content\".\"p_web_id\"
) 
SELECT \"_source\".\"id\", \"_source\".\"name\",  row_to_json(\"web_content_web_content\".*) AS \"web_content\" 
FROM _source  
LEFT JOIN LATERAL ( 
    SELECT \"test\".\"web_content\".\"name\" 
    FROM test.web_content 
    WHERE \"test\".\"web_content\".\"id\" = \"_source\".\"p_web_id\" AND \"test\".\"web_content\".\"id\" = '0'
) AS \"web_content_web_content\" ON TRUE


WITH _source AS (
    UPDATE "test"."web_content" 
    SET name = $1 
    WHERE "test"."web_content"."id" = '0' 
    RETURNING "test"."web_content"."id", "test"."web_content"."name", "test"."web_content"."id"
)
SELECT "_source"."id", "_source"."name",  COALESCE("web_content_web_content"."_web_content_web_content", '[]') AS "web_content" 
FROM _source  
LEFT JOIN LATERAL ( 
    SELECT json_agg("_web_content_web_content") AS "_web_content_web_content" 
    FROM ( 
        SELECT "test"."web_content"."name" 
        FROM test.web_content 
        WHERE "test"."web_content"."p_web_id" = "_source"."id" AND "test"."web_content"."id" = '0'
    ) AS "_web_content_web_content"
) AS "web_content_web_content" ON TRUE

WITH _source AS (
    UPDATE "test"."web_content"
    SET name = 'pippo'
    WHERE "test"."web_content"."id" = '0'
    RETURNING "test"."web_content"."id", "test"."web_content"."name"
)
SELECT "_source"."id", "_source"."name",  COALESCE("web_content_web_content"."_web_content_web_content", '[]') AS "web_content"
FROM _source
LEFT JOIN LATERAL (
    SELECT json_agg("_web_content_web_content") AS "_web_content_web_content"
    FROM (
        SELECT "test"."web_content"."name"
        FROM test.web_content 
        WHERE "test"."web_content"."p_web_id" = "_source"."id"
    ) AS "_web_content_web_content"
) AS "web_content_web_content" ON TRUE;


SELECT 
    c.conname name, 
    c.contype type, 
    n.nspname||'.'||cls.relname table, 
    array_agg(a.attname order by a.attname) cols,
    n2.nspname||'.'||cls2.relname ftable, 
    array_agg(a2.attname order by a2.attname) fcols,
    (n.nspname, cls.relname) = (n2.nspname, cls2.relname) is_self,
    pg_get_constraintdef(c.oid, true) def
FROM pg_catalog.pg_constraint c
JOIN pg_catalog.pg_class cls ON c.conrelid = cls.oid
JOIN pg_catalog.pg_namespace n ON n.oid = cls.relnamespace
JOIN pg_catalog.pg_attribute a ON c.conrelid = a.attrelid AND a.attnum = ANY(c.conkey)
LEFT JOIN pg_catalog.pg_class cls2 ON c.confrelid = cls2.oid
LEFT JOIN pg_catalog.pg_namespace n2 ON n2.oid = cls2.relnamespace
LEFT JOIN pg_catalog.pg_attribute a2 ON c.confrelid = a2.attrelid AND a2.attnum = ANY(c.confkey)
WHERE n.nspname !~ '^pg_'
GROUP BY c.oid, n.nspname, cls.relname, n2.nspname, cls2.relname ORDER BY cls.relname;

SELECT
    c.conname name,
    c.contype type, 
    ns1.nspname||'.'||cls1.relname table,
    columns.cols,
    ns2.nspname||'.'||cls2.relname ftable,
    columns.fcols,
    (ns1.nspname, cls1.relname) = (ns2.nspname, cls2.relname) is_self,
    pg_get_constraintdef(c.oid, true) def
FROM pg_constraint c
JOIN LATERAL (
    SELECT
    array_agg(cols.attname order by ord) cols,
    coalesce(array_agg(fcols.attname order by ord) filter (where fcols.attname is not null), '{}') fcols
    FROM unnest(c.conkey, c.confkey) WITH ORDINALITY AS _(col, fcol, ord)
    JOIN pg_attribute cols ON cols.attrelid = c.conrelid AND cols.attnum = col
    LEFT JOIN pg_attribute fcols ON fcols.attrelid = c.confrelid AND fcols.attnum = fcol
) AS columns ON TRUE
JOIN pg_namespace ns1 ON ns1.oid = c.connamespace
JOIN pg_class cls1 ON cls1.oid = c.conrelid
LEFT JOIN pg_class cls2 ON cls2.oid = c.confrelid
LEFT JOIN pg_namespace ns2 ON ns2.oid = cls2.relnamespace
WHERE ns1.nspname !~ '^pg_'
ORDER BY cls.relname;


WITH pgrst_source AS ( 
    SELECT "test"."students"."name", row_to_json("students_students_info_1".*) AS "students_info" 
    FROM "test"."students" 
    LEFT JOIN LATERAL ( 
        SELECT "students_info_1"."address" 
        FROM "test"."students_info" AS "students_info_1"  
        WHERE "students_info_1"."id" = "test"."students"."id" AND "students_info_1"."code" = "test"."students"."code"   
    ) AS "students_students_info_1" ON TRUE 
    WHERE  "test"."students"."id" = $1   
)  
SELECT null::bigint AS total_result_set, pg_catalog.count(_postgrest_t) AS page_total, coalesce(json_agg(_postgrest_t), '[]')::character varying AS body, nullif(current_setting('response.headers', true), '') AS response_headers, nullif(current_setting('response.status', true), '') AS response_status 
FROM ( SELECT * FROM pgrst_source ) _postgrest_t

WITH pgrst_source AS (
    WITH pgrst_payload AS (
        SELECT $1 AS json_data), pgrst_body AS ( 
            SELECT CASE WHEN json_typeof(json_data) = 'array' 
            THEN json_data ELSE json_build_array(json_data) 
            END AS val 
            FROM pgrst_payload
        ) 
        INSERT INTO "test"."tiobe_pls"("name", "rank") 
        SELECT "name", "rank" 
        FROM json_populate_recordset (null::"test"."tiobe_pls", (SELECT val FROM pgrst_body)
        ) _  
        ON CONFLICT("name") 
        DO UPDATE SET "name" = EXCLUDED."name", "rank" = EXCLUDED."rank" 
        RETURNING "test"."tiobe_pls".*
) 
SELECT '' AS total_result_set, pg_catalog.count(_postgrest_t) AS page_total, array[]::text[] AS header, coalesce(json_agg(_postgrest_t), '[]')::character varying AS body, nullif(current_setting('response.headers', true), '') AS response_headers, nullif(current_setting('response.status', true), '') AS response_status FROM (SELECT "tiobe_pls".* FROM "pgrst_source" AS "tiobe_pls"    ) _postgrest_t
