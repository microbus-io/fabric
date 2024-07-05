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

/*
Package controlapi implements the public API of the control.core microservice,
including clients and data structures.

This microservice is created for the sake of generating the client API for the :888 control subscriptions.
The microservice itself does nothing and should not be included in applications.
*/
package controlapi

import (
	"context"
	"fmt"

	"github.com/microbus-io/fabric/frame"
)

// ServiceInfo is a descriptor of the microservice that answers the ping.
type ServiceInfo struct {
	Hostname string
	Version  int
	ID       string
}

// PingServices performs a ping and returns service info for microservices on the network.
// Results are deduped on a per-service basis.
func (_c *MulticastClient) PingServices(ctx context.Context) <-chan *ServiceInfo {
	ch := _c.Ping(ctx)
	filtered := make(chan *ServiceInfo, cap(ch))
	go func() {
		seen := map[string]bool{}
		for pingRes := range ch {
			if pingRes.err != nil {
				continue
			}
			frame := frame.Of(pingRes.HTTPResponse)
			info := &ServiceInfo{
				Hostname: frame.FromHost(),
			}
			if seen[info.Hostname] {
				continue
			}
			seen[info.Hostname] = true
			filtered <- info
		}
		close(filtered)
	}()
	return filtered
}

// PingVersions performs a ping and returns service info for microservice versions on the network.
// Results are deduped on a per-version basis.
func (_c *MulticastClient) PingVersions(ctx context.Context) <-chan *ServiceInfo {
	ch := _c.Ping(ctx)
	filtered := make(chan *ServiceInfo, cap(ch))
	go func() {
		seen := map[string]bool{}
		for pingRes := range ch {
			if pingRes.err != nil {
				continue
			}
			frame := frame.Of(pingRes.HTTPResponse)
			info := &ServiceInfo{
				Hostname: frame.FromHost(),
				Version:  frame.FromVersion(),
			}
			key := fmt.Sprintf("%s:%d", info.Hostname, info.Version)
			if seen[key] {
				continue
			}
			seen[key] = true
			filtered <- info
		}
		close(filtered)
	}()
	return filtered
}

// PingInstances performs a ping and returns service info for all instances on the network.
func (_c *MulticastClient) PingInstances(ctx context.Context) <-chan *ServiceInfo {
	ch := _c.Ping(ctx)
	filtered := make(chan *ServiceInfo, cap(ch))
	go func() {
		for pingRes := range ch {
			if pingRes.err != nil {
				continue
			}
			frame := frame.Of(pingRes.HTTPResponse)
			info := &ServiceInfo{
				Hostname: frame.FromHost(),
				Version:  frame.FromVersion(),
				ID:       frame.FromID(),
			}
			filtered <- info
		}
		close(filtered)
	}()
	return filtered
}
