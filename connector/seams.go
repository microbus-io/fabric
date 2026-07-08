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

package connector

// --- Fault injection ---
const (
	// Scoped by subscription name (the name passed to Subscribe):
	faultDropAck      = "dropAck"      // ackRequest skips sending the ack -> unicast 404 ack-timeout / multicast zero-responder
	faultDropResponse = "dropResponse" // handleRequest skips sending the response -> caller waits the full time budget (408)

	// Scoped by responder hostname (the verified From-Host on the response):
	faultDuplicateResponse = "duplicateResponse" // handleResponse re-injects the response, exercising the overflow-goroutine push path

	// Scoped by issuer host (access.token.core / bearer.token.core):
	faultJWKSFetchErr = "jwksFetchErr" // fetchActorKeys fails before the network fetch, without poisoning the 1s cooldown
)

// --- Execution checkpoints ---
const (
	checkpointReqRegistered      = "reqRegistered"      // makeRequest, after the await channel is registered, before the request is published
	checkpointAfterAck           = "afterAck"           // the request dispatcher, after the ack is sent, before the handler goroutine is spawned
	checkpointBeforeResponseSend = "beforeResponseSend" // handleRequest, after the handler ran, just before the response is published
)
