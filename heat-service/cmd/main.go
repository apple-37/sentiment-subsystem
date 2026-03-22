package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"sentiment/heat-service/internal/analyzer"
	"sentiment/heat-service/internal/consumer"
	"sentiment/heat-service/internal/storage"
	"sentiment/pkg/config"
	"sentiment/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadConfig()
	logger.InitLogger(cfg.LogLevel)
	defer logger.Log.Sync()

	// 1. 初始化 Redis
	redisStore := storage.NewRedisStore(cfg.RedisAddr)

	// 2. 初始化 MySQL (用于后续持久化归档)
	// 注意：确保 config.yaml 中配置了 mysql_dsn
	mysqlStore, err := storage.NewMySQLStore(cfg.MySQLDSN)
	if err != nil {
		logger.Log.Fatal("Failed to connect MySQL", zap.Error(err))
	}
	defer mysqlStore.Close()

	// 3. 初始化核心分析器
	// (如果未来写了 HTTP 接口，可以直接通过 heatAnalyzer 调用 mysqlStore 进行落盘)
	heatAnalyzer := analyzer.NewHeatAnalyzer(redisStore)
	
	// 4. 初始化 MQ 消费者
	mqConsumer, err := consumer.NewRabbitMQConsumer(cfg.RabbitMQ, cfg.QueueName, heatAnalyzer)
	if err != nil {
		logger.Log.Fatal("Failed to init MQ consumer", zap.Error(err))
	}

	// 5. 启动后台消费 (非阻塞)
	ctx, cancel := context.WithCancel(context.Background())
	go mqConsumer.Start(ctx)

	logger.Log.Info("🔥 Heat-Service is running and waiting for messages...")

	// 6. 监听系统中断信号，实现优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Heat-Service...")
	cancel() // 通知消费者停止接收
	mqConsumer.Close()
}