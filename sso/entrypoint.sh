#!/bin/sh
set -e

echo "Creating storage folder if missing..."
mkdir -p ./storage

if [ ! -f ./storage/userservice.db ]; then
  echo "Creating empty SQLite DB..."
  touch ./storage/sso.db
fi

make migrate

exec make run_docker
