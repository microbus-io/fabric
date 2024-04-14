/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

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
	secretKey := r.URL.Query().Get("secretKey")
	if secretKey == "" {
		secretKey = r.URL.Query().Get("secretkey")
	}
	if secretKey != svc.SecretKey() {
		return errors.Newc(http.StatusNotFound, "incorrect secret key")
	}

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
	var wCloser io.Closer
	writer = w
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		zipper, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if zipper != nil {
			w.Header().Set("Content-Encoding", "gzip")
			writer = zipper
			wCloser = zipper // Gzip writer must be closed to flush buffer
		}
	}

	// Timeout
	ctx := r.Context()
	timeout := pub.Noop()
	secs := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds")
	if secs != "" {
		if s, err := strconv.Atoi(secs); err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(r.Context(), time.Duration(s)*time.Second)
			defer cancel()
		}
	}

	w.Header().Set("Content-Type", "text/plain")

	var delay time.Duration
	var lock sync.Mutex
	var wg sync.WaitGroup
	for serviceInfo := range controlapi.NewMulticastClient(svc).ForHost(host).PingServices(ctx) {
		wg.Add(1)
		go func(s string, delay time.Duration) {
			defer wg.Done()
			time.Sleep(delay) // Stagger requests to avoid all of them coming back at the same time
			host := "https://" + s + ":888/metrics"
			ch := svc.Publish(
				ctx,
				pub.GET(host),
				pub.Header("Accept-Encoding", "gzip"),
				timeout)
			for i := range ch {
				res, err := i.Get()
				if err != nil {
					svc.LogWarn(ctx, "Fetching metrics", log.Error(err))
					continue
				}

				var reader io.Reader
				var rCloser io.Closer
				reader = res.Body
				rCloser = res.Body
				if res.Header.Get("Content-Encoding") == "gzip" {
					unzipper, err := gzip.NewReader(res.Body)
					if err != nil {
						svc.LogWarn(ctx, "Unzippping metrics", log.Error(err))
						continue
					}
					reader = unzipper
					rCloser = unzipper
				}

				lock.Lock()
				_, err = io.Copy(writer, reader)
				lock.Unlock()
				if err != nil {
					svc.LogWarn(ctx, "Copying metrics", log.Error(err))
				}
				rCloser.Close()
			}
		}(serviceInfo.HostName, delay)
		delay += 2 * time.Millisecond
	}
	wg.Wait()
	if wCloser != nil {
		wCloser.Close()
	}

	return nil
}
