package filter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/aixiasang/lsm/inner/myerror"
	"github.com/spaolacci/murmur3"
)

// BloomFilter 是布隆过滤器的实现
type BloomFilter struct {
	m    uint64   // 位数组大小
	k    uint     // 哈希函数数量
	bits []uint64 // 位数组，每64位为一组
	n    uint64   // 已添加元素数量
	// 以下字段用于固定种子的哈希函数
	seeds []uint32 // 哈希函数种子，保证持久性
}

// 默认种子，为了确保哈希函数的一致性和可恢复性
var defaultSeeds = []uint32{
	0x47b6137b,
	0x44974d91,
	0x8824ad5b,
	0xa2b7289d,
	0x705495c7,
	0x2df1424b,
	0x9efc4947,
	0x5c6bfb31,
}

// NewBloomFilter 创建一个新的布隆过滤器
func NewBloomFilter(m uint64, k uint) Filter {
	if m == 0 {
		m = 1024 // 默认大小
	}
	if k == 0 {
		k = 3 // 默认哈希函数数量
	}

	// 确保使用默认种子并且数量足够
	seeds := make([]uint32, k)
	for i := uint(0); i < k; i++ {
		if i < uint(len(defaultSeeds)) {
			seeds[i] = defaultSeeds[i]
		} else {
			// 如果默认种子不足，使用默认种子的组合
			seeds[i] = defaultSeeds[i%uint(len(defaultSeeds))] + uint32(i/uint(len(defaultSeeds))*7)
		}
	}

	return &BloomFilter{
		m:     m,
		k:     k,
		bits:  make([]uint64, (m+63)/64), // 向上取整到64的倍数
		n:     0,
		seeds: seeds,
	}
}

// NewBloomFilterWithParams 使用预期元素数量和误判率创建布隆过滤器
func NewBloomFilterWithParams(expectedElements uint64, falsePositiveRate float64) Filter {
	// 计算最佳的m和k值
	// m = -n*ln(p)/(ln(2)^2)
	m := uint64(math.Ceil(-float64(expectedElements) * math.Log(falsePositiveRate) / math.Pow(math.Log(2), 2)))
	// k = m/n * ln(2)
	k := uint(math.Ceil(float64(m) / float64(expectedElements) * math.Log(2)))

	// 确保k至少为1
	if k < 1 {
		k = 1
	}

	return NewBloomFilter(m, k)
}

// getHash 为一个key计算多个哈希值
func (bf *BloomFilter) getHash(key []byte) []uint64 {
	hashes := make([]uint64, bf.k)

	for i := uint(0); i < bf.k; i++ {
		// 使用固定的种子确保哈希函数的一致性
		h := murmur3.New64WithSeed(bf.seeds[i])
		h.Write(key)
		hashes[i] = h.Sum64() % bf.m
	}

	return hashes
}

// Add 将一个key添加到布隆过滤器中
func (bf *BloomFilter) Add(key []byte) {
	hashes := bf.getHash(key)

	for _, hash := range hashes {
		// 计算位置：哪一组uint64和组内的哪一位
		wordIndex := hash / 64
		bitIndex := hash % 64

		// 将对应位设置为1
		bf.bits[wordIndex] |= 1 << bitIndex
	}

	bf.n++
}

// Contains 检查一个key是否可能存在于布隆过滤器中
func (bf *BloomFilter) Contains(key []byte) bool {
	hashes := bf.getHash(key)

	for _, hash := range hashes {
		// 计算位置：哪一组uint64和组内的哪一位
		wordIndex := hash / 64
		bitIndex := hash % 64

		// 检查对应位是否为1
		if (bf.bits[wordIndex] & (1 << bitIndex)) == 0 {
			return false // 只要有一位不为1，就肯定不在集合中
		}
	}

	return true // 所有位都为1，可能在集合中（也可能是误判）
}

// FalsePositiveRate 计算当前误判率
func (bf *BloomFilter) FalsePositiveRate() float64 {
	// 计算公式: (1 - e^(-k*n/m))^k
	if bf.n == 0 || bf.m == 0 {
		return 0.0
	}

	exponent := -float64(bf.k) * float64(bf.n) / float64(bf.m)
	return math.Pow(1.0-math.Exp(exponent), float64(bf.k))
}

// Reset 重置布隆过滤器
func (bf *BloomFilter) Reset() {
	// 清空位数组
	for i := range bf.bits {
		bf.bits[i] = 0
	}
	bf.n = 0
}

// Save 将布隆过滤器序列化为字节数组
func (bf *BloomFilter) Save() []byte {
	// 计算需要的字节数
	// 元数据 (m, k, n) 各8字节，加上种子(k*4字节)，加上比特位(bits长度*8字节)
	bufSize := 24 + len(bf.seeds)*4 + len(bf.bits)*8
	buffer := bytes.NewBuffer(make([]byte, 0, bufSize))

	// 写入元数据 (m, k, n)
	binary.Write(buffer, binary.BigEndian, bf.m)
	binary.Write(buffer, binary.BigEndian, uint64(bf.k))
	binary.Write(buffer, binary.BigEndian, bf.n)

	// 写入种子
	for _, seed := range bf.seeds {
		binary.Write(buffer, binary.BigEndian, seed)
	}

	// 写入bits数组数据
	for _, bits := range bf.bits {
		binary.Write(buffer, binary.BigEndian, bits)
	}

	return buffer.Bytes()
}

// Load 从字节数组加载布隆过滤器
func (bf *BloomFilter) Load(data []byte) error {
	if len(data) < 24 {
		return myerror.ErrInvalidBloomFilter
	}

	reader := bytes.NewReader(data)

	// 读取元数据 (m, k, n)
	if err := binary.Read(reader, binary.BigEndian, &bf.m); err != nil {
		return err
	}

	var k uint64
	if err := binary.Read(reader, binary.BigEndian, &k); err != nil {
		return err
	}
	bf.k = uint(k)

	if err := binary.Read(reader, binary.BigEndian, &bf.n); err != nil {
		return err
	}

	// 读取种子
	bf.seeds = make([]uint32, bf.k)
	for i := uint(0); i < bf.k; i++ {
		if err := binary.Read(reader, binary.BigEndian, &bf.seeds[i]); err != nil {
			return err
		}
	}

	// 计算bits数组的大小并分配内存
	bitsLen := (bf.m + 63) / 64
	bf.bits = make([]uint64, bitsLen)

	// 读取bits数据
	for i := uint64(0); i < bitsLen; i++ {
		if err := binary.Read(reader, binary.BigEndian, &bf.bits[i]); err != nil {
			if errors.Is(err, io.EOF) && i > 0 {
				return myerror.ErrBloomFilterIncomplete
			}
			return err
		}
	}

	return nil
}
