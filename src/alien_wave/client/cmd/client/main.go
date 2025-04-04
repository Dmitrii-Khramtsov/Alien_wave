// github.com/lonmouth/alien_wave/client/cmd/client/main.go

// ├── cmd
// │   └── client
// │       └── main.go
// ├── proto
// │   └── transmitter.proto
// └── internal
//     ├── config
//     │   └── config.go
//     ├── domain
//     │   ├── models.go
//     │   └── repository.go
//     ├── application
//     │   └── detector.go
//     └── infrastructure
//         ├── grpc
//         |   └── client.go
//         └── postgres
//             └── repository.go

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lonmouth/alien_wave/client/internal/application"
	"github.com/lonmouth/alien_wave/client/internal/config"
	"github.com/lonmouth/alien_wave/client/internal/infrastructure/grpc"
  pg "github.com/lonmouth/alien_wave/client/internal/infrastructure/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Инициализация конфигурации
	cfg := config.Load()

	// Настройка системы
	system := setupSystem(cfg)
	defer teardownSystem(system)

	// Запуск обработки данных
	dataProcessor := startDataProcessing(system)
	defer dataProcessor.Stop()

	// Ожидание сигналов завершения
	waitForShutdownSignal(system, cfg)
}

// SystemComponents содержит все системные компоненты
type SystemComponents struct {
	DB         *gorm.DB
	GRPCClient *grpc.Client
	Detector   *application.Detector
	Cancel     context.CancelFunc
}

// setupSystem инициализирует все системные компоненты
func setupSystem(cfg *config.Config) *SystemComponents {
	// Инициализация контекста с возможностью отмены
	_, cancel := context.WithCancel(context.Background())

	// Подключение к БД
	db := initDatabase(cfg.PostgresDSN)

	// Инициализация репозитория
	repo := pg.NewRepository(db)

	// Подключение к gRPC серверу
	gClient := initGRPCClient(cfg.GRPCServerAddr)

	// Создание детектора аномалий
	detector := application.NewDetector(
		repo,
		cfg.AnomalyK,
		cfg.TrainSamples,
		cfg.LogInterval,
	)

	return &SystemComponents{
		DB:         db,
		GRPCClient: gClient,
		Detector:   detector,
		Cancel:     cancel,
	}
}

// teardownSystem корректно освобождает ресурсы
func teardownSystem(s *SystemComponents) {
	// Закрытие gRPC соединения
	if err := s.GRPCClient.Close(); err != nil {
		log.Printf("GRPC close error: %v", err)
	}

	// Закрытие подключения к БД
	if sqlDB, err := s.DB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("DB close error: %v", err)
		}
	}
}

// initDatabase инициализирует подключение к PostgreSQL
func initDatabase(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("DB connection error:", err)
	}

	// Проверка подключения
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("DB access error:", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatal("DB ping failed:", err)
	}

	return db
}

// initGRPCClient создает gRPC клиент
func initGRPCClient(addr string) *grpc.Client {
	client, err := grpc.NewClient(addr)
	if err != nil {
		log.Fatal("gRPC connection error:", err)
	}
	return client
}

// DataProcessor управляет обработкой данных
type DataProcessor struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// startDataProcessing запускает обработку потока данных
func startDataProcessing(s *SystemComponents) *DataProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	// Получение потока данных
	stream, err := s.GRPCClient.Stream()
	if err != nil {
		log.Fatal("Stream error:", err)
	}

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				data, err := stream.Recv()
				if err != nil {
					log.Printf("Stream error: %v", err)
					return
				}
				s.Detector.Process(data)
			}
		}
	}()

	return &DataProcessor{
		ctx:    ctx,
		cancel: cancel,
		done:   done,
	}
}

// Stop останавливает обработку данных
func (dp *DataProcessor) Stop() {
	dp.cancel()
	<-dp.done
}

// waitForShutdownSignal обрабатывает сигналы завершения
func waitForShutdownSignal(s *SystemComponents, cfg *config.Config) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Ожидание сигнала или ошибки
	select {
	case sig := <-sigCh:
		log.Printf("Received signal: %v", sig)
	case <-s.Detector.ShutdownChannel():
		log.Println("Detector initiated shutdown")
	}

	// Инициируем завершение работы
	s.Cancel()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := s.Detector.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	} else {
		log.Println("Shutdown completed successfully")
	}
}

// SELECT COUNT(*) FROM anomalies;

// SELECT 
//     session_id,
//     ROUND(frequency::numeric, 2) AS frequency,
//     timestamp,
//     ROUND(expected_mean::numeric, 2) AS mean,
//     ROUND(expected_std::numeric, 2) AS std,
//     k
// FROM anomalies
// ORDER BY timestamp DESC
// LIMIT 10;

// TRUNCATE TABLE anomalies; // очистка таблицы

// go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
// go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
