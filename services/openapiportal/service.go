/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package openapiportal

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"gopkg.in/yaml.v3"

	"github.com/microbus-io/fabric/services/control/controlapi"
	"github.com/microbus-io/fabric/services/openapiportal/intermediate"
	"github.com/microbus-io/fabric/services/openapiportal/openapiportalapi"
)

var (
	_ context.Context
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *openapiportalapi.Client
)

/*
Service implements the openapiportal.sys microservice.

The OpenAPI microservice lists links to the OpenAPI endpoint of all microservices that provide one
on the requested port.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	// Create the subscriptions dynamically for each of the configured ports
	ports := svc.Ports()
	for _, port := range strings.Split(ports, ",") {
		port = strings.TrimSpace(port)
		err = svc.Subscribe("//openapi:"+port, svc.listServices)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

// listServices displays links to the OpenAPI endpoint of all microservices that provide one
// on the requested port
func (svc *Service) listServices(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	type info struct {
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
		Host        string
	}
	infos := []*info{}

	var delay time.Duration
	var lock sync.Mutex
	var wg sync.WaitGroup
	var lastErr error
	for serviceInfo := range controlapi.NewMulticastClient(svc).ForHost("all").PingServices(ctx) {
		wg.Add(1)
		go func(s string, delay time.Duration) {
			defer wg.Done()
			time.Sleep(delay) // Stagger requests to avoid all of them coming back at the same time
			u := fmt.Sprintf("https://%s:%s/openapi.json", s, r.URL.Port())
			res, err := svc.Request(ctx, pub.GET(u))
			if err != nil {
				lock.Lock()
				lastErr = errors.Trace(err)
				lock.Unlock()
			}
			if res.StatusCode == http.StatusNotFound {
				// No openapi.json for this service
				return
			}
			oapiDoc := struct {
				Info info `yaml:"info"`
			}{}
			err = yaml.NewDecoder(res.Body).Decode(&oapiDoc)
			if err != nil {
				lock.Lock()
				lastErr = errors.Trace(err)
				lock.Unlock()
				return
			}
			oapiDoc.Info.Host = frame.Of(res).FromHost()
			lock.Lock()
			infos = append(infos, &oapiDoc.Info)
			lock.Unlock()
		}(serviceInfo.HostName, delay)
		delay += 2 * time.Millisecond
	}
	wg.Wait()
	if lastErr != nil {
		return errors.Trace(lastErr)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Host < infos[j].Host
	})
	data := struct {
		Port  string
		Infos []*info
	}{
		Port:  r.URL.Port(),
		Infos: infos,
	}
	output, err := svc.ExecuteResTemplate("list.html", data)
	if err != nil {
		return errors.Trace(err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = w.Write([]byte(output))
	return errors.Trace(err)
}
