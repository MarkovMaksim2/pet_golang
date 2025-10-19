#!/bin/sh
set -e

echo "Creating storage folder if missing..."
mkdir -p ./storage

if [ ! -f ./storage/userservice.db ]; then
  echo "Creating empty SQLite DB..."
  touch ./storage/userservice.db
fi

echo "Running migrations..."
make migrate

echo "Starting service..."
exec make run_docker
