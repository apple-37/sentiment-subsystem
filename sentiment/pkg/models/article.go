package models

import "time"

type RawArticle struct {
	ID          string    `json:"_id"`          // 匹配 JSON 中的 _id
	Authors     []string  `json:"authors"`      // 匹配 authors 数组
	Language    string    `json:"language"`     // 匹配 language
	Text        string    `json:"text"`         // 匹配正文
	Title       string    `json:"title"`        // 匹配标题
	Link        string    `json:"link"`         // 匹配链接
	From        string    `json:"from"`         // 匹配数据来源
	PublishTime time.Time `json:"publish_time"` // 匹配发布时间
}

type ArticleEvent struct {
	ArticleID string             `json:"article_id"`
	Country   string             `json:"country"` // 🌟 新增：由爬虫端或外部传入的国家归属
	Language  string             `json:"language"`
	Keywords  map[string]float64 `json:"keywords"` // 🌟 全部词频，不作任何过滤
}
