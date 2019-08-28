#!/bin/bash

set -e

GOGOPROTO_ROOT="${GOPATH}/src/github.com/gogo/protobuf"
GRPC_GATEWAY_ROOT="${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway"

${GOPATH}/bin/protoc -I".:${GOGOPROTO_ROOT}/protobuf:${GRPC_GATEWAY_ROOT}/third_party/googleapis" ./*.proto --go_out=plugins=grpc:.
${GOPATH}/bin/protoc -I".:${GOGOPROTO_ROOT}/protobuf:${GRPC_GATEWAY_ROOT}/third_party/googleapis" ./*.proto --swagger_out=logtostderr=true:. --grpc-gateway_out=logtostderr=true:.
