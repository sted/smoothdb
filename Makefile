

reset-postgrest-tests:
	psql -c "drop database if exists pgrest" && psql -c "create database pgrest" && psql -f ./test/postgrest/fixtures/load.sql pgrest
