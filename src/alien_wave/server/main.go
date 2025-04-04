// Выполнять из папки `alien_wave/server/proto`
// protoc --proto_path=proto \
//        --go_out=proto \
//        --go-grpc_out=proto \
//        --go_opt=paths=source_relative \
//        --go-grpc_opt=paths=source_relative \
//        proto/transmitter.proto
// Указывает, где искать .proto файлы (относительно текущей директории)
// --go_out=proto/gen: Генерировать .pb.go файлы в proto/gen
// --go_opt=paths=source_relative: Сохраняет структуру директорий исходного .proto файла

// github.com/lonmouth/alien_wave/server
package main

import (
	"context"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	transmitter "github.com/lonmouth/alien_wave/server/proto"
	"google.golang.org/grpc"
)

const (
	port          = ":50051"        // Порт, который будет слушать сервер
	sendInterval  = 1 * time.Second // интервал между сообщениями
	serverTimeout = 5 * time.Second // Таймаут для операций сервера
)

type Server struct {
	transmitter.UnimplementedTransmitterServiceServer
}

func (s *Server) StreamData(
	req *transmitter.Empty,
	stream transmitter.TransmitterService_StreamDataServer, // поток для отправки данных
) error {
	// генерация параметров распределения
	// Создаем локальный генератор случайных чисел
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	// математическое ожидание (μ): среднее значение распределения
	mean := r.Float64()*20 - 10 // [-10.0, 10.0]
	// стандартное отклонение (σ): мера разброса значений вокруг среднего
	std := r.Float64()*1.2 + 0.3     // [0.3, 1.5]
	sessionID := uuid.New().String() // генерация уникального ID сессии

	log.Printf("New session: %s (μ=%.2f, σ=%.2f)", sessionID, mean, std)

	// создаём тикер для регулярной отправки сообщений
	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	// бесконечный цикл генерации данных
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			// // генерация значения частоты с нормальным распределением
			frequency := rand.NormFloat64()*std + mean // NormFloat64 генерирует случайное число с плавающей точкой, которое следует нормальному распределению (характеризуется "колоколообразной" кривой) со средним значением 0 и стандартным отклонением 1
			err := stream.Send(&transmitter.Transmission{
				SessionId:    sessionID,
				Frequency:    frequency,
				TimestampUtc: time.Now().Unix(), // (int64) возвращает количество секунд, прошедших с начала эпохи Unix (1 января 1970 года, 00:00:00 UTC) до текущего момента времени
			})
			if err != nil {
				return err
			}
		}
	}
}

func main() {
	// настраиваем перехват сигналов прерывания (Ctrl+C)
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt, // перехватываем сигнал SIGINT
	)
	defer stop() // восстанавливаем стандартное поведение сигналов при выходе

	// cоздаем TCP-листенер для указанного порта
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// создаем экземпляр gRPC-сервера с настройками
	s := grpc.NewServer(
		grpc.ConnectionTimeout(serverTimeout), // таймаут для соединений
	)
	// регистрируем наш сервис на сервере
	transmitter.RegisterTransmitterServiceServer(s, &Server{})

	// запускаем горутину для обработки graceful shutdown
	go func() {
		<-ctx.Done() // ожидаем сигнал завершения
		log.Println("Shutting down server...")
		s.GracefulStop() // плавная остановка сервера
	}()

	// запускаем сервер и логируем статус
	log.Printf("Server is running on port %s...", port)
	// reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
