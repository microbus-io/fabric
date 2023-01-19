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

package application

// Service is an interface abstraction of a microservice used by the application.
// The connector implements this interface.
type Service interface {
	SetPlane(plane string) error
	SetDeployment(deployment string) error
	Startup() (err error)
	Shutdown() (err error)
	IsStarted() bool
	HostName() string
}
