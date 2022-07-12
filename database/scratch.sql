
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