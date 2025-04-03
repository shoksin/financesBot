package metrics

import (
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shoksin/financesBot/internal/clients/tg"
	"github.com/shoksin/financesBot/internal/logger"
	"github.com/shoksin/financesBot/internal/models/messages"
)

type TgHandler interface {
	RunFunc(tgUpdate tgbotapi.Update, c *tg.Client, msgModel *messages.Model)
}

// Метрики.
var (
	InFlightRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tg",
		Subsystem: "messages",
		Name:      "messages_total", // Общее количество сообщений.
	})
	SummaryResponseTime = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace: "tg",
		Subsystem: "messages",
		Name:      "summary_response_time_seconds", // Время обработки сообщений.
		Objectives: map[float64]float64{
			0.5:  0.1,
			0.9:  0.01,
			0.99: 0.001,
		},
	})
	HistogramResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "tg",
			Subsystem: "messages",
			Name:      "histogram_response_time_seconds",
			Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2},
		},
		[]string{"cmd"},
	)
)

var labels []string

func init() {
	labels = []string{"start", "cat", "curr", "report", "add_tbl", "add_cat", "add_rec", "choice_currency", "set_limit"}

	http.Handle("/", promhttp.Handler())

	logger.Info("Start metrics service")
	go func() {
		// Для просмотра значений метрик по адресу http://127.0.0.1:8080/
		err := http.ListenAndServe("0.0.0.0:8080", nil)
		if err != nil {
			logger.Error("Metrics public error", "err", err)
		}
	}()
}

// MetricsMiddleware Функция сбора метрик.
func MetricsMiddleware(next tg.HandlerFunc) tg.HandlerFunc {
	handler := tg.HandlerFunc(func(tgUpdate tgbotapi.Update, c *tg.Client, msgModel *messages.Model) {
		startTime := time.Now()

		next.RunFunc(tgUpdate, c, msgModel)

		duration := time.Since(startTime)

		// Сохранение метрик продолжительности обработки.
		SummaryResponseTime.Observe(duration.Seconds())

		// Определение команды для сохранения в метрике.
		cmd := "none"
		msg := ""
		if tgUpdate.Message == nil {
			if tgUpdate.CallbackQuery != nil {
				msg = tgUpdate.CallbackQuery.Data
			}
		} else {
			msg = tgUpdate.Message.Text
		}
		if msg != "" {
			for _, lbl := range labels {
				if strings.Contains(msg, "/"+lbl) {
					cmd = lbl
					break
				}
			}
		}
		HistogramResponseTime.WithLabelValues(cmd).Observe(duration.Seconds())
	})

	InFlightRequests.Dec()

	return handler
}
