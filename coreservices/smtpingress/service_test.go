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

package smtpingress

import (
	"testing"
)

// The microservice starts a real SMTP daemon on startup, which cannot run in the test environment.
// These placeholders keep the boilerplate generator from scaffolding tests that boot the daemon.

func TestSMTPIngress_OnChangedPort(t *testing.T) { // MARKER: Port
	t.Skip("requires an SMTP daemon")
}

func TestSMTPIngress_OnChangedEnabled(t *testing.T) { // MARKER: Enabled
	t.Skip("requires an SMTP daemon")
}

func TestSMTPIngress_OnChangedMaxSize(t *testing.T) { // MARKER: MaxSize
	t.Skip("requires an SMTP daemon")
}

func TestSMTPIngress_OnChangedMaxClients(t *testing.T) { // MARKER: MaxClients
	t.Skip("requires an SMTP daemon")
}

func TestSMTPIngress_OnChangedWorkers(t *testing.T) { // MARKER: Workers
	t.Skip("requires an SMTP daemon")
}

func TestSMTPIngress_OnIncomingEmail(t *testing.T) { // MARKER: OnIncomingEmail
	t.Skip("requires an SMTP daemon")
}
