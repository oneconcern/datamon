#! /bin/bash
#
# mocked up application interacting with many databases run by sidecar datamon containers
#
# Scenario:
# - db1:5430: a fresh database is initialized, the app writes in it, then it is uploaded to datamon
# - db2:5429: an existing database is retrieved and spun up, the app reads in it, carries out some changes, then it is uploaded to datamon
# - db3:5428: a existing database is retrieved and spun, the app reads in it, then it is thrown away
#
# This program is a stand-in for a data-science application interacting with postgres db

set -e -o pipefail

trim () {
  local var="$*"
  # remove leading whitespace characters
  var="${var#"${var%%[![:space:]]*}"}"
  # remove trailing whitespace characters
  var="${var%"${var##*[![:space:]]}"}"
  echo -n "$var"
}

# convention from sidecar is to create provide initial postgres user.
# other pg users and setup are the responsibility of the application
PG_SU=postgres
PG_OWNER=dbuser
ds1=(5430 "${PG_SU}") # access server with postgres user
ds11=(5430 "${PG_OWNER}") # access server with db owner user
ds2=(5429 "${PG_SU}")
ds3=(5428 "${PG_SU}")

# with some db names
db1=("${ds11[@]}" scratch)
db2=("${ds2[@]}" updatable)
db22=("${ds2[@]}" alternative)  # 2 databases on same server
db3=("${ds3[@]}" readonly)

run_sql() {
  sql_str=$1
  pg_port=$2
  pg_u=$3
  pg_db=$4 # opt-in
  psql -c "${sql_str}" -h localhost --tuples-only -p "${pg_port}" -U "${pg_u}" "${pg_db}"
}

#
# When the app starts, all sidecars have finished their startup phase: all db are available
# let's verify that
pg_isready -h localhost -p "${ds1[0]}"
pg_isready -h localhost -p "${ds2[0]}"
pg_isready -h localhost -p "${ds3[0]}"

# Carry out some updates on db1
echo "*** 1. Setting up a database (superuser: ${PG_SU}, db owner: ${PG_OWNER})"
run_sql "CREATE DATABASE scratch WITH OWNER ${PG_OWNER};" "${ds1[@]}"

echo "*** 2. Creating some schema objects in database \"scratch\""
run_sql 'CREATE TABLE tabla_e (id serial PRIMARY KEY, an_idx integer);' "${db1[@]}"
for idx in $(seq 1 2 9); do
  run_sql "INSERT INTO tabla_e (an_idx) VALUES (${idx});" "${db1[@]}"
done

echo "*** 3. Carry out some queries against db2.updatable (already populated from bundle frozen in this state)"
rows=$(run_sql 'SELECT count(*) FROM tabla_f;' "${db2[@]}")  # this table doesn't change over runs
if [[ "$(trim "${rows}")" != "3" ]] ; then
  echo "expected 3 rows in db2.tabla_f but got: $(trim "${rows}")"
  exit 1
fi

echo "*** 4. Carry out some queries against alternate database on db2.alternative (already populated from bundle frozen in this state)"
rows=$(run_sql 'SELECT count(*) FROM tabla_g;' "${db22[@]}")  # this table doesn't change over runs
if [[ "$(trim "${rows}")" != "4" ]] ; then
  echo "expected 4 rows in db22.tabla_g but got: $(trim "${rows}")"
  exit 1
fi

echo "*** 5. Now updating db2.updatable: this change is going to be saved in a new bundle"
for idx in $(seq 1 2 9); do
  run_sql "INSERT INTO tabla_e (an_idx) VALUES (${idx});" "${db2[@]}"
done

echo "*** 6. Carry out some queries against db3 (already populated from bundle frozen in this state)"
rows=$(run_sql 'SELECT count(*) FROM tabla_f;' "${db3[@]}")  # this table doesn't change over runs
if [[ "$(trim "${rows}")" != "3" ]] ; then
  echo "expected 3 rows in db3.tabla_f but got: $(trim "${rows}")"
  exit 1
fi

echo "*** 7. Now updating db3: this change is going to be thrown away"
for idx in $(seq 1 2 9); do
  run_sql "INSERT INTO tabla_f (msg) VALUES ('message ""${idx}""');" "${db3[@]}"
done

echo "*** 8. Now exiting: the application wrapper takes over and saves all databases but db3"
