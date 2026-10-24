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

package flowsvc

import (
	"context"

	"github.com/microbus-io/dwarf/workflow"
)

// Service is a minimal stand-in for a microservice that hosts workflow graphs.
type Service struct{}

// NewService creates a new Service.
func NewService() *Service {
	return &Service{}
}

// Approval builds a graph that fans out, fans in, loops back with a goto, and handles an error.
func (svc *Service) Approval(ctx context.Context) (graph *workflow.Graph, err error) {
	graph = workflow.NewGraph("Approval")
	graph.SetURL("Submit", "https://flow.svc:428/submit")
	graph.SetURL("CheckA", "https://flow.svc:428/check-a")
	graph.SetURL("CheckB", "https://flow.svc:428/check-b")
	graph.SetURL("Decide", "https://flow.svc:428/decide")
	graph.SetURL("Recover", "https://flow.svc:428/recover")
	graph.AddTransitionFanOut("Submit", "CheckA", "CheckB")
	graph.SetFanIn("Decide")
	graph.AddTransition("CheckA", "Decide")
	graph.AddTransition("CheckB", "Decide")
	graph.SetReducer("failures", workflow.ReducerAdd)
	graph.AddTransitionWhen("Decide", workflow.End, "approved")
	graph.AddTransitionGoto("Decide", "Submit")
	graph.AddTransitionOnError("Submit", "Recover")
	graph.AddTransition("Recover", workflow.End)
	return graph, nil
}

// Review builds a linear graph that routes with a switch.
func (svc *Service) Review(ctx context.Context) (graph *workflow.Graph, err error) {
	graph = workflow.NewGraph("Review")
	graph.SetURL("Score", "https://flow.svc:428/score")
	graph.SetURL("Accept", "https://flow.svc:428/accept")
	graph.SetURL("Reject", "https://flow.svc:428/reject")
	graph.AddTransitionSwitch("Score", "Accept", "score >= 600")
	graph.AddTransitionSwitch("Score", "Reject", "true")
	graph.AddTransitionChain("Accept", workflow.End)
	graph.AddTransitionChain("Reject", workflow.End)
	return graph, nil
}

// Helper has a graph builder's results but takes no context, so it is not a workflow.
func (svc *Service) Helper() (graph *workflow.Graph, err error) {
	return workflow.NewGraph("Helper"), nil
}
