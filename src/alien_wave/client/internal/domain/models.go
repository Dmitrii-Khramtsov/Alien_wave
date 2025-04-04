// github.com/lonmouth/alien_wave/client/internal/domain/models.go
package domain

import (
	"math"
	"time"
)

// структура, которая хранит статистические данные для вычисления среднего значения и стандартного отклонения на лету
type RunningStats struct {
	count uint    // колличество добавленных значений
	mean  float64 // текущее среднее значение
	m2    float64 // сумма квадратов отклонений - для вычисления стандартного отклонения
}

// функция-конструктор, которая создает и возвращает новый экземпляр RunningStats
func NewRunningStats() *RunningStats {
	return &RunningStats{}
}

func (s *RunningStats) Update(x float64) {
	s.count++
	delta := x - s.mean                // разница между новым значением и текущим средним
	s.mean += delta / float64(s.count) // обновляется с учетом нового значения
	delta2 := x - s.mean               // обновленная разница после изменения среднего
	s.m2 += delta * delta2             // обновляется для учета нового значения в вычислении стандартного отклонения
}

func (s *RunningStats) Mean() float64 { return s.mean } // метод, возвращающий текущее среднее значение

func (s *RunningStats) STD() float64 {
	if s.count < 2 {
		return 0
	}
	return math.Sqrt(s.m2 / float64(s.count-1)) // метод, возвращающий текущее стандартное отклонение
}

func (s *RunningStats) Count() uint { return s.count } // метод, возвращающий количество добавленных значений

type AnomalyChecker struct { // структура для проверки значений на наличие аномалий
	K float64
}

func NewAnomalyChecker(k float64) *AnomalyChecker { // функция-конструктор, создающая новый экземпляр AnomalyChecker с заданным коэффициентом k
	return &AnomalyChecker{K: k}
}

func (c *AnomalyChecker) IsAnomaly(value float64, stats *RunningStats) bool { // сравнивает отклонение значения от среднего с порогом, определяемым как K стандартных отклонений
	return math.Abs(value-stats.Mean()) > c.K*stats.STD()
}

type Anomaly struct {
	SessionID    string    // уникальный идентификатор сессии
	Frequency    float64   // значение частоты, которое считается аномальным
	Timestamp    time.Time // время, когда была обнаружена аномалия
	ExpectedMean float64   // ожидаемое среднее значение
	ExpectedSTD  float64   // ожидаемое стандартное отклонение
	K            float64   // коэффициент использованный для обнаружения аномалии
}
