package analyzer

import (
	"context"
	"fmt"
	"strings"

	"sentiment/heat-service/internal/storage"
	"sentiment/pkg/logger"
	"sentiment/pkg/models"

	"go.uber.org/zap"
)

type HeatAnalyzer struct {
	redisStore *storage.RedisStore
}

func NewHeatAnalyzer(store *storage.RedisStore) *HeatAnalyzer {
	// 🌟 极其干净：不需要任何预设字典
	return &HeatAnalyzer{
		redisStore: store,
	}
}

// Process 接收文章事件，直接进行暴力聚合统计
func (a *HeatAnalyzer) Process(ctx context.Context, event models.ArticleEvent) {
	if event.Country == "" {
		event.Country = "UNKNOWN"
	}
	// 1. 获取当前该国的 Top 100 舆情热词
	currentHotKeywords, err := a.redisStore.GetTopN(ctx, event.Country, 100)
	if err != nil {
		logger.Log.Error("Failed to get top keywords from Redis", zap.Error(err))
		// 即使获取失败，也继续执行基础的累加，保证系统鲁棒性
	}
	
	// 为了 O(1) 查询，把 Redis 结果转为 map
	currentHotMap := make(map[string]float64)
	for _, z := range currentHotKeywords {
		currentHotMap[z.Member.(string)] = z.Score
	}

	// 2. 计算趋势契合分
	var trendAlignmentScore float64
	var overlapCount int
	for word := range event.Keywords {
		if score, ok := currentHotMap[word]; ok {
			// 如果文章的词命中了当前热搜榜，就把热搜榜上的分数加到契合分里
			trendAlignmentScore += score
			overlapCount++
		}
	}

	// 3. 决策：如果重合度高，说明这篇文章在“蹭热点”，也是一种热点
	isHotspot := "No"
	if overlapCount > 3 && trendAlignmentScore > 100.0 { // 至少重合3个词，且总分超过100
		isHotspot = "YES"
	}
	
	logger.Log.Info("Trend alignment calculated", 
		zap.String("id", event.ArticleID),
		zap.Float64("alignment_score", trendAlignmentScore),
		zap.Int("overlap_count", overlapCount),
		zap.String("is_trend_hotspot", isHotspot))

	successCount := 0
	// 🌟 直接遍历所有的词，不进行任何 if exists 的匹配！
	for word, weight := range event.Keywords {
		
		// 过滤掉太短的无意义字符 (可选)
		if len(word) <= 1 {
			continue
		}

		// 直接原子累加到对应国家的 ZSET 中
		err := a.redisStore.IncrKeywordHeat(ctx, event.Country, word, weight)
		if err != nil {
			logger.Log.Error("Redis save failed", zap.Error(err), zap.String("word", word))
			continue
		}
		successCount++
	}

	logger.Log.Info("Aggregated keywords to Redis", 
		zap.String("country", event.Country), 
		zap.Int("words_counted", successCount),
		zap.String("article_id", event.ArticleID))

	if err := a.redisStore.SaveArticleScore(ctx, event.Country, event.ArticleID, event.Title, event.Score); err != nil {
		logger.Log.Error("Failed to save article score", zap.Error(err), zap.String("article_id", event.ArticleID))
		return
	}

	topArticles, err := a.redisStore.GetTopNArticleTitles(ctx, event.Country, 10)
	if err != nil {
		logger.Log.Error("Failed to load top scored articles", zap.Error(err), zap.String("country", event.Country))
		return
	}

	lines := make([]string, 0, len(topArticles))
	for i, article := range topArticles {
		title := strings.TrimSpace(article.Title)
		if title == "" {
			title = "(untitled)"
		}
		lines = append(lines, fmt.Sprintf("%d. %s (score=%.2f)", i+1, title, article.Score))
	}

	logger.Log.Info("Top 10 news titles by score",
		zap.String("country", event.Country),
		zap.String("top10", strings.Join(lines, " | ")))
}