# 说明

- gRPC服务发现&负载均衡
- 参考[grpc-lb](https://github.com/wwcd/grpc-lb)

## 编译

```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go mod vendor build -a -installsuffix cgo -o cmd/cli/cli ./cmd/cli
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go mod vendor build -a -installsuffix cgo -o cmd/gw/gw ./cmd/gw
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go mod vendor build -a -installsuffix cgo -o cmd/svr/svr ./cmd/svr

or

sh ./run.sh build

or

sh ./run.sh build_compose

```

## 启动ETCD

```
sh ./run.sh start_etcd
```

## 测试

```
sh ./run.sh start_svr
sh ./run.sh start_gw
sh ./run.sh start_cli

or

sh ./run.sh compose

```
