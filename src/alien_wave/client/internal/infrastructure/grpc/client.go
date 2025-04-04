// github.com/lonmouth/alien_wave/client/internal/infrastructure/grpc/client.go
package grpc

import (
	"context"

	transmitter "github.com/lonmouth/alien_wave/client/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	client transmitter.TransmitterServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials())) // небезопасного соединения (без TLS)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn, // соединение с gRPC-сервером
		client: transmitter.NewTransmitterServiceClient(conn), // клиент для вызова методов gRPC-сервиса
	}, nil
}

// метод для установления потокового соединения с gRPC-сервером
func (c *Client) Stream() (transmitter.TransmitterService_StreamDataClient, error) {
	return c.client.StreamData(context.Background(), &transmitter.Empty{}) // возвращает клиент для работы с потоком данных.
}

// метод для закрытия соединения с gRPC-сервером
func (c *Client) Close() error {
	return c.conn.Close()
}
