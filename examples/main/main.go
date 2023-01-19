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

package main

import (
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/examples/calculator"
	"github.com/microbus-io/fabric/examples/directory"
	"github.com/microbus-io/fabric/examples/eventsink"
	"github.com/microbus-io/fabric/examples/eventsource"
	"github.com/microbus-io/fabric/examples/hello"
	"github.com/microbus-io/fabric/examples/messaging"
	"github.com/microbus-io/fabric/services/configurator"
	"github.com/microbus-io/fabric/services/httpingress"
	"github.com/microbus-io/fabric/services/metrics"
)

/*
main runs the example microservices.
*/
func main() {
	app := application.New(
		configurator.NewService(),
		httpingress.NewService(),
		metrics.NewService(),
		hello.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		messaging.NewService(),
		calculator.NewService(),
		eventsource.NewService(),
		eventsink.NewService(),
		directory.NewService(),
	)
	app.Run()
}
