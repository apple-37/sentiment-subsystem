package cleaner

import (
	"github.com/yanyiwu/gojieba"
)

type TextCleaner struct {
	jieba *gojieba.Jieba
}

func NewTextCleaner() *TextCleaner {
	return &TextCleaner{
		jieba: gojieba.NewJieba(),
	}
}

func (c *TextCleaner) Close() {
	c.jieba.Free()
}

// ExtractKeywords 提取高频词及 TextRank 权重
func (c *TextCleaner) ExtractKeywords(text string, topN int) map[string]float64 {
	words := c.jieba.ExtractWithWeight(text, topN)
	
	result := make(map[string]float64)
	for _, word := range words {
		result[word.Word] = word.Weight
	}
	
	return result
}