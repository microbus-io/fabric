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
// Every fault here targets one entity, so its name is assembled by a builder the consult and the arming
// both call. Each builder allocates, and the caller pays for it before the seams' enabled gate is read, so
// every consult site is wrapped in c.seams.Enabled().

// faultDropAck names the fault that makes ackRequest skip sending the ack for one subscription, yielding a
// unicast 404 ack-timeout or a multicast zero-responder.
func faultDropAck(subName string) string {
	return "dropAck:" + subName
}

// faultDropResponse names the fault that makes handleRequest skip sending the response for one
// subscription, leaving the caller to wait out its full time budget (408).
func faultDropResponse(subName string) string {
	return "dropResponse:" + subName
}

// faultDuplicateResponse names the fault that makes handleResponse re-inject a response from one responder
// hostname (the verified From-Host), exercising the overflow-goroutine push path.
func faultDuplicateResponse(responderHost string) string {
	return "duplicateResponse:" + responderHost
}

// faultJWKSFetchErr names the fault that fails fetchActorKeys for one issuer host (access.token.core /
// bearer.token.core) before the network fetch, without poisoning the 1s cooldown.
func faultJWKSFetchErr(issuerHost string) string {
	return "jwksFetchErr:" + issuerHost
}

// --- Execution checkpoints ---
const (
	checkpointReqRegistered      = "reqRegistered"      // makeRequest, after the await channel is registered, before the request is published
	checkpointAfterAck           = "afterAck"           // the request dispatcher, after the ack is sent, before the handler goroutine is spawned
	checkpointBeforeResponseSend = "beforeResponseSend" // handleRequest, after the handler ran, just before the response is published
)
