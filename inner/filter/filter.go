package filter

// Filter 过滤器接口
type Filter interface {
	Add(key []byte)           // 添加key
	Contains(key []byte) bool // 判断key是否存在
	Save() []byte             // 保存到文件
	Load(data []byte) error   // 从文件加载
	Reset()                   // 重置
}
