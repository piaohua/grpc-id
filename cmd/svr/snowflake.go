package main

import (
	snowflake "github.com/piaohua/snowflake-golang"
)

func snowInit() uint64 {
	// Create a new Node with localhost
	snowflake.DefaultNode()
	// etcd register node id
	_, _, node, _ := snowID()
	return node
}

func snowID() (uint64, int64, uint64, uint64) {
	id := snowflake.GenerateID()
	return id.Uint64(), id.Time(), id.Node(), id.Sequence()
}
