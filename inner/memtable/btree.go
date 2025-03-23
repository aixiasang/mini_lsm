package memtable

import (
	"bytes"
	"sync"

	"github.com/aixiasang/lsm/inner/myerror"
	"github.com/google/btree"
)

// KVItem 用于存储在B树中的键值对
type KVItem struct {
	key   []byte
	value []byte
}

// Less 实现btree.Item接口的Less方法
func (i *KVItem) Less(than btree.Item) bool {
	return bytes.Compare(i.key, than.(*KVItem).key) < 0
}

// BTreeMemTable B树内存表实现
type BTreeMemTable struct {
	tree  *btree.BTree
	mutex sync.RWMutex // 读写锁，用于并发控制
}

// NewBTreeMemTable 创建一个新的B树内存表
func NewBTreeMemTable(degree int) *BTreeMemTable {
	if degree <= 0 {
		degree = 2 // 默认度为2
	}
	return &BTreeMemTable{
		tree: btree.New(degree),
	}
}

// Put 向B树中插入键值对
func (bt *BTreeMemTable) Put(key, value []byte) error {
	if key == nil {
		return myerror.ErrKeyNil
	}

	item := &KVItem{
		key:   append([]byte{}, key...),   // 深拷贝，避免外部修改
		value: append([]byte{}, value...), // 深拷贝，避免外部修改
	}

	bt.mutex.Lock()         // 写操作加锁
	defer bt.mutex.Unlock() // 确保操作完成后解锁

	bt.tree.ReplaceOrInsert(item)
	return nil
}

// Get 从B树中获取值
func (bt *BTreeMemTable) Get(key []byte) ([]byte, error) {
	if key == nil {
		return nil, myerror.ErrKeyNil
	}

	searchItem := &KVItem{key: key}

	bt.mutex.RLock()         // 读操作加读锁
	defer bt.mutex.RUnlock() // 确保操作完成后解锁

	item := bt.tree.Get(searchItem)
	if item == nil {
		return nil, myerror.ErrKeyNotFound
	}

	kvItem := item.(*KVItem)
	return append([]byte{}, kvItem.value...), nil // 返回拷贝，避免外部修改
}

// Delete 从B树中删除一个键值对
func (bt *BTreeMemTable) Delete(key []byte) error {
	if key == nil {
		return myerror.ErrKeyNil
	}

	searchItem := &KVItem{key: key}

	bt.mutex.Lock()         // 写操作加锁
	defer bt.mutex.Unlock() // 确保操作完成后解锁

	item := bt.tree.Delete(searchItem)
	if item == nil {
		return myerror.ErrKeyNotFound
	}

	return nil
}

// ForEach 遍历B树中的所有键值对
func (bt *BTreeMemTable) ForEach(visitor func(key, value []byte) bool) {
	bt.mutex.RLock()         // 读操作加读锁
	defer bt.mutex.RUnlock() // 确保操作完成后解锁

	bt.tree.Ascend(func(i btree.Item) bool {
		kvItem := i.(*KVItem)
		// 传递拷贝，避免外部修改
		keyCopy := append([]byte{}, kvItem.key...)
		valueCopy := append([]byte{}, kvItem.value...)
		return visitor(keyCopy, valueCopy)
	})
}

// ForEachUnSafe 非安全地遍历B树中的所有键值对
// 直接传递内部引用，不创建拷贝，性能更高
// 注意：调用方负责处理锁定，确保在调用该方法前已获取适当的锁
func (bt *BTreeMemTable) ForEachUnSafe(visitor func(key, value []byte) bool) {
	bt.tree.Ascend(func(i btree.Item) bool {
		kvItem := i.(*KVItem)
		return visitor(kvItem.key, kvItem.value)
	})
}
