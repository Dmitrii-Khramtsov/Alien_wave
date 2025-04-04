// github.com/lonmouth/alien_wave/client/internal/config/config.go
package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GRPCServerAddr  string        // адрес gRPC-сервера
	PostgresDSN     string        // строка подключения к базе данных PostgreSQL
	AnomalyK        float64       // коэффициент для определения аномалий
	TrainSamples    uint          // количество образцов, необходимых для обучения модели
	LogInterval     uint          // интервал логирования
	ShutdownTimeout time.Duration // время ожидания при завершении работы приложения
}

// функция загружает конфигурацию из переменных окружения и возвращает экземпляр Config
func Load() *Config {
	// загрузка переменных из .env файла
	if err := godotenv.Load(); err != nil {
		log.Printf("Notice: .env file not found: %v", err)
	}
	return &Config{
		GRPCServerAddr:  getEnv("GRPC_SERVER_ADDR", "localhost:50051"),
		PostgresDSN:     getEnv("POSTGRES_DSN", "host=localhost user=postgres dbname=anomaly port=5432 sslmode=disable"),
		AnomalyK:        parseFloat(getEnv("ANOMALY_K", "1.5")),
		TrainSamples:    parseUint(getEnv("TRAIN_SAMPLES", "100")),
		LogInterval:     parseUint(getEnv("LOG_INTERVAL", "10")),
		ShutdownTimeout: parseDuration(getEnv("SHUTDOWN_TIMEOUT", "10s")),
	}
}

// получает значение переменной окружения по ключу. Если переменная не установлена, возвращает значение по умолчанию
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// преобразует строку в uint
func parseUint(s string) uint {
	v, _ := strconv.ParseUint(s, 10, 32) // 10 - десятичная система, 32 - uint32
	return uint(v)
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseDuration(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}
