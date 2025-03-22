#!/bin/bash
echo "bobus123123123" > key
chmod 600 key

mongod --config /mongo/mongod.conf &

if [[ "$1" == "create" && -e "init_done" ]]; then
    echo Initializing replicaSet
    sleep 10;
    mongosh --eval '
    rs.initiate({
        _id: "pbrs",
        members: [
            {_id: 0, host: "mongo-a:27017"},
            {_id: 1, host: "mongo-b:27017"},
            {_id: 2, host: "mongo-c:27017"},
        ]
    });
    '
    touch init_done
fi

wait
