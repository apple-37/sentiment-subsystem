package extractor

import "github.com/yanyiwu/gojieba"

type ChineseExtractor struct {
	jieba *gojieba.Jieba
}

func NewChineseExtractor() *ChineseExtractor {
	return &ChineseExtractor{
		jieba: gojieba.NewJieba(),
	}
}

func (c *ChineseExtractor) Extract(text string, topN int) map[string]float64 {
	words := c.jieba.ExtractWithWeight(text, topN)
	result := make(map[string]float64)
	for _, word := range words {
		result[word.Word] = word.Weight
	}
	return result
}

func (c *ChineseExtractor) Close() {
	c.jieba.Free()
}