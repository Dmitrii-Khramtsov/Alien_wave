// github.com/lonmouth/alien_wave/client/internal/infrastructure/postgres/repository.go
package postgres

import (
	"log"
	"time"

	"github.com/lonmouth/alien_wave/client/internal/domain"
	"gorm.io/gorm"
)

type AnomalyModel struct {
	ID           uint      `gorm:"primarykey"`
	SessionID    string    `gorm:"column:session_id"`
	Frequency    float64   `gorm:"column:frequency"`
	Timestamp    time.Time `gorm:"column:timestamp"`
	ExpectedMean float64   `gorm:"column:expected_mean"`
	ExpectedSTD  float64   `gorm:"column:expected_std"`
	K            float64   `gorm:"column:k"`
}

// Указываем явное имя таблицы
func (AnomalyModel) TableName() string {
	return "anomalies"
}

type PostgresRepository struct {
	db *gorm.DB // GORM — ORM (Object-Relational Mapping) для Go
}

func NewRepository(db *gorm.DB) domain.AnomalyRepository {
	if err := db.AutoMigrate(&AnomalyModel{}); err != nil { // выполняет автоматическую миграцию схемы базы данных для модели AnomalyModel
		log.Fatalf("Migration failed: %v", err)
}
log.Println("Database migration completed successfully")
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Save(a domain.Anomaly) error {
	model := AnomalyModel{
		SessionID:    a.SessionID,
		Frequency:    a.Frequency,
		Timestamp:    a.Timestamp,
		ExpectedMean: a.ExpectedMean,
		ExpectedSTD:  a.ExpectedSTD,
		K:            a.K,
	}

	if err := r.db.Create(&model).Error; err != nil {
		log.Printf("❌ Failed to save anomaly: %v", err)
		return err
	}

	log.Printf("✅ Anomaly saved | Session: %s | Freq: %.2f | Mean: %.2f | STD: %.2f",
		a.SessionID, a.Frequency, a.ExpectedMean, a.ExpectedSTD)
	return nil
}

// psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS dmitrii;"
// psql -h localhost -U postgres -c "CREATE DATABASE dmitrii OWNER dmitrii;"