package config

import (
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Redis    RedisConfig    `mapstructure:"redis"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
}

type AppConfig struct {
	LogLevel string `mapstructure:"log_level"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"` 
}

type MySQLConfig struct {
	DSN string `mapstructure:"dsn"` 
}

type RabbitMQConfig struct {
	URL       string `mapstructure:"url"`        
	QueueName string `mapstructure:"queue_name"`
}
func LoadConfig() *Config {
    viper.AddConfigPath("./configs")        // 如果在根目录运行
    viper.AddConfigPath("../configs")       // 如果在 heat-service 目录运行
    viper.AddConfigPath("../../configs")    // 如果在 heat-service/cmd 目录运行
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