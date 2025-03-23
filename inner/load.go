package inner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/aixiasang/lsm/inner/memtable"
	"github.com/aixiasang/lsm/inner/myerror"
	"github.com/aixiasang/lsm/inner/sst"
	"github.com/aixiasang/lsm/inner/wal"
)

func (t *LsmTree) load() error {
	if err := t.loadSST(); err != nil {
		return err
	}
	if err := t.loadWAL(); err != nil {
		return err
	}
	return nil
}

type sstFile struct {
	level    int
	seq      uint32
	filePath string
}

// 载入sst
func (t *LsmTree) loadSST() error {
	filePath := filepath.Join(t.conf.DataDir, t.conf.SSTDir)
	files, err := os.ReadDir(filePath)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	sstFiles := make([]*sstFile, 0)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sst") {
			return myerror.ErrSSTCorrupted
		}
		fileName := strings.TrimSuffix(file.Name(), ".sst")
		level, seq, err := parseSSTFileName(fileName)
		if err != nil {
			return err
		}
		sstFilePath := filepath.Join(filePath, file.Name())

		sstFiles = append(sstFiles, &sstFile{
			level:    level,
			seq:      seq,
			filePath: sstFilePath,
		})
	}
	sort.Slice(sstFiles, func(i, j int) bool {
		if sstFiles[i].level < sstFiles[j].level {
			return true
		}
		if sstFiles[i].level == sstFiles[j].level {
			return sstFiles[i].seq < sstFiles[j].seq
		}
		return false
	})
	for _, sstFile := range sstFiles {
		sstReader, err := sst.NewSSTReader(t.conf, sstFile.filePath)
		if err != nil {
			return err
		}
		node, err := sst.NewNode(t.conf, sstFile.filePath, sstFile.level, int32(sstFile.seq), sstReader)
		if err != nil {
			return err
		}
		if t.conf.IsDebug {
			fmt.Printf("level: %d, seq: %d, len(t.nodes[level]): %d\n", sstFile.level, sstFile.seq, len(t.nodes[sstFile.level]))
		}
		t.nodes[sstFile.level] = append(t.nodes[sstFile.level], node)
	}
	return nil
}
func parseSSTFileName(fileName string) (int, uint32, error) {

	parts := strings.Split(fileName, "_")
	if len(parts) != 2 {
		return 0, 0, myerror.ErrSSTCorrupted
	}
	level, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	seq, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, 0, err
	}
	return level, uint32(seq), nil
}

// 载入wal
func (t *LsmTree) loadWAL() error {
	// 遍历wal目录
	filePath := filepath.Join(t.conf.DataDir, t.conf.WalDir)
	files, err := os.ReadDir(filePath)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	walIds := make([]uint32, 0)
	for _, file := range files {
		// wal-*.log
		if !strings.HasPrefix(file.Name(), "wal-") || !strings.HasSuffix(file.Name(), ".log") {
			return myerror.ErrWalCorrupted
		}
		walName := strings.TrimSuffix(file.Name(), ".log")
		walName = strings.TrimPrefix(walName, "wal-")
		walId, err := strconv.ParseUint(walName, 10, 32)
		if err != nil {
			return err
		}
		walIds = append(walIds, uint32(walId))
	}
	// 对于wal文件进行排序
	sort.Slice(walIds, func(i, j int) bool {
		return walIds[i] < walIds[j]
	})
	for i, walId := range walIds {
		curWal, err := wal.NewWal(t.conf, walId)
		if err != nil {
			return err
		}
		curIndex := t.conf.MemTableConstructor(memtable.MemTableType(t.conf.MemTableType), t.conf.MemTableDegree)
		if err := curWal.ReadAll(curIndex); err != nil {
			return err
		}
		t.immutableIndex = append(t.immutableIndex, &immutable{
			wal:   curWal,
			index: curIndex,
		})
		if i == len(walIds)-1 {
			t.walId = walId
		}
	}
	return nil
}
