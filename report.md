# 全球新闻舆情子系统报告

## 1. 报告目标

1. 一篇新闻是如何被打分的？
2. 热词是如何被选出来并持续更新的？
3. 为什么系统能输出“前十新闻标题”？

---

## 2. 系统关键链路总览

完整链路如下：

MongoDB -> mongo-exporter -> test_data.txt -> crawler-service -> RabbitMQ -> heat-service -> Redis/MySQL

各环节职责：

1. `mongo-exporter`：把 MongoDB 文档导出为 JSONL（test_data.txt）。
2. `crawler-service`：对每篇新闻做关键词抽取、国家归属判定、文章评分，并发布事件。
3. `heat-service`：消费事件，更新热词榜和新闻得分榜，输出前十标题。
4. Redis：承载实时排序结构（热词和新闻榜单）。
5. MySQL：用于长期归档（当前链路重点是实时计算）。

---

## 3. 核心链路 A：新闻评分系统如何工作

### 3.1 输入

每篇新闻至少包含以下字段：

1. `article_id`
2. `title`
3. `text`
4. `language`
5. `from`（来源站点）

### 3.2 关键词权重生成

评分前先提取关键词，策略是“多语言分词 + 统一权重语义”：

1. 中文走中文提取器。
2. 日语走日语提取器。
3. 其他语言走通用提取器。

输出是 `keywords: map[word]weight`。

### 3.3 国家归属判定

系统先按来源域名/TLD 判断国家；不确定时再按语言兜底。

这一步确保同一主题新闻在不同媒体阵地可分国家统计，而不是混成一个全局热度。

### 3.4 文章潜力分计算

对每篇新闻计算潜力分：

$$
S_{article} = 0.6 \cdot \sum_{k \in K} w_k + 0.4 \cdot (10 \cdot a)
$$

其中：

1. $w_k$：关键词权重。
2. $a$：信源权威度（按域名映射）。

解读：

1. 第一项刻画“文本内容强度”。
2. 第二项刻画“信源影响力”。
3. 两者做线性融合，避免只看词频或只看来源。

### 3.5 评分结果出队

评分后，`crawler-service` 发布 MQ 事件，核心字段包括：

1. `article_id`
2. `title`
3. `score`
4. `country`
5. `language`
6. `keywords`

这就是后续“热词更新”和“前十标题榜”共同依赖的数据契约。

---

## 4. 核心链路 B：热词如何被选择并更新

`heat-service` 收到一条事件后，会在同一次处理中完成两套计算：趋势判断 + 热词聚合。

### 4.1 趋势契合分（判断是否蹭到当前热点）

先读取该国家当前 Top100 热词，转为 map 后计算交集得分：

$$
S_{align} = \sum_{k \in K \cap H} heat(k)
$$

其中：

1. $K$：当前新闻关键词集合。
2. $H$：该国家当前热词集合。

系统同时统计重合词数量，形成判定信号：

1. 重合词数 > 3
2. 且 $S_{align} > 100$

满足时记为趋势热点新闻。

### 4.2 热词榜更新

无论是否命中趋势热点，系统都会把本篇关键词写入 Redis：

1. key: `heat_map:<country>`（ZSET）
2. 操作：`ZINCRBY`，分值增量 = 关键词权重

结果是热词分数被持续累加，形成实时排行榜。

### 4.3 热词时效性（衰减）

系统提供衰减接口，通过乘法衰减模拟新闻降温：

$$
score_t = \lambda \cdot score_{t-1}, \quad 0 < \lambda < 1
$$

并清理过低分词项，抑制长尾噪声。

---

## 5. 核心链路 C：为什么能输出前十新闻标题

这是本次改动后的关键能力。

### 5.1 新闻榜单写入

每条事件到达 `heat-service` 后，系统会：

1. 把 `score` 写入 `article_rank:<country>`（ZSET，member=article_id）。
2. 把 `title` 写入 `article_title:<country>`（HASH，field=article_id）。

这样“排序信息”和“展示信息”被拆分存储：

1. ZSET 负责高效排名。
2. HASH 负责标题映射。

### 5.2 前十标题读取

系统按分数倒序取前 10 个 article_id，再到 HASH 取标题并拼接日志输出：

`Top 10 news titles by score`

因此，前十标题的本质是“ZSET 排序 + HASH 回填展示字段”。

---

## 6. 关键数据结构设计（Redis）

1. `heat_map:<country>`
   - 类型：ZSET
   - 语义：国家热词榜
   - member：keyword
   - score：累计热度

2. `article_rank:<country>`
   - 类型：ZSET
   - 语义：国家新闻得分榜
   - member：article_id
   - score：新闻评分

3. `article_title:<country>`
   - 类型：HASH
   - 语义：文章标题映射
   - field：article_id
   - value：title

这一设计使“热词分析”和“新闻榜单”既共享事件输入，又互不干扰。

---

## 7. 验证与观测

### 7.1 端到端运行

在仓库根目录执行：

```bash
make prepare-data
make run-heat
make run-crawler
```

### 7.2 关键观测点

1. `mongo-exporter`：`export finished` 的 `documents` 是否大于 0。
2. `crawler-service`：是否出现 `Successfully pushed to MQ`。
3. `heat-service`：是否出现 `Top 10 news titles by score`。

### 7.3 Redis 快速检查

```bash
redis-cli --raw
ZREVRANGE heat_map:CN 0 9 WITHSCORES
ZREVRANGE article_rank:CN 0 9 WITHSCORES
HGET article_title:CN <article_id>
```

---

## 8. 当前结论

1. 先用统一机制给新闻打分（内容强度 + 信源权威）。
2. 再按国家维度做热词聚合与趋势判断。
3. 最后基于实时排序结构输出前十新闻标题。

这三步共同保证了结果既有实时性，也有业务可解释性。
