package main

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sony/sonyflake"
)

var sf *sonyflake.Sonyflake

func sonyInit() uint64 {
	var st sonyflake.Settings
	st.StartTime = time.Date(2019, 8, 26, 0, 0, 0, 0, time.UTC)
	sf = sonyflake.NewSonyflake(st)
	if sf == nil {
		log.Panic("sonyflake not created")
	}
	// etcd register machine id
	_, _, _, _, machine := sonyID()
	return machine
}

func sonyID() (id, actualMSB, actualTime, actualSequence, actualMachineID uint64) {
	id, err := sf.NextID()
	if err != nil {
		log.Fatal("id not generated")
	}
	parts := sonyflake.Decompose(id)
	actualMSB = parts["msb"]
	actualTime = parts["time"]
	actualSequence = parts["sequence"]
	actualMachineID = parts["machine-id"]
	return id, actualMSB, actualTime, actualSequence, actualMachineID
}
