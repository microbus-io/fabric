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

package foremanapi

import (
	"context"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"
)

// RunAndParse creates a flow, starts it, blocks until it stops, then unmarshals the outcome's State into result.
// Returns the full workflow.FlowOutcome alongside any unmarshal error; inspect outcome.Status / outcome.Error to learn
// whether the workflow succeeded.
func (_c Client) RunAndParse(ctx context.Context, workflowURL string, initialState any, opts *workflow.FlowOptions, result any) (outcome *workflow.FlowOutcome, err error) {
	outcome, err = _c.Run(ctx, workflowURL, initialState, opts)
	if err != nil {
		return outcome, errors.Trace(err)
	}
	if result != nil && outcome != nil {
		err = outcome.State.Parse(result)
		if err != nil {
			return outcome, errors.Trace(err)
		}
	}
	return outcome, nil
}

// AwaitAndParse blocks until the flow stops, then unmarshals the outcome's State into result.
func (_c Client) AwaitAndParse(ctx context.Context, flowKey string, result any) (outcome *workflow.FlowOutcome, err error) {
	outcome, err = _c.Await(ctx, flowKey)
	if err != nil {
		return outcome, errors.Trace(err)
	}
	if result != nil && outcome != nil {
		err = outcome.State.Parse(result)
		if err != nil {
			return outcome, errors.Trace(err)
		}
	}
	return outcome, nil
}

// PollAndParse is Poll plus unmarshaling a stopped flow's State into result. A still-running outcome
// (outcome.Stopped() is false) leaves result untouched, so the caller re-polls.
func (_c Client) PollAndParse(ctx context.Context, flowKey string, result any) (outcome *workflow.FlowOutcome, err error) {
	outcome, err = _c.Poll(ctx, flowKey)
	if err != nil {
		return outcome, errors.Trace(err)
	}
	if result != nil && outcome != nil && outcome.Stopped() {
		err = outcome.State.Parse(result)
		if err != nil {
			return outcome, errors.Trace(err)
		}
	}
	return outcome, nil
}

// SnapshotAndParse returns the current outcome and unmarshals its State into result.
func (_c Client) SnapshotAndParse(ctx context.Context, flowKey string, result any) (outcome *workflow.FlowOutcome, err error) {
	outcome, err = _c.Snapshot(ctx, flowKey)
	if err != nil {
		return outcome, errors.Trace(err)
	}
	if result != nil && outcome != nil {
		err = outcome.State.Parse(result)
		if err != nil {
			return outcome, errors.Trace(err)
		}
	}
	return outcome, nil
}
