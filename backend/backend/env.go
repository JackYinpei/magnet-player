package backend

import (
	"os"
	"sync"

	"github.com/joho/godotenv"
)

var (
	once sync.Once
)

// LoadEnv loads environment variables from .env file
func LoadEnv() error {
	var err error
	once.Do(func() {
		err = godotenv.Load()
	})
	return err
}

func LoadEnvFrom(path string) {
	once.Do(func() {
		godotenv.Load(path)
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
