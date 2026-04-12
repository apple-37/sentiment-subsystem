package parser

import (
	"bufio"
	"encoding/json"
	"strings"

	"sentiment/pkg/models"
)

// ParseRawJSONL 解析标准的 JSONL 格式内容
// 这种格式每行都是一个合法的 JSON 对象
func ParseRawJSONL(rawData string) []models.RawArticle {
	var articles []models.RawArticle

	// 使用 Scanner 逐行处理，适应大数据量
	scanner := bufio.NewScanner(strings.NewReader(rawData))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行或非 JSON 行（如你提供的文件头描述 --- START OF FILE ---）
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var article models.RawArticle
		// 使用标准库解析 JSON
		err := json.Unmarshal([]byte(line), &article)
		if err != nil {
			// 如果单行解析失败，跳过并继续处理下一行
			continue
		}

		// 验证关键字段（如 ID 不为空）
		if article.ID != "" {
			articles = append(articles, article)
		}
	}

	return articles
}
