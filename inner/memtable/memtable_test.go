package memtable

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

// 通用测试函数，用于测试 MemTable 接口的基本功能
func testMemTableBasicOperations(t *testing.T, mt MemTable, name string) {
	// 测试 Put 和 Get
	t.Run(fmt.Sprintf("%s-PutAndGet", name), func(t *testing.T) {
		// 插入一些数据
		key1 := []byte("key1")
		value1 := []byte("value1")
		err := mt.Put(key1, value1)
		if err != nil {
			t.Fatalf("Failed to put: %v", err)
		}

		// 获取并验证数据
		result, err := mt.Get(key1)
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if !bytes.Equal(result, value1) {
			t.Errorf("Got unexpected value: got %s, want %s", result, value1)
		}

		// 测试获取不存在的键
		_, err = mt.Get([]byte("non-existent"))
		if err == nil {
			t.Error("Expected error when getting non-existent key, got nil")
		}
	})

	// 测试 Delete
	t.Run(fmt.Sprintf("%s-Delete", name), func(t *testing.T) {
		key := []byte("key-to-delete")
		value := []byte("value-to-delete")

		// 先插入数据
		err := mt.Put(key, value)
		if err != nil {
			t.Fatalf("Failed to put: %v", err)
		}

		// 删除数据
		err = mt.Delete(key)
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		// 验证数据已被删除
		_, err = mt.Get(key)
		if err == nil {
			t.Error("Expected error after deletion, got nil")
		}

		// 测试删除不存在的键
		err = mt.Delete([]byte("non-existent"))
		if err == nil {
			t.Error("Expected error when deleting non-existent key, got nil")
		}
	})

	// 测试 ForEach
	t.Run(fmt.Sprintf("%s-ForEach", name), func(t *testing.T) {
		// 清空内存表 (通过简单地创建一个新的实例)
		var newMt MemTable
		switch name {
		case "BTree":
			newMt = NewBTreeMemTable(2)
		case "SkipList":
			newMt = NewSkipListMemTable()
		}
		mt = newMt

		// 插入有序数据
		keys := [][]byte{
			[]byte("key1"),
			[]byte("key2"),
			[]byte("key3"),
		}
		values := [][]byte{
			[]byte("value1"),
			[]byte("value2"),
			[]byte("value3"),
		}

		for i := range keys {
			err := mt.Put(keys[i], values[i])
			if err != nil {
				t.Fatalf("Failed to put: %v", err)
			}
		}

		// 使用 ForEach 遍历并收集结果
		var resultKeys [][]byte
		var resultValues [][]byte

		mt.ForEach(func(key, value []byte) bool {
			// 注意：这里应该是副本，所以我们需要创建新的切片来存储它们
			keyClone := append([]byte{}, key...)
			valueClone := append([]byte{}, value...)
			resultKeys = append(resultKeys, keyClone)
			resultValues = append(resultValues, valueClone)
			return true
		})

		// 验证收集的结果数量
		if len(resultKeys) != len(keys) {
			t.Errorf("ForEach didn't iterate over all elements: got %d, want %d", len(resultKeys), len(keys))
		}

		// 验证结果顺序是否正确
		for i := range resultKeys {
			if !bytes.Equal(resultKeys[i], keys[i]) {
				t.Errorf("ForEach key order mismatch at index %d: got %s, want %s", i, resultKeys[i], keys[i])
			}
			if !bytes.Equal(resultValues[i], values[i]) {
				t.Errorf("ForEach value order mismatch at index %d: got %s, want %s", i, resultValues[i], values[i])
			}
		}
	})

	// 测试 ForEachUnSafe
	t.Run(fmt.Sprintf("%s-ForEachUnSafe", name), func(t *testing.T) {
		// 使用 ForEachUnSafe 遍历并收集结果
		var resultKeys [][]byte
		var resultValues [][]byte

		mt.ForEachUnSafe(func(key, value []byte) bool {
			// 由于这是不安全的遍历，我们需要自己创建副本
			keyClone := append([]byte{}, key...)
			valueClone := append([]byte{}, value...)
			resultKeys = append(resultKeys, keyClone)
			resultValues = append(resultValues, valueClone)
			return true
		})

		// 验证结果与上一个测试相同
		keys := [][]byte{
			[]byte("key1"),
			[]byte("key2"),
			[]byte("key3"),
		}
		values := [][]byte{
			[]byte("value1"),
			[]byte("value2"),
			[]byte("value3"),
		}

		// 验证收集的结果数量
		if len(resultKeys) != len(keys) {
			t.Errorf("ForEachUnSafe didn't iterate over all elements: got %d, want %d", len(resultKeys), len(keys))
		}

		// 验证结果顺序是否正确
		for i := range resultKeys {
			if !bytes.Equal(resultKeys[i], keys[i]) {
				t.Errorf("ForEachUnSafe key order mismatch at index %d: got %s, want %s", i, resultKeys[i], keys[i])
			}
			if !bytes.Equal(resultValues[i], values[i]) {
				t.Errorf("ForEachUnSafe value order mismatch at index %d: got %s, want %s", i, resultValues[i], values[i])
			}
		}
	})
}

// 测试并发操作
func testMemTableConcurrentOperations(t *testing.T, mt MemTable, name string) {
	t.Run(fmt.Sprintf("%s-Concurrent", name), func(t *testing.T) {
		const numGoroutines = 10
		const numOpsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// 并发写入
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					key := []byte(fmt.Sprintf("key-%d-%d", id, j))
					value := []byte(fmt.Sprintf("value-%d-%d", id, j))
					err := mt.Put(key, value)
					if err != nil {
						t.Errorf("Concurrent Put failed: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()

		// 验证所有键都存在
		for i := 0; i < numGoroutines; i++ {
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := []byte(fmt.Sprintf("key-%d-%d", i, j))
				expectedValue := []byte(fmt.Sprintf("value-%d-%d", i, j))

				value, err := mt.Get(key)
				if err != nil {
					t.Errorf("Failed to get concurrent key: %v", err)
					continue
				}

				if !bytes.Equal(value, expectedValue) {
					t.Errorf("Unexpected value for concurrent key: got %s, want %s", value, expectedValue)
				}
			}
		}
	})
}

// 测试工厂函数
func TestNewMemTable(t *testing.T) {
	t.Run("BTreeFactory", func(t *testing.T) {
		mt := NewMemTable(MemTableTypeBTree, 4)
		if _, ok := mt.(*BTreeMemTable); !ok {
			t.Error("NewMemTable did not return a BTreeMemTable")
		}
	})

	t.Run("SkipListFactory", func(t *testing.T) {
		mt := NewMemTable(MemTableTypeSkipList, 0)
		if _, ok := mt.(*SkipListMemTable); !ok {
			t.Error("NewMemTable did not return a SkipListMemTable")
		}
	})

	t.Run("DefaultDegreeFactory", func(t *testing.T) {
		mt := NewMemTableWithDefaultDegree(MemTableTypeBTree)
		if _, ok := mt.(*BTreeMemTable); !ok {
			t.Error("NewMemTableWithDefaultDegree did not return a BTreeMemTable")
		}
	})

	t.Run("InvalidType", func(t *testing.T) {
		mt := NewMemTable(99, 0) // 使用无效的类型
		if mt != nil {
			t.Error("Expected nil for invalid memtable type")
		}
	})
}

// 测试 B 树实现
func TestBTreeMemTable(t *testing.T) {
	mt := NewBTreeMemTable(2)
	testMemTableBasicOperations(t, mt, "BTree")
	testMemTableConcurrentOperations(t, mt, "BTree")
}

// 测试跳表实现
func TestSkipListMemTable(t *testing.T) {
	mt := NewSkipListMemTable()
	testMemTableBasicOperations(t, mt, "SkipList")
	testMemTableConcurrentOperations(t, mt, "SkipList")
}
