#!/bin/bash

set -ex
COMPOSE="docker compose -f ./compose.yml -f ./mongo-rs-compose.yml"

if [ ! -f mongo-key ]; then
  openssl rand -base64 741 > mongo-key
fi

$COMPOSE up -d mongo-a mongo-b mongo-c
sleep 5
$COMPOSE exec mongo-c mongosh mongodb://root:root@localhost --eval '
try {
  rs.status(); 
} catch (e) {
  rs.initiate({
    _id: "rs0",
    members: [
      { _id: 0, host: "mongo-a:27017" },
      { _id: 1, host: "mongo-b:27017" },
      { _id: 2, host: "mongo-c:27017" },
    ],
  });
}
'
exec $COMPOSE up $@
