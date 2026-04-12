package extractor

import (
	"github.com/ikawaha/kagome-dict/ipa" // 引入 IPA 标准日语词典
	"github.com/ikawaha/kagome/v2/tokenizer"
)

type JapaneseExtractor struct {
	t *tokenizer.Tokenizer
}

func NewJapaneseExtractor() *JapaneseExtractor {
	// 🌟 传入 IPA 词典，并配置忽略句首(BOS)和句尾(EOS)的虚拟 Token
	t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		// 字典加载失败属于严重致命错误，直接 panic
		panic(err)
	}
	return &JapaneseExtractor{t: t}
}

func (j *JapaneseExtractor) Extract(text string, topN int) map[string]float64 {
	tokens := j.t.Tokenize(text)
	
	var validWords[]string
	for _, token := range tokens {
		// 过滤掉虚拟节点
		if token.Class == tokenizer.DUMMY {
			continue
		}
		
		// 获取词性特征 (Features)
		features := token.Features()
		if len(features) > 0 {
			// features[0] 代表主要词性（品詞）。在抽取热词时，通常只保留“名詞”（名词）
			if features[0] == "名詞" {
				validWords = append(validWords, token.Surface)
			}
		}
	}

	// 🌟 共享我们自己手写的 TextRank 统计算法
	return calculateTextRank(validWords, topN)
}

func (j *JapaneseExtractor) Close() {
	// Kagome 底层纯 Go 实现，靠 GC 自动回收，无需手动 Free
}