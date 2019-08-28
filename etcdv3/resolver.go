package etcdv3

import (
	"fmt"
	"strings"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"google.golang.org/grpc/resolver"
)

const schema = "etcdv3_resolver"

// Resolver is the implementaion of grpc.resolve.Builder
type Resolver struct {
	target  string
	service string
	cli     *EtcdClient
	cc      resolver.ClientConn
}

// NewResolver return resolver builder
// target example: "http://127.0.0.1:2379,http://127.0.0.1:12379,http://127.0.0.1:22379"
// service is service name
func NewResolver(target string, service string) resolver.Builder {
	return &Resolver{target: target, service: service}
}

// Scheme return etcdv3 schema
func (r *Resolver) Scheme() string {
	return schema
}

// ResolveNow ...
func (r *Resolver) ResolveNow(rn resolver.ResolveNowOption) {
}

// Close ...
func (r *Resolver) Close() {
}

// Build to resolver.Resolver
func (r *Resolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	var err error

	dialTimeout := time.Duration(5) * time.Second
	timeout := time.Duration(3) * time.Second
	endPoints := strings.Split(r.target, ",")
	r.cli, err = NewEtcdClientWithTimeout(endPoints, dialTimeout, timeout)
	if err != nil {
		return nil, fmt.Errorf("grpclb: create clientv3 client failed: %v", err)
	}

	r.cc = cc

	go r.watch(fmt.Sprintf("/%s/%s/", schema, r.service))

	return r, nil
}

func (r *Resolver) watch(prefix string) {
	addrDict := make(map[string]resolver.Address)

	update := func() {
		addrList := make([]resolver.Address, 0, len(addrDict))
		for _, v := range addrDict {
			addrList = append(addrList, v)
		}
		r.cc.UpdateState(resolver.State{
			Addresses: addrList,
		})
	}

	resp, err := r.cli.GetValuesByPrefix(prefix)
	if err == nil {
		for _, v := range resp {
			addrDict[v] = resolver.Address{Addr: v}
		}
	}

	update()

	rch := r.cli.WatchWithPrefixPrevKV(prefix)
	for n := range rch {
		for _, ev := range n.Events {
			switch ev.Type {
			case mvccpb.PUT:
				addrDict[string(ev.Kv.Key)] = resolver.Address{Addr: string(ev.Kv.Value)}
			case mvccpb.DELETE:
				delete(addrDict, string(ev.PrevKv.Key))
			}
		}
		update()
	}
}
