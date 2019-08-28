#!/bin/bash

set -e

pull() {
    docker pull golang:1.12.1
    docker pull bitnami/etcd:latest
}

build() {
    IMAGE=grpc-id/bin:latest
    docker build -t ${IMAGE} . -f Dockerfile.bin

    IMAGE=grpc-id/svr:0.1
    docker build -t ${IMAGE} . -f Dockerfile.svr \
        --build-arg CREATE_AT=`date +%Y-%m-%dT%H:%M:%S` \
        --build-arg PORT=50001

    IMAGE=grpc-id/gw:0.1
    docker build -t ${IMAGE} . -f Dockerfile.gw \
        --build-arg CREATE_AT=`date +%Y-%m-%dT%H:%M:%S` \
        --build-arg PORT=60001

    IMAGE=grpc-id/cli:0.1
    docker build -t ${IMAGE} . -f Dockerfile.cli \
        --build-arg CREATE_AT=`date +%Y-%m-%dT%H:%M:%S`
}

build_compose() {
    #IMAGE=grpc-id/bin:latest
    #docker build -t ${IMAGE} . -f Dockerfile.bin

    sed -i '' 's/^ENTRYPOINT/#ENTRYPOINT/g' Dockerfile.svr
    sed -i '' 's/^ENTRYPOINT/#ENTRYPOINT/g' Dockerfile.gw
    sed -i '' 's/^ENTRYPOINT/#ENTRYPOINT/g' Dockerfile.cli

    IMAGE=grpc-id/svr:0.2
    docker build -t ${IMAGE} . -f Dockerfile.svr \
        --build-arg CREATE_AT=`date +%Y-%m-%dT%H:%M:%S` \
        --build-arg PORT=50001

    IMAGE=grpc-id/gw:0.2
    docker build -t ${IMAGE} . -f Dockerfile.gw \
        --build-arg CREATE_AT=`date +%Y-%m-%dT%H:%M:%S` \
        --build-arg PORT=60001

    IMAGE=grpc-id/cli:0.2
    docker build -t ${IMAGE} . -f Dockerfile.cli \
        --build-arg CREATE_AT=`date +%Y-%m-%dT%H:%M:%S`

    sed -i '' 's/^#ENTRYPOINT/ENTRYPOINT/g' Dockerfile.svr
    sed -i '' 's/^#ENTRYPOINT/ENTRYPOINT/g' Dockerfile.gw
    sed -i '' 's/^#ENTRYPOINT/ENTRYPOINT/g' Dockerfile.cli
}

start_etcd() {
    #Step 1: Create a network
    docker network create app-tier --driver bridge
    #Step 2: Launch the etcd server instance
    docker run -d --name etcd-server \
        --network app-tier \
        --publish 2379:2379 \
        --publish 2380:2380 \
        --env ALLOW_NONE_AUTHENTICATION=yes \
        --env ETCD_ADVERTISE_CLIENT_URLS=http://etcd-server:2379 \
        bitnami/etcd:latest
    #Step 3: Launch your etcd client instance
    docker run -it --rm \
        --network app-tier \
        --env ALLOW_NONE_AUTHENTICATION=yes \
        bitnami/etcd:latest \
        etcdctl --endpoints http://etcd-server:2379 put /message Hello
}

stop_etcd() {
    CONTAINER=etcd-server
    docker stop ${CONTAINER}
}

ctl_etcd() {
    docker exec -it etcd-server etcdctl get /etcdv3_resolver/grpc_id_service/localhost:50002
}

svr_ip() {
    docker network inspect app-tier
    CONTAINER=grpc_id_svr_1
    #ip addr show|grep inet|grep -v 127.0.0.1|awk '{print $2}'|tr -d "/16"
    docker exec -it ${CONTAINER} \
        /sbin/ifconfig -a|grep inet|grep -v 127.0.0.1|grep -v inet6|awk '{print $2}'|tr -d "addr:"
    CONTAINER=grpc_id_svr_2
    docker exec -it ${CONTAINER} \
        /sbin/ifconfig -a|grep inet|grep -v 127.0.0.1|grep -v inet6|awk '{print $2}'|tr -d "addr:"
    CONTAINER=grpc_id_svr_3
    docker exec -it ${CONTAINER} \
        /sbin/ifconfig -a|grep inet|grep -v 127.0.0.1|grep -v inet6|awk '{print $2}'|tr -d "addr:"
}

start_svr() {
    IMAGE=grpc-id/svr:0.1
    CONTAINER=grpc_id_svr_1
    docker run --rm -tid \
        --network app-tier \
        --ip 172.20.0.7 \
        --name ${CONTAINER} ${IMAGE} \
        -host 172.20.0.7 \
        -port 50001 \
        -reg http://etcd-server:2379

    IMAGE=grpc-id/svr:0.1
    CONTAINER=grpc_id_svr_2
    docker run --rm -tid \
        --network app-tier \
        --ip 172.20.0.8 \
        --name ${CONTAINER} ${IMAGE} \
        -host 172.20.0.8 \
        -port 50001 \
        -reg http://etcd-server:2379

    IMAGE=grpc-id/svr:0.1
    CONTAINER=grpc_id_svr_3
    docker run --rm -tid \
        --network app-tier \
        --ip 172.20.0.9 \
        --name ${CONTAINER} ${IMAGE} \
        -host 172.20.0.9 \
        -port 50001 \
        -reg http://etcd-server:2379
}

stop_svr() {
    CONTAINER=grpc_id_svr_1
    docker stop ${CONTAINER}
    CONTAINER=grpc_id_svr_2
    docker stop ${CONTAINER}
    CONTAINER=grpc_id_svr_3
    docker stop ${CONTAINER}
}

start_gw() {
    IMAGE=grpc-id/gw:0.1
    CONTAINER=grpc_id_gw
    docker run --rm -tid -p 60001:60001 \
        --network app-tier \
        --link etcd-server \
        --name ${CONTAINER} ${IMAGE} \
        -port 60001 \
        -reg http://etcd-server:2379
}

stop_gw() {
    CONTAINER=grpc_id_gw
    docker stop ${CONTAINER}
}

start_cli() {
    IMAGE=grpc-id/cli:0.1
    CONTAINER=grpc_id_cli
    docker run --rm -tid -p 60002:60002 \
        --network app-tier \
        --link etcd-server \
        --name ${CONTAINER} ${IMAGE} \
        -reg http://etcd-server:2379
}

stop_cli() {
    CONTAINER=grpc_id_cli
    docker stop ${CONTAINER}
}

logs_svr() {
    CONTAINER=grpc_id_svr_1
    docker logs ${CONTAINER}
}

grpc_svr() {
    go run cmd/svr/svr.go cmd/svr/snowflake.go cmd/svr/sonyflake.go -host 0.0.0.0 -port 50001 -reg http://localhost:2379
    go run cmd/svr/svr.go cmd/svr/snowflake.go cmd/svr/sonyflake.go -host 0.0.0.0 -port 50002 -reg http://localhost:2379
    go run cmd/svr/svr.go cmd/svr/snowflake.go cmd/svr/sonyflake.go -host 0.0.0.0 -port 50003 -reg http://localhost:2379
}

cli_test() {
    go run cmd/cli/cli.go
}

gw_svr() {
    go run cmd/gw/gw.go -host 0.0.0.0 -port 60001 -reg http://localhost:2379
}

gw_test() {
    curl -X POST http://localhost:60001/snowflake -d '{"name": "fromGW"}'
    echo "\r"
    curl -X POST http://localhost:60001/sonyflake -d '{"name": "fromGW"}'
}

case $1 in
    build)
        build;;
    start)
        start;;
    stop)
        stop;;
    logs)
        logs;;
    start_svr)
        start_svr;;
    stop_svr)
        stop_svr;;
    start_gw)
        start_gw;;
    stop_gw)
        stop_gw;;
    start_cli)
        start_cli;;
    stop_cli)
        stop_cli;;
    svr_ip)
        svr_ip;;
    cli_test)
        cli_test;;
    gw_test)
        gw_test;;
    build_compose)
        build_compose;;
    compose)
        docker-compose up -d
        docker-compose images
        docker-compose ps
        ;;
    *)
        echo "./run.sh build|start|stop"
esac
