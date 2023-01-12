package metrics

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/utils"

	"github.com/microbus-io/fabric/services/control/controlapi"
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
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return
}

/*
Collect returns the latest aggregated metrics.
*/
func (svc *Service) Collect(w http.ResponseWriter, r *http.Request) (err error) {
	var wg sync.WaitGroup
	var delay time.Duration
	var lock sync.Mutex

	host := r.URL.Query().Get("service")
	if host == "" {
		host = "all"
	}
	err = utils.ValidateHostName(host)
	if err != nil {
		return errors.Trace(err)
	}

	// Zip
	var writer io.Writer
	var wcloser io.Closer
	writer = w
	if strings.Index(r.Header.Get("Accept-Encoding"), "gzip") >= 0 {
		zipper, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if zipper != nil {
			w.Header().Set("Content-Encoding", "gzip")
			writer = zipper
			wcloser = zipper // Gzip writer must be closed to flush buffer
		}
	}

	// Timeout
	timeout := pub.Noop()
	secs := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds")
	if secs != "" {
		if s, err := strconv.ParseInt(secs, 10, 0); err == nil {
			timeout = pub.TimeBudget(time.Duration(s) * time.Second)
		}
	}

	w.Header().Set("Content-Type", "text/plain")

	for pingRes := range controlapi.NewClient(svc).ForHost(host).PingServices(r.Context()) {
		pong, err := pingRes.Get()
		if err != nil {
			return errors.Trace(err)
		}
		wg.Add(1)

		go func(s string, delay time.Duration) {
			defer wg.Done()
			time.Sleep(delay) // Stagger requests to avoid all of them coming back at the same time
			host := "https://" + s + ":888/metrics"
			ch := svc.Publish(
				r.Context(),
				pub.GET(host),
				pub.Header("Accept-Encoding", "gzip"),
				timeout)
			for i := range ch {
				res, err := i.Get()
				if err != nil {
					svc.LogWarn(r.Context(), "Fetching metrics", log.Error(err))
					continue
				}

				var reader io.Reader
				var rcloser io.Closer
				reader = res.Body
				rcloser = res.Body
				if res.Header.Get("Content-Encoding") == "gzip" {
					unzipper, err := gzip.NewReader(res.Body)
					if err != nil {
						svc.LogWarn(r.Context(), "Unzippping metrics", log.Error(err))
						continue
					}
					reader = unzipper
					rcloser = unzipper
				}

				lock.Lock()
				_, err = io.Copy(writer, reader)
				lock.Unlock()
				if err != nil {
					svc.LogWarn(r.Context(), "Copying metrics", log.Error(err))
				}
				rcloser.Close()
			}
		}(pong.Service, delay)
		delay += 2 * time.Millisecond
	}

	wg.Wait()
	if wcloser != nil {
		wcloser.Close()
	}

	return nil
}
