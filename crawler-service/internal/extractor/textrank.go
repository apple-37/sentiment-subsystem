package extractor

import (
	"sort"
)

// calculateTextRank 是一个标准的 TextRank 算法实现 (模拟 PageRank)
// 它接受切分好的词组序列，并返回计算出权重的 TopN
func calculateTextRank(words []string, topN int) map[string]float64 {
	// 1. 构建共现图 (窗口大小设为 5，与 Jieba 默认一致)
	window := 5
	graph := make(map[string][]string)
	
	for i, w1 := range words {
		for j := i + 1; j < i+window && j < len(words); j++ {
			w2 := words[j]
			if w1 != w2 {
				graph[w1] = append(graph[w1], w2)
				graph[w2] = append(graph[w2], w1) // 无向图
			}
		}
	}

	// 2. 初始化权重为 1.0
	weights := make(map[string]float64)
	for k := range graph {
		weights[k] = 1.0
	}

	// 3. 迭代计算 (阻尼系数 d=0.85, 迭代 10 次足够收敛)
	d := 0.85
	for iter := 0; iter < 10; iter++ {
		newWeights := make(map[string]float64)
		for vi, neighbors := range graph {
			var sum float64
			for _, vj := range neighbors {
				// 邻居 vj 将自己的权重平分给所有的边
				sum += weights[vj] / float64(len(graph[vj]))
			}
			newWeights[vi] = (1 - d) + d*sum
		}
		weights = newWeights
	}

	// 4. 排序截取 TopN
	type kv struct {
		Key   string
		Value float64
	}
	var ss[]kv
	for k, v := range weights {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value // 降序
	})

	result := make(map[string]float64)
	for i := 0; i < len(ss) && i < topN; i++ {
		result[ss[i].Key] = ss[i].Value
	}

	return result
}