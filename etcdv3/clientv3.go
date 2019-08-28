package etcdv3

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
)

// EtcdClient etcd client
type EtcdClient struct {
	*clientv3.Client
	isTimeout bool
	timeout   time.Duration
}

func NewEtcdClient(endpoints []string, timeout time.Duration) (*EtcdClient, error) {
	return NewEtcdClientWithConfig(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: timeout,
	})
}

func NewEtcdClientWithConfig(cfg clientv3.Config) (*EtcdClient, error) {
	cli, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}

	return &EtcdClient{Client: cli}, nil
}

func NewEtcdClientWithTimeout(endpoints []string, dialTimeout, timeout time.Duration) (*EtcdClient, error) {
	cli, err := NewEtcdClientWithConfig(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return nil, err
	}

	cli.timeout = timeout
	cli.isTimeout = timeout.Seconds() > 0
	return cli, nil
}

func (p *EtcdClient) Close() error {
	return p.Client.Close()
}

func (p *EtcdClient) ctx() (ctx context.Context, cancel context.CancelFunc) {
	if p.isTimeout {
		ctx, cancel = context.WithTimeout(context.Background(), p.timeout)
		return ctx, cancel
	}
	ctx, cancel = context.WithCancel(context.Background())
	return ctx, cancel
}

func (p *EtcdClient) Get(key string) (val string, ok bool) {
	kv := clientv3.NewKV(p.Client)

	ctx, cancel := p.ctx()
	resp, err := kv.Get(ctx, key)
	cancel()
	if err != nil {
		return "", false
	}

	if len(resp.Kvs) > 0 {
		return string(resp.Kvs[0].Value), true
	}

	return "", false
}

func (p *EtcdClient) Set(key, val string) error {
	kv := clientv3.NewKV(p.Client)

	ctx, cancel := p.ctx()
	_, err := kv.Put(ctx, key, val)
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func (p *EtcdClient) SetWithSTM(key, val string) error {
	existing := func(stm concurrency.STM) error {
		if val := stm.Get(key); val != "" {
			return fmt.Errorf("%s=%s exist", key, val)
		}
		stm.Put(key, val)
		return nil
	}

	if _, serr := concurrency.NewSTM(p.Client, existing); serr != nil {
		return serr
	}

	return nil
}

func (p *EtcdClient) SetWithSTMs(m map[string]string) error {
	existing := func(stm concurrency.STM) error {
		for key, val := range m {
			if val := stm.Get(key); val != "" {
				return fmt.Errorf("%s=%s exist", key, val)
			}
			stm.Put(key, val)
		}
		return nil
	}

	if _, serr := concurrency.NewSTM(p.Client, existing); serr != nil {
		return serr
	}

	return nil
}

func (p *EtcdClient) GetValues(keys ...string) (map[string]string, error) {
	kvc := clientv3.NewKV(p.Client)

	var opts []clientv3.Op
	for _, k := range keys {
		opts = append(opts, clientv3.OpGet(k))
	}

	ctx, cancel := p.ctx()
	resp, err := kvc.Txn(ctx).Then(opts...).Commit()
	cancel()
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)
	for _, respOp := range resp.Responses {
		if respRange := respOp.GetResponseRange(); respRange != nil {
			for _, kv := range respRange.Kvs {
				m[string(kv.Key)] = string(kv.Value)
			}
		}
	}

	return m, nil
}

func (p *EtcdClient) GetValuesByRange(keyPrefix, endKey string) (map[string]string, error) {
	m := make(map[string]string)
	kv := clientv3.NewKV(p.Client)

	ctx, cancel := p.ctx()
	resp, err := kv.Get(ctx, keyPrefix, clientv3.WithRange(endKey))
	cancel()
	if err != nil {
		return nil, err
	}

	for _, v := range resp.Kvs {
		m[string(v.Key)] = string(v.Value)
	}

	return m, nil
}

func (p *EtcdClient) GetValuesByPrefix(keyPrefix string) (map[string]string, error) {
	m := make(map[string]string)
	kv := clientv3.NewKV(p.Client)

	ctx, cancel := p.ctx()
	resp, err := kv.Get(ctx, keyPrefix, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}

	for _, v := range resp.Kvs {
		m[string(v.Key)] = string(v.Value)
	}

	return m, nil
}

func (p *EtcdClient) SetValues(m map[string]string) error {
	kvc := clientv3.NewKV(p.Client)

	var opts []clientv3.Op
	for k, v := range m {
		opts = append(opts, clientv3.OpPut(k, v))
	}

	ctx, cancel := p.ctx()
	_, err := kvc.Txn(ctx).Then(opts...).Commit()
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func (p *EtcdClient) DelValues(keys ...string) error {
	kvc := clientv3.NewKV(p.Client)

	var opts []clientv3.Op
	for _, k := range keys {
		opts = append(opts, clientv3.OpDelete(k))
	}

	ctx, cancel := p.ctx()
	_, err := kvc.Txn(ctx).Then(opts...).Commit()
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func (p *EtcdClient) DelValuesWithPrefix(keyPrefixs ...string) error {
	kvc := clientv3.NewKV(p.Client)

	var opts []clientv3.Op
	for _, k := range keyPrefixs {
		opts = append(opts, clientv3.OpDelete(k, clientv3.WithPrefix()))
	}

	ctx, cancel := p.ctx()
	_, err := kvc.Txn(ctx).Then(opts...).Commit()
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func (p *EtcdClient) Delete(key string) error {
	kvc := clientv3.NewKV(p.Client)

	ctx, cancel := p.ctx()
	_, err := kvc.Delete(ctx, key)
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func (p *EtcdClient) GetAllValues() (map[string]string, error) {
	kvc := clientv3.NewKV(p.Client)

	ctx, cancel := p.ctx()
	resp, err := kvc.Get(ctx, "", clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)
	for _, v := range resp.Kvs {
		m[string(v.Key)] = string(v.Value)
	}

	return m, nil
}

func (p *EtcdClient) Clear() error {
	kvc := clientv3.NewKV(p.Client)

	ctx, cancel := p.ctx()
	_, err := kvc.Delete(ctx, "", clientv3.WithPrefix())
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func (p *EtcdClient) LeaseGrant(ttl int64) (clientv3.LeaseID, error) {
	cli := p.Client

	ctx, cancel := p.ctx()
	resp, err := cli.Grant(ctx, ttl)
	cancel()
	if err != nil {
		return clientv3.NoLease, err
	}

	return resp.ID, nil
}

func (p *EtcdClient) LeasePut(key, val string, leaseID clientv3.LeaseID) error {
	cli := p.Client

	ctx, cancel := p.ctx()
	_, err := cli.Put(ctx, key, val, clientv3.WithLease(leaseID))
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func (p *EtcdClient) LeasePutWithSTM(key, val string, leaseID clientv3.LeaseID) error {
	existing := func(stm concurrency.STM) error {
		if val := stm.Get(key); val != "" {
			return fmt.Errorf("%s=%s exist", key, val)
		}
		stm.Put(key, val, clientv3.WithLease(leaseID))
		return nil
	}

	if _, serr := concurrency.NewSTM(p.Client, existing); serr != nil {
		return serr
	}

	return nil
}

func (p *EtcdClient) LeasePutWithSTMs(m map[string]string, leaseID clientv3.LeaseID) error {
	existing := func(stm concurrency.STM) error {
		for key, val := range m {
			if val := stm.Get(key); val != "" {
				return fmt.Errorf("%s=%s exist", key, val)
			}
			stm.Put(key, val, clientv3.WithLease(leaseID))
		}
		return nil
	}

	if _, serr := concurrency.NewSTM(p.Client, existing); serr != nil {
		return serr
	}

	return nil
}

func (p *EtcdClient) LeaseRevoke(leaseID clientv3.LeaseID) error {
	cli := p.Client

	ctx, cancel := p.ctx()
	_, err := cli.Revoke(ctx, leaseID)
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func (p *EtcdClient) LeaseKeepAlive(leaseID clientv3.LeaseID) (int64, error) {
	cli := p.Client

	ch, err := cli.KeepAlive(context.TODO(), leaseID)
	if err != nil {
		return 0, err
	}
	ka := <-ch

	return ka.TTL, nil
}

func (p *EtcdClient) LeaseKeepAliveOnce(leaseID clientv3.LeaseID) (int64, error) {
	cli := p.Client

	ka, err := cli.KeepAliveOnce(context.TODO(), leaseID)
	if err != nil {
		return 0, err
	}

	return ka.TTL, nil
}

func (p *EtcdClient) Watch(keyPrefix string) clientv3.WatchChan {
	cli := p.Client

	rch := cli.Watch(context.TODO(), keyPrefix)

	return rch
}

func (p *EtcdClient) WatchWithPrefix(keyPrefix string) clientv3.WatchChan {
	cli := p.Client

	rch := cli.Watch(context.TODO(), keyPrefix, clientv3.WithPrefix())

	return rch
}

func (p *EtcdClient) WatchWithPrefixPrevKV(keyPrefix string) clientv3.WatchChan {
	cli := p.Client

	rch := cli.Watch(context.TODO(), keyPrefix, clientv3.WithPrefix(), clientv3.WithPrevKV())

	return rch
}

func (p *EtcdClient) WatchWithRange(keyPrefix, endKey string) clientv3.WatchChan {
	cli := p.Client

	rch := cli.Watch(context.TODO(), keyPrefix, clientv3.WithRange(endKey))

	return rch
}
