package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/microbus-io/fabric/services/metrics/intermediate"

	"github.com/microbus-io/fabric/services/metrics/metricsapi"
)

var (
	_ errors.TracedError
	_ http.Request
	_ metricsapi.Client
)

/*
Service implements the metrics.sys microservice.

The Metrics service is a system microservice that aggregates metrics from other microservices and makes them available for collection.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
	registry                   *prometheus.Registry
	metricsHandler             http.Handler
	gaugeVec                   *prometheus.GaugeVec
	uptimeBase                 time.Time
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	svc.registry = prometheus.NewRegistry()
	//svc.prometheusReg  = registry
	svc.metricsHandler = promhttp.HandlerFor(svc.registry, promhttp.HandlerOpts{})
	svc.uptimeBase = time.Now()
	svc.gaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_uptime_duration_seconds_total",
		Help: "Duration of time since the service was started, in seconds.",
	}, []string{"metrics"})
	err = svc.registry.Register(svc.gaugeVec)
	return
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return // TODO: OnShutdown
}

/*
Collect returns the latest aggregated metrics.
*/
func (svc *Service) Collect(w http.ResponseWriter, r *http.Request) (err error) {
	gauge, err := svc.gaugeVec.GetMetricWithLabelValues("metrics")
	if err != nil {
		return errors.Trace(err)
	}
	gauge.Set(time.Since(svc.uptimeBase).Seconds())
	svc.metricsHandler.ServeHTTP(w, r)
	return nil
}
