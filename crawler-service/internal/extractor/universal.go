package extractor

import (
	"regexp"
	"strings"
)

type UniversalExtractor struct {
	wordReg *regexp.Regexp
}

func NewUniversalExtractor() *UniversalExtractor {
	return &UniversalExtractor{
		wordReg: regexp.MustCompile(`[\p{L}]+`), // 匹配各国字母字符
	}
}

func (u *UniversalExtractor) Extract(text string, topN int) map[string]float64 {
	text = strings.ToLower(text)
	matches := u.wordReg.FindAllString(text, -1)

	var validWords[]string
	for _, word := range matches {
		// 过滤长度 <= 3 的无意义虚词 (le, a, the, of 等)
		if len(word) > 3 {
			validWords = append(validWords, word)
		}
	}

	// 🌟 调用通用 TextRank
	return calculateTextRank(validWords, topN)
}

func (u *UniversalExtractor) Close() {}