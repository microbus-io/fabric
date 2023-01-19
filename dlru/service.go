/*
Copyright 2023 Microbus LLC and various contributors

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

package dlru

import (
	"context"
	"net/http"

	"github.com/microbus-io/fabric/log"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
)

// Service is an interface abstraction of a microservice used by the distributed cache.
// The connector implements this interface.
type Service interface {
	ID() string
	HostName() string
	Subscribe(path string, handler sub.HTTPHandler, options ...sub.Option) error
	Unsubscribe(path string) error
	Publish(ctx context.Context, options ...pub.Option) <-chan *pub.Response
	Request(ctx context.Context, options ...pub.Option) (*http.Response, error)
	LogInfo(ctx context.Context, msg string, fields ...log.Field)
	LogDebug(ctx context.Context, msg string, fields ...log.Field)
}
