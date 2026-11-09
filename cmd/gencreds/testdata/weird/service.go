/*
Copyright (c) 2023-2026 Microbus LLC and various contributors

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

package weird

import (
	"context"

	"github.com/microbus-io/errors"

	"github.com/microbus-io/fabric/cmd/gencreds/testdata/weird/weirdapi"
)

var (
	_ context.Context
	_ errors.TracedError
	_ weirdapi.Client
)

/*
Service implements weird, a gencreds fixture exercising every route shape.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove
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
Plain is a baseline function on the safe trust segment.
*/
func (svc *Service) Plain(ctx context.Context) (result string, err error) { // MARKER: Plain
	return "", nil
}

/*
PathArg accepts a path argument.
*/
func (svc *Service) PathArg(ctx context.Context, id string) (err error) { // MARKER: PathArg
	return nil
}

/*
GreedyArg accepts a greedy tail path argument.
*/
func (svc *Service) GreedyArg(ctx context.Context, tail string) (err error) { // MARKER: GreedyArg
	return nil
}

/*
PeriodInPath has a period inside its path segment.
*/
func (svc *Service) PeriodInPath(ctx context.Context) (err error) { // MARKER: PeriodInPath
	return nil
}

/*
AnyMethod accepts any HTTP method.
*/
func (svc *Service) AnyMethod(ctx context.Context) (err error) { // MARKER: AnyMethod
	return nil
}

/*
InternalPort is on the :444 internal port.
*/
func (svc *Service) InternalPort(ctx context.Context) (err error) { // MARKER: InternalPort
	return nil
}

/*
TrustRoot is the trust-root :666 endpoint.
*/
func (svc *Service) TrustRoot(ctx context.Context, cmd string) (err error) { // MARKER: TrustRoot
	return nil
}

/*
SlashHostRoot has route "//root".
*/
func (svc *Service) SlashHostRoot(ctx context.Context) (err error) { // MARKER: SlashHostRoot
	return nil
}

/*
SlashHostPort has route "//alt.host:0/alt-path".
*/
func (svc *Service) SlashHostPort(ctx context.Context) (err error) { // MARKER: SlashHostPort
	return nil
}

/*
SlashHostPathArg has route "//alt.host/items/{id}".
*/
func (svc *Service) SlashHostPathArg(ctx context.Context, id string) (err error) { // MARKER: SlashHostPathArg
	return nil
}

/*
SpecialHostRoute has route "//my$.xml/lookup" - URL-special character in the hostname segment.
*/
func (svc *Service) SpecialHostRoute(ctx context.Context) (err error) { // MARKER: SpecialHostRoute
	return nil
}

/*
UpperCasePath has route ":443/UPPERCASE.xml" - uppercase path segment with a period.
*/
func (svc *Service) UpperCasePath(ctx context.Context) (err error) { // MARKER: UpperCasePath
	return nil
}
