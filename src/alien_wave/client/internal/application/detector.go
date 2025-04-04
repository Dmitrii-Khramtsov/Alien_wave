// github.com/lonmouth/alien_wave/client/internal/application/detector.go
package application // прикладной слой
import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/lonmouth/alien_wave/client/internal/domain"
	transmitter "github.com/lonmouth/alien_wave/client/proto"
)

// структура, представляющая детектор аномалий
type Detector struct {
	repo         domain.AnomalyRepository // репозиторий для сохранения аномалий
	stats        *domain.RunningStats     // объект для хранения статистических данных
	checker      *domain.AnomalyChecker   // объект для проверки значений на аномальность
	trainingMode bool                     // флаг, указывающий, находится ли детектор в режиме обучения
	trainSize    uint                     // количество точек данных, необходимых для завершения обучения
	logInterval  uint
	currentUUID  string        // текущий идентификатор сессии
	mu           sync.Mutex    // мьютекс используется для синхронизации доступа к общим ресурсам, таким как stats и currentUUID, чтобы избежать состояния гонки
	shutdownCh   chan struct{} // канал, используемый для управления завершением работы детектора
}

func NewDetector(
	repo domain.AnomalyRepository,
	k float64,
	trainSize uint,
	logInterval uint,
) *Detector {
	return &Detector{
		repo:        repo,
		stats:       domain.NewRunningStats(),
		checker:     domain.NewAnomalyChecker(k),
		trainSize:   trainSize,
		logInterval: logInterval,
		shutdownCh:  make(chan struct{}), // создает канал shutdownCh для управления завершением работы
	}
}

// метод для доступа к каналу shutdown
func (d *Detector) ShutdownChannel() <-chan struct{} {
	return d.shutdownCh
}

// метод Shutdown позволяет корректно завершать работу детектора, что важно для освобождения ресурсов и завершения всех операций
func (d *Detector) Shutdown(ctx context.Context) error {
	select {
	case <-d.shutdownCh:  // если канал shutdownCh закрыт
		return nil          // значит завершение работы уже было инициировано
	case <-ctx.Done():    // если контекст ctx был отменен
		return ctx.Err()    // метод возвращает ошибку, связанную с отменой контекста (ctx.Err())
	default:
		close(d.shutdownCh) // метод закрывает канал shutdownCh, сигнализируя о необходимости завершения работы
		return nil
	}
}

func (d *Detector) saveAnomaly(point *transmitter.Transmission) {
	anomaly := domain.Anomaly{ // создает объект Anomaly с данными из точки данных и текущей статистики
		SessionID:    point.SessionId,
		Frequency:    point.Frequency,
		Timestamp:    time.Unix(point.TimestampUtc, 0),
		ExpectedMean: d.stats.Mean(),
		ExpectedSTD:  d.stats.STD(),
		K:            d.checker.K,
	}

	if err := d.repo.Save(anomaly); err != nil {
		log.Printf("Failed to save anomaly: %v", err)
	}
}

func (d *Detector) reset() {
	d.stats = domain.NewRunningStats()
	d.trainingMode = true
}

func (d *Detector) Process(point *transmitter.Transmission) { // метод для обработки точки данных (обрабатывает данные в реальном времени)
	log.Printf("📥 Received data | Session: %s | Freq: %.2f", 
        point.SessionId, point.Frequency)

	if d.currentUUID != point.SessionId {
		d.reset()                       // сбрасывает статистику
		d.currentUUID = point.SessionId // устанавливает новый currentUUID
	}

	if d.trainingMode {
		d.stats.Update(point.Frequency)     // обновляет статистику
		if d.stats.Count() >= d.trainSize { // проверяет, завершено ли обучение
			d.trainingMode = false
			log.Printf("Training completed. μ=%.2f, σ=%.2f", d.stats.Mean(), d.stats.STD())
		}
	}

	if d.checker.IsAnomaly(point.Frequency, d.stats) { // является ли точка данных аномальной
		d.saveAnomaly(point)
	}
// 	if d.checker.IsAnomaly(point.Frequency, d.stats) || d.stats.Count()%50 == 0 {
// 		anomaly := domain.Anomaly{
// 				SessionID:    "TEST-ANOMALY",
// 				Frequency:    100.0,
// 				Timestamp:    time.Now(),
// 				ExpectedMean: d.stats.Mean(),
// 				ExpectedSTD:  d.stats.STD(),
// 				K:            d.checker.K,
// 		}
// 		d.repo.Save(anomaly)
// }

	if d.stats.Count()%10 == 0 { // логирует статистику каждые 10 точек данных
		log.Printf("Processed: %d, μ=%.2f, σ=%.2f", d.stats.Count(), d.stats.Mean(), d.stats.STD())
	}
}
