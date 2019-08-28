package etcdv3

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Deregister Prefix should start and end with no slash
var Deregister = make(chan struct{})

// Register register services
func Register(target string, ttl int, m map[string]string) error {
	log.Infof("services: %v", m)

	// get endpoints for register dial address
	var err error
	dialTimeout := time.Duration(5) * time.Second
	timeout := time.Duration(3) * time.Second
	endPoints := strings.Split(target, ",")
	cli, err := NewEtcdClientWithTimeout(endPoints, dialTimeout, timeout)
	if err != nil {
		return fmt.Errorf("grpclb: create clientv3 client failed: %v", err)
	}

	leaseID, err := cli.LeaseGrant(int64(ttl))
	if err != nil {
		return fmt.Errorf("grpclb: create clientv3 lease failed: %v", err)
	}

	if err := cli.LeasePutWithSTMs(m, leaseID); err != nil {
		return fmt.Errorf("grpclb: set services '%v' with ttl to clientv3 failed: %s", m, err.Error())
	}

	if _, err := cli.LeaseKeepAlive(leaseID); err != nil {
		return fmt.Errorf("grpclb: refresh services '%v' with ttl to clientv3 failed: %s", m, err.Error())
	}

	// wait deregister then delete
	go func() {
		<-Deregister
		var keys []string
		for key := range m {
			keys = append(keys, key)
		}
		cli.DelValues(keys...)
		Deregister <- struct{}{}
	}()

	return nil
}

// UnRegister delete registered services from etcd
func UnRegister() {
	Deregister <- struct{}{}
	<-Deregister
}
