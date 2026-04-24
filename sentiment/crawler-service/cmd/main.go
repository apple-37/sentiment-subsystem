package main

import (
	"net/url"
	"os"
	"strings"

	"sentiment/crawler-service/internal/extractor"
	"sentiment/crawler-service/internal/parser"
	"sentiment/crawler-service/internal/publisher"
	"sentiment/pkg/config"
	"sentiment/pkg/logger"
	"sentiment/pkg/models"

	"go.uber.org/zap"
)

var domainToCountry = map[string]string{
	"people.com.cn": "CN",
	"xinhua.net":    "CN",
	"nouvelobs.com": "FR",
	"lemonde.fr":    "FR",
	"aljazeera.net": "ARAB",
	"nytimes.com":   "USA",
	"yahoo.co.jp":   "JP",
}

// 新增：信源权威度权重表 (1.0 为基准，越高越权威)
var sourceAuthority = map[string]float64{
	"people.com.cn": 1.5,
	"xinhua.net":    1.5,
	"nytimes.com":   1.4,
	"lemonde.fr":    1.2,
	"aljazeera.net": 1.2,
	"nouvelobs.com": 1.0,
}

func getCountryBySource(sourceURL string, lang string) string {
	u, err := url.Parse(sourceURL)
	host := ""
	if err == nil && u.Host != "" {
		host = strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	}

	// 1. 先尝试域名后缀匹配 (这是最快的特征)
	if strings.HasSuffix(host, ".jp") {
		return "JP"
	}
	if strings.HasSuffix(host, ".fr") {
		return "FR"
	}
	if strings.HasSuffix(host, ".ru") {
		return "RU"
	}
	if strings.HasSuffix(host, ".de") {
		return "DE"
	}
	if strings.HasSuffix(host, ".es") {
		return "ES"
	}
	if strings.HasSuffix(host, ".cn") {
		return "CN"
	}

	// 2. 补充你数据里出现的特定大站
	if strings.Contains(host, "techcrunch.com") {
		return "USA"
	}
	if strings.Contains(host, "nytimes.com") {
		return "USA"
	}
	if strings.Contains(host, "reuters") {
		return "UK"
	}
	if strings.Contains(host, "aljazeera") {
		return "ARAB"
	}

	// 3. 根据语言代码兜底映射 (你的数据里 language 字段是 en, fr, de 等)
	switch strings.ToLower(lang) {
	case "zh", "zh-cn":
		return "CN"
	case "en":
		return "USA/UK" // 英文通常归类到主要英语国家
	case "ja":
		return "JP"
	case "fr":
		return "FR"
	case "de":
		return "DE"
	case "es":
		return "ES"
	case "ru":
		return "RU"
	case "ar":
		return "ARAB"
	default:
		return "GLOBAL" // 替代 UNKNOWN，好听一点
	}
}

func main() {
	cfg := config.LoadConfig()
	logger.InitLogger(cfg.App.LogLevel)
	defer logger.Log.Sync()

	mqPub, err := publisher.NewRabbitMQPublisher(cfg.RabbitMQ.URL, cfg.RabbitMQ.QueueName)
	if err != nil {
		logger.Log.Fatal("Failed to connect MQ", zap.Error(err))
	}
	defer mqPub.Close()

	// 🌟 1. 初始化工厂
	extractorFactory := extractor.NewExtractorFactory()
	defer extractorFactory.CloseAll()

	byteValue, err := os.ReadFile("../test_data.txt")
	if err != nil {
		logger.Log.Fatal("Cannot open input data", zap.Error(err))
	}

	articles := parser.ParseRawJSONL(string(byteValue))
	logger.Log.Info("Parsed raw data", zap.Int("count", len(articles)))

	for _, rawArticle := range articles {
		if rawArticle.Text == "" {
			continue
		}

		source := rawArticle.From
		targetCountry := getCountryBySource(source, rawArticle.Language)

		// 🌟 2. 核心：通过工厂获取对应语言的提取器
		worker := extractorFactory.GetExtractor(rawArticle.Language)

		// 🌟 3. 调用统一接口进行 TextRank 计算
		keywordsWeight := worker.Extract(rawArticle.Text, 30)
		var keywordSignificanceSum float64
		for _, weight := range keywordsWeight {
			keywordSignificanceSum += weight
		}

		// 2. 获取信源权威度
		authorityScore := 1.0 // 默认为 1.0
		for domain, score := range sourceAuthority {
			if strings.Contains(source, domain) {
				authorityScore = score
				break
			}
		}

		// 3. 计算最终潜力分 (权重可调，比如信源权重占 40%)
		// 这个公式完全可以根据业务需求调整
		hotspotPotentialScore := (keywordSignificanceSum * 0.6) + (authorityScore * 10 * 0.4) // 乘以10是为了平衡量级

		// 4. 决策：如果潜力分超过阈值，就标记为潜在热点
		isHotspot := "No"
		if hotspotPotentialScore > 50.0 { // 50.0 是一个需要根据数据调试的经验阈值
			isHotspot = "YES"
		}
		// 增加这行日志，排查为何没结果
		logger.Log.Debug("Processing Language Detail",
			zap.String("lang", rawArticle.Language),
			zap.Int("keyword_count", len(keywordsWeight)),
			zap.Float64("weight_sum", keywordSignificanceSum))

		logger.Log.Info("Article Processed",
			zap.String("id", rawArticle.ID),
			zap.Float64("potential_score", hotspotPotentialScore),
			zap.String("is_potential_hotspot", isHotspot)) // 直接在日志里输出决策
		event := models.ArticleEvent{
			ArticleID: rawArticle.ID,
			Title:     rawArticle.Title,
			Score:     hotspotPotentialScore,
			Country:   targetCountry,
			Language:  rawArticle.Language,
			Keywords:  keywordsWeight,
		}

		if err := mqPub.Publish(event); err != nil {
			logger.Log.Error("Publish failed", zap.String("id", event.ArticleID), zap.Error(err))
		} else {
			logger.Log.Info("Successfully pushed to MQ",
				zap.String("id", event.ArticleID),
				zap.String("country", targetCountry))
		}
	}
}
