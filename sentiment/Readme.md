# 全球多国新闻舆情热度计算子系统

## 项目定位

该子系统负责把新闻文本转成可计算的舆情信号，并输出两类实时结果：

1. 国家维度热词榜（关键词热度）
2. 新闻得分榜（按分数排序的前十标题）

系统采用异步微服务链路：`crawler-service` 负责打分并发布事件，`heat-service` 负责聚合并产出榜单。

## 关键链路（建议先看）

### 1. 输入与解析

`crawler-service` 从 `test_data.txt` 按行读取 JSONL 新闻，过滤空正文，然后得到：

1. 文章基础信息（id、title、language、from）
2. 国家归属（根据域名/TLD + 语言兜底）

### 2. 关键词提取与文章评分

每篇新闻根据语言选择提取器（中文/日语/通用），统一输出关键词权重 `Keywords`。

文章分数（潜力分）计算公式：

$$
S_{article} = 0.6 \cdot \sum_{k \in K} w_k + 0.4 \cdot (10 \cdot a)
$$

其中：

1. $w_k$：关键词权重（TextRank/提词器输出）
2. $a$：信源权威度（按来源域名映射）

分数、标题和关键词会被打包进 MQ 事件：

1. `article_id`
2. `title`
3. `score`
4. `country`
5. `language`
6. `keywords`

### 3. 消费聚合与热词更新

`heat-service` 消费事件后执行两条并行价值链：

1. 热词链：把本篇新闻的关键词按权重写入 `heat_map:<country>`（Redis ZSET）
2. 新闻链：把本篇新闻分数写入 `article_rank:<country>`，并把标题写入 `article_title:<country>`

### 4. 输出前十标题

每处理一篇新闻，服务都会读取 `article_rank:<country>` 前 10 项，拼接标题并输出日志：

`Top 10 news titles by score`

## 热词与评分各自回答的问题

1. 热词榜回答“这个国家现在在讨论什么词”。
2. 新闻榜回答“这个国家当前最值得关注的是哪几篇文章”。

两者不是互斥关系：同一篇文章既能推动热词上升，也能进入新闻标题榜。

## 技术栈

1. 语言：Go 1.25+
2. MQ：RabbitMQ
3. 缓存与实时排序：Redis ZSET/HASH
4. 持久化：MySQL
5. 日志：zap
6. 配置：viper

## 快速运行

在仓库根目录可直接使用 `Makefile` 串起流程：

```bash
make prepare-data   # Mongo 导出并复制到 sentiment/test_data.txt
make run-heat       # 启动 heat-service
make run-crawler    # 启动 crawler-service
```

## 验证方式

### 1. 看服务日志

1. `crawler-service` 应看到 `Successfully pushed to MQ`
2. `heat-service` 应看到 `Top 10 news titles by score`

### 2. 看 Redis

```bash
redis-cli --raw
ZREVRANGE heat_map:CN 0 9 WITHSCORES
ZREVRANGE article_rank:CN 0 9 WITHSCORES
HGET article_title:CN <article_id>
```

## 注意事项

1. 如果导出日志显示 `documents: 0`，通常是 Mongo 数据库/集合配置不匹配，而不是代码异常。
2. 当前“前十标题”以日志形式输出；如需 API 查询，可在 `heat-service` 增加 HTTP 只读接口。
