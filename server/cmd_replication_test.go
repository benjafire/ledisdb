package server

import (
	"fmt"
	"github.com/siddontang/ledisdb/config"
	"os"
	"reflect"
	"testing"
	"time"
)

func checkDataEqual(master *App, slave *App) error {
	mdb, _ := master.ldb.Select(0)
	sdb, _ := slave.ldb.Select(0)

	mkeys, _ := mdb.Scan(nil, 100, true, "")
	skeys, _ := sdb.Scan(nil, 100, true, "")

	if len(mkeys) != len(skeys) {
		return fmt.Errorf("keys number not equal")
	} else if !reflect.DeepEqual(mkeys, skeys) {
		return fmt.Errorf("keys not equal")
	} else {
		for _, k := range mkeys {
			v1, _ := mdb.Get(k)
			v2, _ := sdb.Get(k)
			if !reflect.DeepEqual(v1, v2) {
				return fmt.Errorf("value not equal")
			}
		}
	}

	return nil
}

func TestReplication(t *testing.T) {
	data_dir := "/tmp/test_replication"
	os.RemoveAll(data_dir)

	masterCfg := new(config.Config)
	masterCfg.DataDir = fmt.Sprintf("%s/master", data_dir)
	masterCfg.Addr = "127.0.0.1:11182"
	masterCfg.BinLog.MaxFileSize = 1 * 1024 * 1024
	masterCfg.BinLog.MaxFileNum = 10

	var master *App
	var slave *App
	var err error
	master, err = NewApp(masterCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()

	slaveCfg := new(config.Config)
	slaveCfg.DataDir = fmt.Sprintf("%s/slave", data_dir)
	slaveCfg.Addr = "127.0.0.1:11183"
	slaveCfg.SlaveOf = masterCfg.Addr

	slave, err = NewApp(slaveCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()

	go master.Run()

	db, _ := master.ldb.Select(0)

	value := make([]byte, 10)

	db.Set([]byte("a"), value)
	db.Set([]byte("b"), value)
	db.Set([]byte("c"), value)
	db.Set([]byte("d"), value)

	go slave.Run()

	time.Sleep(1 * time.Second)

	if err = checkDataEqual(master, slave); err != nil {
		t.Fatal(err)
	}

	db.Set([]byte("a1"), value)
	db.Set([]byte("b1"), value)
	db.Set([]byte("c1"), value)
	db.Set([]byte("d1"), value)

	time.Sleep(1 * time.Second)
	if err = checkDataEqual(master, slave); err != nil {
		t.Fatal(err)
	}

	slave.slaveof("")

	db.Set([]byte("a2"), value)
	db.Set([]byte("b2"), value)
	db.Set([]byte("c2"), value)
	db.Set([]byte("d2"), value)

	db.Set([]byte("a3"), value)
	db.Set([]byte("b3"), value)
	db.Set([]byte("c3"), value)
	db.Set([]byte("d3"), value)

	if err = checkDataEqual(master, slave); err == nil {
		t.Fatal("must error")
	}

	slave.slaveof(masterCfg.Addr)
	time.Sleep(1 * time.Second)

	if err = checkDataEqual(master, slave); err != nil {
		t.Fatal(err)
	}

}
