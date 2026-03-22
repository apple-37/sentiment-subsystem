package parser

import (
	"regexp"
	"strings"

	"sentiment/pkg/models"
)

var (
	// 预编译正则，保证极高匹配性能
	// 1. 匹配每一个大括号包含的完整记录块
	blockReg = regexp.MustCompile(`(?s)\{.*?\n\s*\}`)
	// 2. 提取基础字段
	idReg   = regexp.MustCompile(`_id:\s*ObjectId\('([^']+)'\)`)
	langReg = regexp.MustCompile(`language:\s*'([^']+)'`)
	// 3. 提取 text 块 (从 text: 到下一个字段前)
	textBlockReg = regexp.MustCompile(`(?s)text:\s*([\s\S]+?),\s*\w+:`)
	// 4. 提取被单引号包裹的真实文本，处理多行拼接
	stringLiteralReg = regexp.MustCompile(`'(?:\\'|[^'])*'`)
)

// ParseRawMongoDump 将爬虫丢过来的脏日志解析为结构化的 Article 列表
func ParseRawMongoDump(rawData string) []models.RawArticle {
	var articles[]models.RawArticle

	// 找到所有的 {...} 数据块
	blocks := blockReg.FindAllString(rawData, -1)

	for _, block := range blocks {
		article := models.RawArticle{}

		// 提取 ID
		if m := idReg.FindStringSubmatch(block); len(m) > 1 {
			article.ID = m[1]
		}

		// 提取语言
		if m := langReg.FindStringSubmatch(block); len(m) > 1 {
			article.Language = m[1]
		}

		// 提取并清洗文本 (核心难点：处理 '\n' + '...' 这种恶心的拼接)
		if m := textBlockReg.FindStringSubmatch(block); len(m) > 1 {
			rawTextBlock := m[1]
			// 找到所有的单引号字符串
			strMatches := stringLiteralReg.FindAllString(rawTextBlock, -1)
			
			var fullText strings.Builder
			for _, str := range strMatches {
				// 去掉首尾的单引号
				cleanStr := strings.Trim(str, "'")
				// 替换转义的单引号 (\') 为正常单引号
				cleanStr = strings.ReplaceAll(cleanStr, `\'`, `'`)
				// 将所有的换行符 \n 替换为实际的换行
				cleanStr = strings.ReplaceAll(cleanStr, `\n`, "\n")
				fullText.WriteString(cleanStr)
			}
			article.Text = fullText.String()
		}

		// 只有解析出有效 ID 的记录才算成功
		if article.ID != "" {
			articles = append(articles, article)
		}
	}

	return articles
}