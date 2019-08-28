package main

import (
	"context"
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/piaohua/grpc-id/cmd/pb"
	"github.com/piaohua/grpc-id/etcdv3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/resolver"
)

var (
	svc = flag.String("service", "grpc_id_service", "service name")
	reg = flag.String("reg", "http://localhost:2379", "register etcd address")
)

func main() {
	flag.Parse()
	r := etcdv3.NewResolver(*reg, *svc)
	resolver.Register(r)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// https://github.com/grpc/grpc/blob/master/doc/naming.md
	// The gRPC client library will use the specified scheme to pick the right resolver plugin and pass it the fully qualified name string.
	// e.g. target string "unknown_scheme://authority/endpoint"
	conn, err := grpc.DialContext(ctx, r.Scheme()+"://authority/"+*svc, grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name), grpc.WithBlock())
	cancel()
	if err != nil {
		log.Panic(err.Error())
	}

	ticker := time.NewTicker(1 * time.Millisecond)
	for t := range ticker.C {
		client := pb.NewIDClient(conn)
		snow, err := client.GetSnowflake(context.Background(), &pb.SnowflakeRequest{Name: "fromCli " + strconv.Itoa(t.Nanosecond())})
		if err == nil {
			log.Infof("%v: Reply id:%d, time:%d, node:%d, sequence:%d",
				t, snow.Id, snow.Time, snow.Node, snow.Sequence)
		}
		sony, err := client.GetSonyflake(context.Background(), &pb.SonyflakeRequest{Name: "fromCli " + strconv.Itoa(t.Nanosecond())})
		if err == nil {
			log.Infof("%v: Reply id:%d, time:%d, machine:%d, sequence:%d",
				t, sony.Id, sony.Time, sony.Machine, sony.Sequence)
		}
	}
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
