当然！之前的 README 已经过时了，它没有体现出我们后来加入的**持久化、衰减机制、以及最关键的抽象化多语言 TextRank 算法**。

这是一份为你全面重构的、可以作为最终交付文档的 `README.md`。它详细阐述了已实现的所有功能、背后的技术决策与算法原理。

---

# 🌍 全球多国新闻舆情热度计算子系统 (Sentiment Analysis Subsystem)

## 📖 项目简介
本项目是一个基于 Go 微服务架构的**轻量级、无 LLM 依赖的全球新闻舆情热点计算系统**。

系统能够处理爬虫抓取的非标准、多语言原始数据，通过**统一的 TextRank 权重算法**和**基于信息源的阵地溯源模型**，实现对中、美、俄、法、日、阿拉伯等多个国家/地区的实时舆情热点捕捉、量化、衰减与持久化归档。

作为大型舆情监控平台的后端子系统，本项目在设计上充分考虑了**高性能、高扩展性与业务逻辑的严谨性**。

---

## ✨ 已实现核心功能与技术方案

| 功能点 | 实现方法 (技术与算法) |
| :--- | :--- |
| **无大模型 NLP 核心** | **自研通用 TextRank 算法**：基于词语共现图（Co-occurrence Graph）和模拟 PageRank 迭代计算，为各语言提供统一的关键词权重度量衡。 |
| **多语言架构** | **抽象工厂模式**：通过 `KeywordExtractor` 接口解耦，内置中文(`Jieba`)、日语(`Kagome+IPA词典`)及通用语种(`Unicode Regex: \p{L}+`)提取器。 |
| **舆情阵地溯源** | **URL 解析模型**：彻底抛弃不切实际的“用户IP追踪”，通过解析爬虫数据中的 `from`/`link` 字段，实现基于**域名白名单 + TLD推断**的精准归属地判定。 |
| **高性能脏数据处理** | **预编译正则提取**：针对爬虫输出的非标准类 JSON (含 `ObjectId`, `ISODate`, 单引号拼接)，实现靶向字段提取，性能远超完整 AST 解析。 |
| **实时热度聚合** | **Redis ZSET**：利用 `ZINCRBY` 原子操作实现 O(log(N)) 复杂度的热度实时累加；`ZREVRANGE` 提供即时的 TopN 热词排行榜。 |
| **热度时效性衰减** | **牛顿冷却定律 (Redis 实现)**：封装 `ZUNIONSTORE` 指令，通过权重乘法原子性地对整个热词榜单进行批量衰减，并清理低分“冷寂词”，模拟新闻热度的自然降温。 |
| **数据持久化归档** | **MySQL 存储**：设计了 `daily_hot_keywords` 表，封装了批量事务写入逻辑，为长期舆情趋势分析和历史回溯提供数据基础。 |
| **微服务架构** | **异步事件驱动**：通过 **RabbitMQ** 将“数据清洗/提取”与“热度聚合/存储”两个服务完全解耦，提升系统吞吐量和容错能力。 |

---

## 🛠 技术栈

### 后端开发
- **Language**: Golang 1.25 (基于 WSL2/Ubuntu 构建)
- **Config & Log**: `spf13/viper`, `uber-go/zap`
- **NLP**: `yanyiwu/gojieba` (中文), `ikawaha/kagome/v2` (日语)
- **Database Driver**: `go-sql-driver/mysql`, `go-redis/redis/v8`

### 基础设施与中间件
- **Message Queue**: RabbitMQ (服务间异步通信总线)
- **Cache & Real-time Storage**: Redis (核心热度计算引擎)
- **Persistent Storage**: MySQL (长期舆情数据归档)
- **Containerization**: Docker & Docker Compose

---

## 🧠 核心实现原理详解

### 1. TextRank：跨语言的统一权重“度量衡”
为确保“法国热词A的得分”和“日本热词B的得分”具有可比性，我们抽象了核心算法：
1. **分词(Tokenization)**: 各语言提取器（如 `Jieba`, `Kagome`）负责将长文本切分为有意义的词组序列，并过滤掉助词、虚词等噪音。
2. **计算(Calculation)**: 所有词组序列统一送入 `calculateTextRank` 函数。该函数通过构建词语共现图，并模拟 PageRank 算法进行 10 次迭代，最终输出每个词的全局重要性得分（权重）。这保证了所有语言最终产出的 `score` 都在一个统一的数学框架下。

### 2. URL 溯源：解决“谁在关注”的难题
在爬虫无法获取读者 IP 的前提下，我们通过解析新闻来源 URL 来判定其“舆论阵地”：
- **`people.com.cn`** 的报道，无论内容是什么，其产生的舆情热度首先归属于中国区 (`CN`)。
- **“俄罗斯的事，阿拉伯人最关注”** 的场景得以实现：当阿拉伯语媒体（如 `aljazeera.net`）大量转载或讨论关于俄罗斯的新闻时，我们的爬虫会从这些阿拉伯语站点抓取数据。`getCountryBySource` 函数通过 URL 域名识别出这是阿拉伯阵地，从而将该事件的热度精准地累加到 `ARAB`。

---

## 🚀 快速启动与测试

### 环境准备
确保本机已安装 Go 1.25+ 和 Docker。

### 1. 启动基础设施
启动 Redis, RabbitMQ 和 MySQL:
建库建表
```sql
-- 1. 创建名为 sentiment 的数据库，并支持完整的 UTF-8 字符（防止某些小语种乱码）
CREATE DATABASE IF NOT EXISTS `sentiment` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 2. 切换到该数据库
USE `sentiment`;

-- 3. 创建持久化归档表
CREATE TABLE IF NOT EXISTS `daily_hot_keywords` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `country` VARCHAR(32) NOT NULL COMMENT '国家/地区归属',
    `keyword` VARCHAR(128) NOT NULL COMMENT '热词',
    `score` FLOAT NOT NULL COMMENT '当日热度得分',
    `record_date` DATE NOT NULL COMMENT '归档日期',
    INDEX `idx_country_date` (`country`, `record_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='每日国家热词归档表';

-- 4. 查看是否创建成功
SHOW TABLES;
```
### 2. 准备数据与配置
- 将爬虫抓取的原始脏数据粘贴至项目根目录的 `test_data.txt`。
- 修改 `configs/config.yaml`，确保 Redis, RabbitMQ, MySQL 的地址和密码正确。

### 3. 启动服务
```bash
# 终端 1: 启动消费者
cd heat-service
go run cmd/main.go
# > Heat-Service is running and waiting for messages...

# 终端 2: 启动生产者，处理数据
cd crawler-service
go run cmd/main.go
# > Successfully pushed to MQ...
```

### 4. 验证结果
```bash
# 使用 --raw 参数正常显示中日文
redis-cli --raw

# 查看中国区实时热点
ZREVRANGE heat_map:CN 0 9 WITHSCORES
```

---

## 📈 后续架构演进
本服务已具备完整的流式处理和存储能力。后续的“衰减”与“归档”任务，推荐采用**外部触发**模式（符合云原生最佳实践），而不是在服务内存中硬编码定时器。
1. **外部定时触发 (K8s CronJob)**: 部署一个 K8s 定时任务，每日凌晨通过 HTTP 请求调用 `heat-service` 暴露的 `/api/v1/archive` 和 `/api/v1/decay` 接口，执行归档和衰减操作。
2. **数据可视化接口**: 在 `heat-service` 中增加一个 HTTP 服务，提供 `GET /api/v1/heatmap?country=CN` 接口，供前端大屏系统实时拉取 TopN 舆情数据。
