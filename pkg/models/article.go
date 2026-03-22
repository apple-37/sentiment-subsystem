package models

import "time"

type RawArticle struct {
	ID          string    `json:"_id"`
	Title       string    `json:"title"`
	Text        string    `json:"text"`
	Language    string    `json:"language"`
	From        string    `json:"from"`
	PublishTime time.Time `json:"publish_time"`
}

type ArticleEvent struct {
	ArticleID string             `json:"article_id"`
	Country   string             `json:"country"`  // 🌟 新增：由爬虫端或外部传入的国家归属
	Language  string             `json:"language"`
	Keywords  map[string]float64 `json:"keywords"` // 🌟 全部词频，不作任何过滤
}