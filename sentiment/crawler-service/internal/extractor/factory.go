package extractor

import "strings"

// KeywordExtractor 定义了所有语言提取器必须实现的通用接口
type KeywordExtractor interface {
	// Extract 提取文本中的 TopN 关键词及其权重
	Extract(text string, topN int) map[string]float64
	// Close 释放资源 (例如 CGO 绑定的内存)
	Close()
}

// ExtractorFactory 提取器工厂
type ExtractorFactory struct {
	zhExtractor KeywordExtractor // 专用的中文提取器
	universal   KeywordExtractor // 通用的多语言提取器 (英/法/德/俄/阿拉伯)
	jaExtractor KeywordExtractor // 🌟 增加日语
}

// NewExtractorFactory 初始化所有提取器
func NewExtractorFactory() *ExtractorFactory {
	return &ExtractorFactory{
		zhExtractor: NewChineseExtractor(),
		universal:   NewUniversalExtractor(),
		jaExtractor: NewJapaneseExtractor(),
	}
}

// GetExtractor 根据语言代码返回对应的实现类 (多态)
func (f *ExtractorFactory) GetExtractor(lang string) KeywordExtractor {
	lang = strings.ToLower(lang)
	switch lang {
	case "zh", "zh-cn":
		return f.zhExtractor
	case "ja": return f.jaExtractor
	default:
		// 英文、法语、俄语、德语、阿拉伯语等，全部走通用提取器
		return f.universal
	}
}

func (f *ExtractorFactory) CloseAll() {
	if f.zhExtractor != nil { f.zhExtractor.Close() }
	if f.jaExtractor != nil { f.jaExtractor.Close() }
	if f.universal != nil { f.universal.Close() }
}