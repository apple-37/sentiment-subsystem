package consumer

import (
	"context"
	"encoding/json"

	"sentiment/heat-service/internal/analyzer"
	"sentiment/pkg/logger"
	"sentiment/pkg/models"

	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type RabbitMQConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	queue    amqp.Queue
	analyzer *analyzer.HeatAnalyzer // 注入分析器
}

func NewRabbitMQConsumer(mqURL, queueName string, analyzer *analyzer.HeatAnalyzer) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(mqURL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	// 声明队列，确保它存在
	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	return &RabbitMQConsumer{conn, ch, q, analyzer}, err
}

// Start 开始阻塞监听消息
func (c *RabbitMQConsumer) Start(ctx context.Context) {
	msgs, err := c.channel.Consume(c.queue.Name, "", true, false, false, false, nil)
	if err != nil {
		logger.Log.Error("Failed to register a consumer", zap.Error(err))
		return
	}

	for {
		select {
		case <-ctx.Done(): // 接收到停机信号
			logger.Log.Info("Consumer gracefully shutting down...")
			return
		case d := <-msgs: // 接收到消息
			if d.Body == nil {
				continue
			}

			var event models.ArticleEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				logger.Log.Error("Failed to unmarshal event", zap.Error(err))
				continue
			}
			// 🌟 新增这一行！看看我们到底提取出了哪些词！
			logger.Log.Info("📥 Received Article from MQ", 
				zap.String("id", event.ArticleID), 
				zap.Any("keywords", event.Keywords))

			// 送入分析器进行 O(1) 的热度映射计算
			c.analyzer.Process(ctx, event)
		}
	}
}

// Close 释放 MQ 资源
func (c *RabbitMQConsumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}