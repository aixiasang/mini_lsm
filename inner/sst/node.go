package sst

import (
	"bytes"

	"github.com/aixiasang/lsm/inner/config"
	"github.com/aixiasang/lsm/inner/filter"
	"github.com/aixiasang/lsm/inner/myerror"
)

type Node struct {
	conf     *config.Config          // 配置
	filename string                  // 文件名
	level    int                     // 层级
	seq      int32                   // 序列号
	size     int64                   // 大小
	minKey   []byte                  // 最小键
	maxKey   []byte                  // 最大键
	index    []*Index                // 索引
	filter   map[int64]filter.Filter // 过滤器
	reader   *SSTReader              // 读取器
	kvList   []*KeyValue             // 数据块
}
type KeyValue struct {
	Key   []byte
	Value []byte
}

func NewNode(conf *config.Config, filename string, level int, seq int32, reader *SSTReader) (*Node, error) {
	size := reader.FileSize()
	minKey := reader.MinKey()
	maxKey := reader.MaxKey()
	index := reader.Index()
	bloomFilter := reader.Filter()
	kvList := reader.KvList()
	return &Node{
		conf:     conf,
		filename: filename,
		level:    level,
		seq:      seq,
		size:     size,
		minKey:   minKey,
		maxKey:   maxKey,
		index:    index,
		filter:   bloomFilter,
		reader:   reader,
		kvList:   kvList,
	}, nil
}

func (n *Node) Get(key []byte) ([]byte, error) {
	// 查看bloomFilter中是否存在key

	// 查看kvlist中是否存在key
	for _, kv := range n.kvList {
		if bytes.Equal(kv.Key, key) {
			if kv.Value == nil {
				return nil, myerror.ErrValueNil
			}
			return kv.Value, nil
		}
	}
	return nil, myerror.ErrKeyNotFound
}
func (n *Node) GetFilename() string {
	return n.filename
}
func (n *Node) GetLevel() int {
	return n.level
}
func (n *Node) GetSeq() int32 {
	return n.seq
}
func (n *Node) GetSize() int64 {
	return n.size
}
func (n *Node) GetMinKey() []byte {
	return n.minKey
}
func (n *Node) GetMaxKey() []byte {
	return n.maxKey
}
func (n *Node) GetIndex() []*Index {
	return n.index
}
