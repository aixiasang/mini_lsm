package inner

import (
	"testing"

	"github.com/aixiasang/lsm/inner/config"
	"github.com/aixiasang/lsm/inner/utils"
)

func TestLsmTree_Put(t *testing.T) {
	conf := config.DefaultConfig()
	conf.DataDir = "./data"
	conf.WalDir = "./wal"
	conf.SSTDir = "./sst"
	conf.MemTableType = config.MemTableTypeBTree
	conf.MemTableDegree = 16
	tree, err := NewLsmTree(conf)
	if err != nil {
		t.Fatal(err)
	}
	m := make(map[string]string)
	for i := 0; i < 100; i++ {
		key := utils.GetKey(i)
		value := utils.GetValue(10)
		tree.Put(key, value)
		m[string(key)] = string(value)
	}
	for i := 0; i < 100; i++ {
		value, err := tree.Get(utils.GetKey(i))
		if err != nil {
			t.Fatal(err)
		}
		if string(value) != m[string(utils.GetKey(i))] {
			t.Fatalf("value mismatch: %s != %s", string(value), m[string(utils.GetKey(i))])
		}
	}
	tree.Close()
}
func TestLsmTree_Get(t *testing.T) {
	conf := config.DefaultConfig()
	conf.DataDir = "./data"
	conf.WalDir = "./wal"
	conf.SSTDir = "./sst"
	conf.MemTableType = config.MemTableTypeBTree
	conf.MemTableDegree = 16
	tree, err := NewLsmTree(conf)
	if err != nil {
		t.Fatal(err)
	}
	// m := make(map[string]string)
	// for i := 0; i < 100; i++ {
	// 	key := utils.GetKey(i)
	// 	value := utils.GetValue(10)
	// 	tree.Put(key, value)
	// 	m[string(key)] = string(value)
	// }
	for i := 0; i < 100; i++ {
		value, err := tree.Get(utils.GetKey(i))
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(utils.GetKey(i)), string(value))
		// if string(value) != m[string(utils.GetKey(i))] {
		// t.Fatalf("value mismatch: %s != %s", string(value), m[string(utils.GetKey(i))])
		// }
	}
	tree.Close()
}
