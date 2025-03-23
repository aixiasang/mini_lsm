package memtable

import (
	"bytes"
	"sync"

	"github.com/aixiasang/lsm/inner/myerror"
	"github.com/huandu/skiplist"
)

// BytesCompare 用于比较两个字节切片的函数
// 返回值小于0表示k1 < k2，等于0表示k1 == k2，大于0表示k1 > k2
func BytesCompare(k1, k2 interface{}) int {
	b1 := k1.([]byte)
	b2 := k2.([]byte)
	return bytes.Compare(b1, b2)
}

// SkipListMemTable 跳表内存表实现
type SkipListMemTable struct {
	list  *skiplist.SkipList
	mutex sync.RWMutex // 读写锁，用于并发控制
}

// NewSkipListMemTable 创建一个新的跳表内存表
func NewSkipListMemTable() *SkipListMemTable {
	// 使用 GreaterThanFunc 创建支持 []byte 作为键的跳表
	return &SkipListMemTable{
		list: skiplist.New(skiplist.GreaterThanFunc(BytesCompare)),
	}
}

// Put 向跳表中插入键值对
func (sl *SkipListMemTable) Put(key, value []byte) error {
	if key == nil {
		return myerror.ErrKeyNil
	}

	// 深拷贝键和值，避免外部修改
	keyCopy := append([]byte{}, key...)
	valueCopy := append([]byte{}, value...)

	sl.mutex.Lock()         // 写操作加锁
	defer sl.mutex.Unlock() // 确保操作完成后解锁

	sl.list.Set(keyCopy, valueCopy)
	return nil
}

// Get 从跳表中获取值
func (sl *SkipListMemTable) Get(key []byte) ([]byte, error) {
	if key == nil {
		return nil, myerror.ErrKeyNil
	}

	sl.mutex.RLock()         // 读操作加读锁
	defer sl.mutex.RUnlock() // 确保操作完成后解锁

	element := sl.list.Get(key)
	if element == nil {
		return nil, myerror.ErrKeyNotFound
	}

	// 返回值的深拷贝，避免外部修改
	value := element.Value.([]byte)
	return append([]byte{}, value...), nil
}

// Delete 从跳表中删除一个键值对
func (sl *SkipListMemTable) Delete(key []byte) error {
	if key == nil {
		return myerror.ErrKeyNil
	}

	sl.mutex.Lock()         // 写操作加锁
	defer sl.mutex.Unlock() // 确保操作完成后解锁

	if sl.list.Remove(key) == nil {
		return myerror.ErrKeyNotFound
	}

	return nil
}

// ForEach 遍历跳表中的所有键值对
func (sl *SkipListMemTable) ForEach(visitor func(key, value []byte) bool) {
	sl.mutex.RLock()         // 读操作加读锁
	defer sl.mutex.RUnlock() // 确保操作完成后解锁

	// 从第一个元素开始遍历
	for element := sl.list.Front(); element != nil; element = element.Next() {
		// 创建键值的深拷贝
		key := element.Key().([]byte)
		value := element.Value.([]byte)

		keyCopy := append([]byte{}, key...)
		valueCopy := append([]byte{}, value...)

		if !visitor(keyCopy, valueCopy) {
			break
		}
	}
}

// ForEachUnSafe 非安全地遍历跳表中的所有键值对
// 直接传递内部引用，不创建拷贝，性能更高
// 注意：调用方负责处理锁定，确保在调用该方法前已获取适当的锁
func (sl *SkipListMemTable) ForEachUnSafe(visitor func(key, value []byte) bool) {
	for element := sl.list.Front(); element != nil; element = element.Next() {
		key := element.Key().([]byte)
		value := element.Value.([]byte)

		if !visitor(key, value) {
			break
		}
	}
}
