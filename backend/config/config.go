package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 应用配置结构
type Config struct {
	// 服务器配置
	Server ServerConfig `json:"server"`
	
	// 数据库配置
	Database DatabaseConfig `json:"database"`
	
	// API配置
	API APIConfig `json:"api"`
	
	// Torrent配置
	Torrent TorrentConfig `json:"torrent"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
	Env  string `json:"env"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path            string `json:"path"`
	MaxConnections  int    `json:"max_connections"`
	ConnMaxLifetime int    `json:"conn_max_lifetime"` // 秒
}

// APIConfig API相关配置
type APIConfig struct {
	JinaAPIKey  string `json:"-"` // 不序列化到JSON
	TMDBAPIKey  string `json:"-"` // 不序列化到JSON
	OpenAIAPIKey string `json:"-"` // 不序列化到JSON
}

// TorrentConfig Torrent相关配置
type TorrentConfig struct {
	DataDir               string `json:"data_dir"`
	MaxConnections        int    `json:"max_connections"`
	EnableDHT             bool   `json:"enable_dht"`
	EnablePEX             bool   `json:"enable_pex"`
	SeedEnabled           bool   `json:"seed_enabled"`
	MetadataTimeoutSec    int    `json:"metadata_timeout_sec"`
}

// Load 加载配置
func Load() (*Config, error) {
	// 尝试加载.env文件，如果不存在也不报错
	_ = godotenv.Load()
	
	config := &Config{
		Server: ServerConfig{
			Host: getEnvWithDefault("SERVER_HOST", "localhost"),
			Port: getEnvWithDefault("SERVER_PORT", "8080"),
			Env:  getEnvWithDefault("ENV", "development"),
		},
		Database: DatabaseConfig{
			Path:            getEnvWithDefault("DB_PATH", "./data/torrents.db"),
			MaxConnections:  getEnvIntWithDefault("DB_MAX_CONNECTIONS", 10),
			ConnMaxLifetime: getEnvIntWithDefault("DB_CONN_MAX_LIFETIME", 3600),
		},
		API: APIConfig{
			JinaAPIKey:   getEnvWithDefault("JINA_API_KEY", ""),
			TMDBAPIKey:   getEnvWithDefault("TMDB_API_KEY", ""),
			OpenAIAPIKey: getEnvWithDefault("OPENAI_API_KEY", ""),
		},
		Torrent: TorrentConfig{
			DataDir:            getEnvWithDefault("TORRENT_DATA_DIR", "./data"),
			MaxConnections:     getEnvIntWithDefault("TORRENT_MAX_CONNECTIONS", 50),
			EnableDHT:          getEnvBoolWithDefault("TORRENT_ENABLE_DHT", true),
			EnablePEX:          getEnvBoolWithDefault("TORRENT_ENABLE_PEX", true),
			SeedEnabled:        getEnvBoolWithDefault("TORRENT_SEED_ENABLED", true),
			MetadataTimeoutSec: getEnvIntWithDefault("TORRENT_METADATA_TIMEOUT", 30),
		},
	}
	
	// 验证必要的配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}
	
	return config, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("服务器端口不能为空")
	}
	
	if c.Database.Path == "" {
		return fmt.Errorf("数据库路径不能为空")
	}
	
	if c.Torrent.DataDir == "" {
		return fmt.Errorf("Torrent数据目录不能为空")
	}
	
	return nil
}

// IsProduction 判断是否为生产环境
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// IsDevelopment 判断是否为开发环境
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// GetServerAddress 获取服务器地址
func (c *Config) GetServerAddress() string {
	return c.Server.Host + ":" + c.Server.Port
}

// 辅助函数

// getEnvWithDefault 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntWithDefault 获取整数环境变量，如果不存在或转换失败则返回默认值
func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBoolWithDefault 获取布尔环境变量，如果不存在或转换失败则返回默认值
func getEnvBoolWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}