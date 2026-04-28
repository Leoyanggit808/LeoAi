package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	MySQLDSN       string
	RedisAddr      string
	DeepSeekKey    string
	JWTSecret      string
	RedisPassword  string
	ChromaURL      string
	SiliconFlowKey string // ← 新增
	SiliconFlowURL string // ← 新增
}

func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}

	return &Config{
		MySQLDSN:       viper.GetString("mysql.dsn"),
		RedisAddr:      viper.GetString("redis.addr"),
		DeepSeekKey:    viper.GetString("deepseek.api_key"),
		JWTSecret:      viper.GetString("jwt.secret"),
		RedisPassword:  viper.GetString("redis.password"),
		ChromaURL:      viper.GetString("chroma.url"),
		SiliconFlowKey: viper.GetString("siliconflow.api_key"),
		SiliconFlowURL: viper.GetString("siliconflow.base_url"),
	}
}
