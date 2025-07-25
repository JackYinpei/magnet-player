package backend

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

var (
	once sync.Once
)

// LoadEnv loads environment variables from .env file
// Deprecated: Use config.Load() instead for better configuration management
func LoadEnv() error {
	var err error
	once.Do(func() {
		err = godotenv.Load()
		if err != nil {
			fmt.Println("No .env file found, using system environment variables")
			err = nil // 不将缺少.env文件视为错误
		}
	})
	return err
}

// LoadEnvFrom loads environment variables from specified file path
// Deprecated: Use config.Load() instead for better configuration management
func LoadEnvFrom(path string) {
	once.Do(func() {
		if err := godotenv.Load(path); err != nil {
			fmt.Printf("Warning: Could not load .env file from %s: %v\n", path, err)
		}
	})
}

// GetEnvWithDefault gets an environment variable or returns the default value if not found
func GetEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetEnv gets an environment variable
func GetEnv(key string) string {
	return os.Getenv(key)
}

// MustGetEnv gets an environment variable and panics if not found
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("Required environment variable not set: " + key)
	}
	return value
}
