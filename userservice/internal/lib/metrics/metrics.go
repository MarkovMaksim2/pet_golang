package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var requestMetrics = promauto.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace:  "userservice",
		Subsystem:  "grpc",
		Name:       "request",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	},
	[]string{"status", "method"},
)

func ObserveRequest(methodName string, status int, duration time.Duration) {
	requestMetrics.WithLabelValues(strconv.Itoa(status), methodName).Observe(duration.Seconds())
}
