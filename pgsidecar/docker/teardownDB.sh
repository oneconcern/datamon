#!/bin/bash
set -e

echo Tearing down DB

if [ -z "$DATA_OUTPUTREPO" ]
then
  echo DATA_OUTPUTREPO environment variable must be set
  exit 1
else
  mkdir -p /tmp/db/output_dump/$OUTPUT_DUMP_PATH/
  echo Dumping existing database
  pg_dump --no-owner -d postgres -U postgres > /tmp/db/output_dump/$OUTPUT_DUMP_PATH/dump.sql
  echo Uploading dump into datamon
  datamon config create --name "Divya Konda" --email divya@oneconcern.com
  datamon bundle upload --path /tmp/db/output_dump/ --message "Adding/editing $OUTPUT_DUMP_PATH" --repo $DATA_OUTPUTREPO
fi