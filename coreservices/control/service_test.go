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

package control

import (
	"testing"
)

// The microservice exists only to generate the client API for the :888 control subscriptions and is
// deliberately unstartable. These placeholders keep the boilerplate generator from scaffolding tests
// that would attempt to start it.

func TestControl_Ping(t *testing.T) { // MARKER: Ping
	t.Skip("control.core is unstartable by design")
}

func TestControl_ConfigRefresh(t *testing.T) { // MARKER: ConfigRefresh
	t.Skip("control.core is unstartable by design")
}

func TestControl_Trace(t *testing.T) { // MARKER: Trace
	t.Skip("control.core is unstartable by design")
}

func TestControl_Metrics(t *testing.T) { // MARKER: Metrics
	t.Skip("control.core is unstartable by design")
}

func TestControl_OpenAPI(t *testing.T) { // MARKER: OpenAPI
	t.Skip("control.core is unstartable by design")
}

func TestControl_OnNewSubs(t *testing.T) { // MARKER: OnNewSubs
	t.Skip("control.core is unstartable by design")
}
