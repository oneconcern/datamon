#!/bin/bash
set -e

echo Initializing DB

echo Loading PostGIS extensions
psql -U postgres -d postgres -c "CREATE EXTENSION IF NOT EXISTS postgis"

if [ -z "$DATA_INPUTREPO" ]
then
  echo DATA_INPUTREPO environment variable must be set
  exit 1
else
  mkdir -p /tmp/db/data_dump/
  datamon bundle download --repo $DATA_INPUTREPO --destination /tmp/db/data_dump/
  for dump in /tmp/db/data_dump/$INPUT_DUMP_PATH/*; do
    echo Restoring "$dump"
    psql -d postgres -U postgres < "$dump"
  done;
fi