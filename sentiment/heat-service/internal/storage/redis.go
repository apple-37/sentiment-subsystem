package storage

import (
	"context"

	"github.com/go-redis/redis/v8"
)

type ArticleRank struct {
	ArticleID string
	Title     string
	Score     float64
}

type RedisStore struct {
	client *redis.Client
}

// 修改函数，增加 password 参数
func NewRedisStore(addr string, password string) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password, // 这里的 password 就是你设置的密码
			DB:       0,        // 默认数据库为 0，也可以根据需要指定
		}),
	}
}

// IncrKeywordHeat 在 Redis 的 SortedSet 中增加词汇热度
func (r *RedisStore) IncrKeywordHeat(ctx context.Context, country, keyword string, weight float64) error {
	key := "heat_map:" + country // 例如: heat_map:CN
	return r.client.ZIncrBy(ctx, key, weight, keyword).Err()
}

// GetTopN 获取某个国家的 TopN 热词，用于归档
func (r *RedisStore) GetTopN(ctx context.Context, country string, n int64) ([]redis.Z, error) {
	key := "heat_map:" + country
	// 0 到 n-1 即代表前 N 个
	return r.client.ZRevRangeWithScores(ctx, key, 0, n-1).Result()
}

// DecayHeatMap 时效性衰减（牛顿冷却定律变形）
// factor: 衰减系数，例如 0.8 表示当前热度打八折
func (r *RedisStore) DecayHeatMap(ctx context.Context, country string, factor float64) error {
	key := "heat_map:" + country
	
	// 核心魔法：ZUNIONSTORE 覆盖自身，并应用 weights 权重
	// 这等同于遍历所有元素执行 score = score * factor，但它是底层的 C 语言批量操作，极快！
	err := r.client.ZUnionStore(ctx, key, &redis.ZStore{
		Keys:    []string{key},
		Weights:[]float64{factor},
	}).Err()
	if err != nil {
		return err
	}

	// 附带清理机制：为了防止长尾词占用过多内存，移除热度低于 1.0 的“冷寂词”
	return r.client.ZRemRangeByScore(ctx, key, "-inf", "1.0").Err()
}

// SaveArticleScore 保存新闻得分，并记录标题用于榜单展示
func (r *RedisStore) SaveArticleScore(ctx context.Context, country, articleID, title string, score float64) error {
	rankKey := "article_rank:" + country
	titleKey := "article_title:" + country

	pipe := r.client.TxPipeline()
	pipe.ZAdd(ctx, rankKey, &redis.Z{Score: score, Member: articleID})
	pipe.HSet(ctx, titleKey, articleID, title)
	_, err := pipe.Exec(ctx)
	return err
}

// GetTopNArticleTitles 获取某个国家得分最高的前 N 篇新闻
func (r *RedisStore) GetTopNArticleTitles(ctx context.Context, country string, n int64) ([]ArticleRank, error) {
	rankKey := "article_rank:" + country
	titleKey := "article_title:" + country

	top, err := r.client.ZRevRangeWithScores(ctx, rankKey, 0, n-1).Result()
	if err != nil {
		return nil, err
	}

	result := make([]ArticleRank, 0, len(top))
	for _, item := range top {
		articleID, ok := item.Member.(string)
		if !ok {
			continue
		}
		title, err := r.client.HGet(ctx, titleKey, articleID).Result()
		if err == redis.Nil {
			title = ""
		} else if err != nil {
			return nil, err
		}
		result = append(result, ArticleRank{
			ArticleID: articleID,
			Title:     title,
			Score:     item.Score,
		})
	}

	return result, nil
}