#!/bin/sh

if [ "$#" -ne 3 ]; then
  echo "Usage: $0 host port username"
  exit 1
fi

psql -c "alter system set primary_conninfo to 'user=$3 host=$1 port=$2 sslmode=prefer sslcompression=1 target_session_attrs=any'" -U $3 postgres
service postgresql restart
