package config

import (
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	RedisAddr  string `mapstructure:"redis_addr"`
	RabbitMQ   string `mapstructure:"rabbitmq_url"`
	QueueName  string `mapstructure:"queue_name"` // 新增
	LogLevel   string `mapstructure:"log_level"`
	MySQLDSN   string `mapstructure:"mysql_dsn"`
}

func LoadConfig() *Config {
	viper.AddConfigPath("../configs") // 相对路径要根据运行入口调整
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}
	return &cfg
}