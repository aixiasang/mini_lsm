package memtable

// MemTable 内存表接口
type MemTable interface {
	Put(key, value []byte) error                        // 插入
	Get(key []byte) ([]byte, error)                     // 查询
	Delete(key []byte) error                            // 删除
	ForEach(visitor func(key, value []byte) bool)       // 遍历
	ForEachUnSafe(visitor func(key, value []byte) bool) // 遍历
}

type MemTableType int8

const (
	MemTableTypeBTree MemTableType = iota
	MemTableTypeSkipList
)

func NewMemTable(mtType MemTableType, degree int) MemTable {
	switch mtType {
	case MemTableTypeBTree:
		return NewBTreeMemTable(degree)
	case MemTableTypeSkipList:
		return NewSkipListMemTable()
	default:
		return nil
	}
}

func NewMemTableWithDefaultDegree(mtType MemTableType) MemTable {
	return NewMemTable(mtType, 32)
}
