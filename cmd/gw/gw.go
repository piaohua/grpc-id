package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/piaohua/grpc-id/cmd/pb"
	"github.com/piaohua/grpc-id/etcdv3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/resolver"
)

var (
	svc  = flag.String("service", "grpc_id_service", "service name")
	host = flag.String("host", "localhost", "listening host")
	port = flag.String("port", "60001", "listening port")
	reg  = flag.String("reg", "http://localhost:2379", "register etcd address")
)

func main() {
	flag.Parse()
	r := etcdv3.NewResolver(*reg, *svc)
	resolver.Register(r)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// https://github.com/grpc/grpc/blob/master/doc/naming.md
	// The gRPC client library will use the specified scheme to pick the right resolver plugin and pass it the fully qualified name string.
	conn, err := grpc.DialContext(ctx, r.Scheme()+"://authority/"+*svc, grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name), grpc.WithBlock())
	cancel()
	if err != nil {
		log.Panic(err)
	}

	mux := runtime.NewServeMux()
	err = pb.RegisterIDHandler(ctx, mux, conn)
	if err != nil {
		log.Panic(err)
	}

	log.Infof("starting gw service at %s", *port)
	// Start HTTP server (and proxy calls to gRPC server endpoint)
	log.Fatal(http.ListenAndServe(net.JoinHostPort(*host, *port), mux))
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}
