// github.com/lonmouth/alien_wave/client/internal/domain/repository.go
package domain

type AnomalyRepository interface {
	Save(a Anomaly) error // метод для сохранения аномалий
}
