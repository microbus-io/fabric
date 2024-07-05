/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package openapiportal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/microbus-io/fabric/errors"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"

	"github.com/microbus-io/fabric/coreservices/control/controlapi"
	"github.com/microbus-io/fabric/coreservices/openapiportal/intermediate"
	"github.com/microbus-io/fabric/coreservices/openapiportal/openapiportalapi"
)

var (
	_ context.Context
	_ *http.Request
	_ time.Duration
	_ *errors.TracedError
	_ *openapiportalapi.Client
)

/*
Service implements the openapiportal.core microservice.

The OpenAPI microservice lists links to the OpenAPI endpoint of all microservices that provide one
on the requested port.
*/
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
List displays links to the OpenAPI endpoint of all microservices that provide one on the request's port.
*/
func (svc *Service) List(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()

	type info struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Host        string `json:"-"`
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
				if errors.StatusCode(err) == http.StatusNotFound {
					// No openapi.json for this service
					return
				}
				lock.Lock()
				lastErr = errors.Trace(err)
				lock.Unlock()
				return
			}
			if res.StatusCode == http.StatusNotFound {
				// No openapi.json for this service
				return
			}
			oapiDoc := struct {
				Info info `json:"info"`
			}{}
			err = json.NewDecoder(res.Body).Decode(&oapiDoc)
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
		}(serviceInfo.Hostname, delay)
		delay += time.Millisecond
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
