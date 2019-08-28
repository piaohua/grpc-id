package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/piaohua/grpc-id/cmd/pb"
	"github.com/piaohua/grpc-id/etcdv3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	serv = flag.String("service", "grpc_id_service", "service name")
	host = flag.String("host", "localhost", "listening host")
	port = flag.String("port", "50001", "listening port")
	reg  = flag.String("reg", "http://localhost:2379", "register etcd address")
)

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", net.JoinHostPort(*host, *port))
	if err != nil {
		log.Panic(err)
	}

	machineID := sonyInit()
	nodeID := snowInit()

	m := make(map[string]string)
	serviceValue := net.JoinHostPort(*host, *port)
	serviceKey := fmt.Sprintf("/%s/%s/%s", (&etcdv3.Resolver{}).Scheme(), *serv, serviceValue)
	snowKey := fmt.Sprintf("/snowflake/node/%d", nodeID)
	sonyKey := fmt.Sprintf("/sonyflake/machine/%d", machineID)
	m[serviceKey] = serviceValue
	m[snowKey] = fmt.Sprintf("%d", nodeID)
	m[sonyKey] = fmt.Sprintf("%d", machineID)

	err = etcdv3.Register(*reg, 15, m)
	if err != nil {
		log.Panic(err)
	}

	log.Infof("starting id service at %s", *port)
	s := grpc.NewServer()
	pb.RegisterIDServer(s, &server{})
	go s.Serve(lis)

	signals()

	etcdv3.UnRegister()
	log.Infof("stop id service at %s", *port)
}

// server is used to implement IDServer.
type server struct{}

// GetSnowflake implements IDServer
func (s *server) GetSnowflake(ctx context.Context, in *pb.SnowflakeRequest) (*pb.SnowflakeReply, error) {
	log.Infof("Receive is %s", in.Name)
	id, actualTime, node, sequence := snowID()
	// Print out the ID, timestamp, node number, sequence number
	log.Infof("id %d, actualTime %d, node %d, sequence %d",
		id, actualTime, node, sequence)
	return &pb.SnowflakeReply{
		Id:       id,
		Time:     actualTime,
		Node:     node,
		Sequence: sequence,
	}, nil
}

// GetSonyflake implements IDServer
func (s *server) GetSonyflake(ctx context.Context, in *pb.SonyflakeRequest) (*pb.SonyflakeReply, error) {
	log.Infof("Receive is %s", in.Name)
	id, msb, actualTime, sequence, machineID := sonyID()
	log.Infof("id %d, msb %d, actualTime %d, sequence %d, machineID %d",
		id, msb, actualTime, sequence, machineID)
	return &pb.SonyflakeReply{
		Id:       id,
		Time:     actualTime,
		Machine:  machineID,
		Sequence: sequence,
		Msb:      msb,
	}, nil
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

func signals() {
	// Go signal notification works by sending `os.Signal`
	// values on a channel. We'll create a channel to
	// receive these notifications (we'll also make one to
	// notify us when the program can exit).
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	// `signal.Notify` registers the given channel to
	// receive notifications of the specified signals.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// This goroutine executes a blocking receive for
	// signals. When it gets one it'll print it out
	// and then notify the program that it can finish.
	go func() {
		sig := <-sigs
		log.Warn("sig ", sig)
		done <- true
	}()

	// The program will wait here until it gets the
	// expected signal (as indicated by the goroutine
	// above sending a value on `done`) and then exit.
	log.Info("awaiting signal")
	<-done
	log.Warn("exiting")
}
