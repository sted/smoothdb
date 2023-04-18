

reset-postgrest-tests:
	psql -c "drop database pgrest" && psql -c "create database pgrest" && psql -f ./test/postgrest/fixtures/load.sql pgrest
