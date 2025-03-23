package inner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/aixiasang/lsm/inner/config"
	"github.com/aixiasang/lsm/inner/memtable"
	"github.com/aixiasang/lsm/inner/myerror"
	"github.com/aixiasang/lsm/inner/sst"
	"github.com/aixiasang/lsm/inner/wal"
)

type LsmTree struct {
	conf           *config.Config    // 配置
	mutableIndex   memtable.MemTable // 内存表
	walId          uint32            // 写日志文件id
	curWal         *wal.Wal          // 当前写日志
	immutableIndex []*immutable      // 不可变索引
	compactCh      chan *immutable   // 压缩通道，用于异步传递不可变索引进行压缩
	stopCh         chan struct{}     // 停止信号通道
	nodes          [][]*sst.Node     // 节点 - array of slices of nodes for each level
	seq            []*atomic.Uint32  // 序列号
	levelSize      int               // 层级大小
}

func NewLsmTree(conf *config.Config) (*LsmTree, error) {
	dbDir := conf.DataDir

	if err := os.MkdirAll(filepath.Join(dbDir, conf.WalDir), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dbDir, conf.SSTDir), 0755); err != nil {
		return nil, err
	}

	// Ensure LevelSize is at least 1
	if conf.LevelSize <= 0 {
		conf.LevelSize = 1
	}

	// Initialize levelSize from config
	levelSize := conf.LevelSize

	// Initialize nodes as an array of levelSize slices
	nodes := make([][]*sst.Node, levelSize)
	for i := 0; i < levelSize; i++ {
		nodes[i] = make([]*sst.Node, 0)
	}

	// Initialize sequence numbers for each level
	seq := make([]*atomic.Uint32, levelSize)
	for i := 0; i < levelSize; i++ {
		seq[i] = &atomic.Uint32{}
	}

	tree := &LsmTree{
		conf:           conf,
		mutableIndex:   memtable.NewMemTable(memtable.MemTableTypeBTree, 16),
		immutableIndex: []*immutable{},
		compactCh:      make(chan *immutable, 10), // 缓冲区大小为10
		stopCh:         make(chan struct{}),
		nodes:          nodes,
		seq:            seq,
		levelSize:      levelSize,
	}
	go tree.compactWorker()

	if err := tree.load(); err != nil {
		return nil, err
	}
	// 启动后台goroutine监听compactCh通道，执行压缩操作
	if tree.walId != 0 {
		tree.walId++
	}
	curWal, err := wal.NewWal(conf, tree.walId)
	if err != nil {
		return nil, err
	}
	tree.curWal = curWal
	return tree, nil
}

// compactWorker 持续监听compactCh通道，执行压缩操作
func (t *LsmTree) compactWorker() {
	for {
		select {
		case immutable := <-t.compactCh:
			// 收到不可变索引，执行压缩
			if err := t.doCompact(immutable); err != nil {
				// 错误处理，实际应用中可能需要记录日志
				// 这里简单地将错误打印出来
				// 实际应用可能需要更复杂的错误处理机制
				log.Printf("Compact error: %v", err)
			}
		case <-t.stopCh:
			// 收到停止信号，结束goroutine
			return
		}
	}
}

// Close 关闭LSM树，释放资源
func (t *LsmTree) Close() error {
	// 发送停止信号
	close(t.stopCh)

	// 关闭当前WAL
	if err := t.curWal.Close(); err != nil {
		return err
	}

	// 关闭所有不可变索引的WAL
	for _, imm := range t.immutableIndex {
		if err := imm.wal.Close(); err != nil {
			return err
		}
	}

	return nil
}

type immutable struct {
	wal   *wal.Wal
	index memtable.MemTable
}

func (t *LsmTree) rotateWal() error {
	immutable := &immutable{
		wal:   t.curWal,
		index: t.mutableIndex,
	}

	// 将不可变索引添加到列表
	t.immutableIndex = append(t.immutableIndex, immutable)

	// 将不可变索引发送到压缩通道，触发异步压缩
	select {
	case t.compactCh <- immutable:
		// 发送成功
	default:
		// 通道已满，此处可以选择阻塞等待或记录日志
		// 这里选择继续执行，不阻塞主流程
	}

	t.walId++
	curWal, err := wal.NewWal(t.conf, t.walId)
	if err != nil {
		return err
	}
	t.curWal = curWal
	t.mutableIndex = memtable.NewMemTable(memtable.MemTableTypeBTree, 16)
	return nil
}

func (t *LsmTree) Put(key, value []byte) error {
	if err := t.curWal.Write(key, value); err != nil {
		return err
	}
	if err := t.mutableIndex.Put(key, value); err != nil {
		return err
	}

	if t.curWal.Size() > t.conf.WalSize {
		return t.rotateWal()
	}
	return nil
}

func (t *LsmTree) Get(key []byte) ([]byte, error) {
	value, err := t.mutableIndex.Get(key)
	if err == nil {
		return value, nil
	}
	if err == myerror.ErrKeyNotFound {
		// 从不可变索引中查找
		for i := len(t.immutableIndex) - 1; i >= 0; i-- {
			value, err := t.immutableIndex[i].index.Get(key)
			if err == nil {
				return value, nil
			}
			if err == myerror.ErrKeyNotFound {
				continue
			}
			if value != nil {
				return value, nil
			}
			return nil, myerror.ErrKeyNotFound
		}
		// 从节点中查找
		for level := range t.nodes {
			nodeSlice := t.nodes[level]
			// fmt.Printf("level: %d, len(nodeSlice): %d\n", level, len(nodeSlice))
			for i := len(nodeSlice) - 1; i >= 0; i-- {
				// if nodeSlice[i] == nil {
				// 	continue
				// }
				value, err := nodeSlice[i].Get(key)
				if err == nil {
					if value == nil {
						return value, myerror.ErrValueNil
					}
					return value, nil
				} else if err == myerror.ErrKeyNotFound {
					continue
				} else {
					return nil, err
				}

			}
		}
		// 如果所有节点都找不到，返回ErrKeyNotFound
		return nil, myerror.ErrKeyNotFound
	}

	return nil, err
}

func (t *LsmTree) Delete(key []byte) error {
	if err := t.curWal.Write(key, nil); err != nil {
		return err
	}
	if err := t.mutableIndex.Put(key, nil); err != nil {
		return err
	}

	if t.curWal.Size() > t.conf.WalSize {
		return t.rotateWal()
	}
	return nil
}

// doCompact 对单个不可变索引执行压缩操作
func (t *LsmTree) doCompact(imm *immutable) error {
	// 确保传入的immutable存在于immutableIndex中
	found := false
	var index int
	for i, item := range t.immutableIndex {
		if item == imm {
			found = true
			index = i
			break
		}
	}

	if !found {
		return nil // 该不可变索引已被处理或移除
	}

	// 调用底层compact方法将memtable转为SST文件
	if t.conf.IsDebug {
		fmt.Println("compact")
		fmt.Printf("levelSize: %d, seq length: %d\n", t.levelSize, len(t.seq))
	}

	// Check if t.seq has elements before accessing index 0
	if len(t.seq) == 0 {
		// Try to re-initialize the sequence array if it's empty
		t.seq = make([]*atomic.Uint32, t.levelSize)
		for i := 0; i < t.levelSize; i++ {
			t.seq[i] = &atomic.Uint32{}
		}

		if len(t.seq) == 0 {
			return fmt.Errorf("failed to initialize sequence array, levelSize: %d", t.levelSize)
		}
	}

	seq := t.seq[0].Load()
	t.seq[0].Add(1)
	sstFilePath := t.getSSTFilePath(0, seq)
	if err := t.writeMemTableToSST(imm, sstFilePath); err != nil {
		return err
	}

	// // 压缩完成后，可以关闭WAL并从immutableIndex中移除该索引
	// if err := imm.wal.Close(); err != nil {
	// 	return err
	// }

	// 从immutableIndex中移除该索引
	if err := t.immutableIndex[index].wal.Delete(); err != nil {
		return err
	}
	t.immutableIndex[index] = nil
	// 需要加锁保护，此处简化处理
	t.immutableIndex = append(t.immutableIndex[:index], t.immutableIndex[index+1:]...)
	// 将SST文件添加到节点中

	sstReader, err := sst.NewSSTReader(t.conf, sstFilePath)
	if err != nil {
		return err
	}
	node, err := sst.NewNode(t.conf, sstFilePath, 0, int32(seq), sstReader)
	if err != nil {
		return err
	}
	t.nodes[0] = append(t.nodes[0], node)
	return nil
}

// getSSTFilePath 获取SST文件路径
func (t *LsmTree) getSSTFilePath(level int, seq uint32) string {
	return filepath.Join(t.conf.DataDir, t.conf.SSTDir, fmt.Sprintf("%d_%d.sst", level, seq))
}

// writeMemTableToSST 将memtable内容写入SST文件
func (t *LsmTree) writeMemTableToSST(imm *immutable, sstFilePath string) error {
	//将memtable中的数据写入到新的SST文件中
	sstable, err := sst.NewSSTWriter(t.conf, sstFilePath)
	if err != nil {
		return err
	}

	// 使用ForEachUnSafe遍历索引中的所有键值对
	imm.index.ForEachUnSafe(func(key, value []byte) bool {
		if err := sstable.Add(key, value); err != nil {
			return false
		}
		return true
	})

	if err := sstable.Flush(); err != nil {
		return err
	}

	if err := sstable.Close(); err != nil {
		return err
	}

	return nil
}
