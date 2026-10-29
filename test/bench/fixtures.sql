-- Data set of the k6 comparison benchmark (see run.sh). Deterministic: the
-- same rows on every load, so runs on different days compare.
--
-- Roles: bench_auth is the authenticator both servers connect as (LOGIN
-- NOINHERIT, as the README recommends), bench_anon the anonymous role,
-- bench_user the role of the JWT scenarios.

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'bench_auth') THEN
    CREATE ROLE bench_auth LOGIN NOINHERIT;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'bench_anon') THEN
    CREATE ROLE bench_anon NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'bench_user') THEN
    CREATE ROLE bench_user NOLOGIN;
  END IF;
END $$;
GRANT bench_anon, bench_user TO bench_auth;

DROP SCHEMA IF EXISTS bench CASCADE;
CREATE SCHEMA bench;
SELECT setseed(0.42);

-- small tables: the to-one embed of the original benchmark
CREATE TABLE bench.clients (
  id   int PRIMARY KEY,
  name text NOT NULL
);
INSERT INTO bench.clients
SELECT i, 'client ' || i FROM generate_series(1, 50) i;

CREATE TABLE bench.projects (
  id        int PRIMARY KEY,
  name      text NOT NULL,
  client_id int REFERENCES bench.clients(id)
);
INSERT INTO bench.projects
SELECT i, 'project ' || i, 1 + (i % 50) FROM generate_series(1, 500) i;
CREATE INDEX ON bench.projects(client_id);

-- large tables: 10 000 authors, 200 000 books
CREATE TABLE bench.authors (
  id      int PRIMARY KEY,
  name    text NOT NULL,
  country text NOT NULL
);
INSERT INTO bench.authors
SELECT i, 'author ' || i,
       (ARRAY['it','fr','de','es','us','uk','jp','br'])[1 + (i % 8)]
FROM generate_series(1, 10000) i;

CREATE TABLE bench.books (
  id         int PRIMARY KEY,
  author_id  int NOT NULL REFERENCES bench.authors(id),
  title      text NOT NULL,
  year       int NOT NULL,
  price      numeric(10,2) NOT NULL,
  genre      text NOT NULL,
  tags       text[] NOT NULL,
  meta       jsonb NOT NULL,
  published  date NOT NULL,
  created_at timestamptz NOT NULL
);
INSERT INTO bench.books
SELECT i,
       1 + (i % 10000),
       'book ' || i || ' ' || md5(i::text),
       1950 + (i % 75),
       round((5 + random() * 95)::numeric, 2),
       (ARRAY['fiction','history','science','poetry','travel','crime','fantasy','biography'])[1 + (i % 8)],
       ARRAY['t' || (i % 13), 't' || (i % 7)],
       jsonb_build_object('isbn', lpad(i::text, 13, '0'),
                          'lang', (ARRAY['en','it','fr','de'])[1 + (i % 4)],
                          'pages', 100 + (i % 900)),
       date '1950-01-01' + (i % 27000),
       timestamptz '2020-01-01 00:00:00+00' + (i || ' minutes')::interval
FROM generate_series(1, 200000) i;
CREATE INDEX ON bench.books(author_id);
CREATE INDEX ON bench.books(genre, year);

-- writes: the insert and patch scenarios
CREATE TABLE bench.events (
  id      bigserial PRIMARY KEY,
  kind    text NOT NULL,
  payload jsonb NOT NULL,
  at      timestamptz NOT NULL DEFAULT now()
);
INSERT INTO bench.events (kind, payload)
SELECT 'seed', jsonb_build_object('n', i) FROM generate_series(1, 10000) i;

-- functions
CREATE FUNCTION bench.books_by_genre(g text) RETURNS SETOF bench.books
  STABLE LANGUAGE sql AS $$ SELECT * FROM bench.books WHERE genre = g $$;
CREATE FUNCTION bench.add(a int, b int) RETURNS int
  IMMUTABLE LANGUAGE sql AS $$ SELECT a + b $$;

GRANT USAGE ON SCHEMA bench TO bench_anon, bench_user;
GRANT SELECT ON ALL TABLES IN SCHEMA bench TO bench_anon, bench_user;
GRANT INSERT, UPDATE ON bench.events TO bench_anon, bench_user;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA bench TO bench_anon, bench_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA bench TO bench_anon, bench_user;

VACUUM ANALYZE;
