/*
Copyright 2023 Microbus Open Source Foundation and various contributors

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

// Code generated by Microbus. DO NOT EDIT.

/*
Package directory implements the directory.example microservice.

The directory microservice stores personal records in a SQL database.
*/
package directory

import (
	"context"
	"net/http"
	"time"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"

	"github.com/microbus-io/fabric/examples/directory/intermediate"
	"github.com/microbus-io/fabric/examples/directory/directoryapi"
)

var (
	_ context.Context
	_ *http.Request
	_ time.Duration
	_ connector.Service
	_ *errors.TracedError
	_ *directoryapi.Client
)

// The default host name of the microservice is directory.example.
const HostName = "directory.example"

// NewService creates a new directory.example microservice.
func NewService() connector.Service {
	s := &Service{}
	s.Intermediate = intermediate.NewService(s, Version)
	return s
}

// Mock is a mockable version of the directory.example microservice,
// allowing functions, sinks and web handlers to be mocked.
type Mock = intermediate.Mock

// New creates a new mockable version of the microservice.
func NewMock() *Mock {
	return intermediate.NewMock(Version)
}

// Config initializers
var (
	_ int
	// Maria initializes the Maria config property of the microservice
	Maria = intermediate.Maria
)
