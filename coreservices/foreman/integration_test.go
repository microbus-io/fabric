/*
Copyright (c) 2026 Microbus LLC and various contributors

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

package foreman

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
)

// This integration test exercises the foreman end-to-end over the Microbus bus against a real downstream
// microservice that serves workflow graphs and task endpoints. Unlike the engine's in-process fixtures
// (which inject a TestProxy Host), this drives the actual adapter seam: host.go's LoadGraph GETs a graph
// over the bus and decodes the {"graph": …} envelope, and ExecuteTask marshals the *workflow.Flow to JSON,
// POSTs it to the task, and unmarshals the returned flow - so the downstream endpoints below do their own
// (un)marshaling, mirroring what a generated workflow microservice's intermediate.go does.

// graphHandler turns a graph builder into a bus endpoint that serves the {"graph": …} envelope the
// foreman's LoadGraph expects.
func graphHandler(build func() *workflow.Graph) sub.HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		g := build()
		if err := g.Validate(); err != nil {
			return errors.Trace(err)
		}
		w.Header().Set("Content-Type", "application/json")
		return errors.Trace(json.NewEncoder(w).Encode(struct {
			Graph *workflow.Graph `json:"graph"`
		}{Graph: g}))
	}
}

// taskHandler turns a task body into a bus endpoint: decode the posted flow carrier, run the body (which
// reads/writes state and may arm control signals), and encode the flow back for the engine to read.
func taskHandler(body func(f *workflow.Flow) error) sub.HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var flow workflow.Flow
		if err := json.NewDecoder(r.Body).Decode(&flow); err != nil {
			return errors.Trace(err)
		}
		if err := body(&flow); err != nil {
			return err // no trace: preserve the status code so the foreman can classify it
		}
		w.Header().Set("Content-Type", "application/json")
		return errors.Trace(json.NewEncoder(w).Encode(&flow))
	}
}

// subscribeGraph registers a workflow graph endpoint (GET on :428). The sub.Workflow feature option is
// required for the subscription to activate; its zero-struct args are only for OpenAPI reflection, which
// the foreman does not use (it GETs the graph by URL directly).
func subscribeGraph(c *connector.Connector, name, route string, build func() *workflow.Graph) {
	c.Subscribe(name, graphHandler(build), sub.At("GET", route), sub.Workflow(struct{}{}, struct{}{}))
}

// subscribeTask registers a workflow task endpoint (POST on :428). sub.Task is the required feature option.
func subscribeTask(c *connector.Connector, name, route string, body func(f *workflow.Flow) error) {
	c.Subscribe(name, taskHandler(body), sub.At("POST", route), sub.Task(struct{}{}, struct{}{}))
}

// linearGraph is A -> B -> C -> END, used by several subtests as a trivial happy path.
func linearGraph(host string) *workflow.Graph {
	g := workflow.NewGraph("Linear")
	g.SetEndpoint("A", host+":428/a")
	g.SetEndpoint("B", host+":428/b")
	g.SetEndpoint("C", host+":428/c")
	g.AddTransition("A", "B")
	g.AddTransition("B", "C")
	g.AddTransition("C", workflow.END)
	return g
}

// registerLinear wires the linear graph + its A/B/C tasks (n := ((n+1)*10)+5) onto a connector.
func registerLinear(c *connector.Connector, host string) {
	subscribeGraph(c, "LinearGraph", ":428/linear", func() *workflow.Graph { return linearGraph(host) })
	subscribeTask(c, "A", ":428/a", func(f *workflow.Flow) error { f.SetInt("n", f.GetInt("n")+1); return nil })
	subscribeTask(c, "B", ":428/b", func(f *workflow.Flow) error { f.SetInt("n", f.GetInt("n")*10); return nil })
	subscribeTask(c, "C", ":428/c", func(f *workflow.Flow) error { f.SetInt("n", f.GetInt("n")+5); return nil })
}

func TestForemanIntegration(t *testing.T) {
	ctx := context.Background()
	const host = "inttest.flows"

	var retryAttempts atomic.Int32

	wf := connector.New(host).Init(func(c *connector.Connector) error {
		registerLinear(c, host)

		// Subgraph: Parent P -> END, where P calls flow.Subgraph(Child) and adopts its output. Child K -> END.
		subscribeGraph(c, "ParentGraph", ":428/parent", func() *workflow.Graph {
			g := workflow.NewGraph("Parent")
			g.SetEndpoint("P", host+":428/p")
			g.AddTransition("P", workflow.END)
			return g
		})
		subscribeGraph(c, "ChildGraph", ":428/child", func() *workflow.Graph {
			g := workflow.NewGraph("Child")
			g.SetEndpoint("K", host+":428/k")
			g.AddTransition("K", workflow.END)
			return g
		})
		subscribeTask(c, "P", ":428/p", func(f *workflow.Flow) error {
			var out map[string]any
			yield, err := f.Subgraph(host+":428/child", map[string]any{"v": f.GetInt("x")}, &out)
			if yield || err != nil {
				return err
			}
			w, _ := out["w"].(float64)
			f.SetInt("result", int(w))
			return nil
		})
		subscribeTask(c, "K", ":428/k", func(f *workflow.Flow) error { f.SetInt("w", f.GetInt("v")*2); return nil })

		// Interrupt: I -> END. First pass parks via flow.Interrupt; on resume it records the answer.
		subscribeGraph(c, "InterruptGraph", ":428/interrupt", func() *workflow.Graph {
			g := workflow.NewGraph("Interrupt")
			g.SetEndpoint("I", host+":428/i")
			g.AddTransition("I", workflow.END)
			return g
		})
		subscribeTask(c, "I", ":428/i", func(f *workflow.Flow) error {
			var resume map[string]any
			yield, _ := f.Interrupt(map[string]any{"need": "info"}, &resume)
			if yield {
				return nil
			}
			ans, _ := resume["answer"].(float64)
			f.SetInt("answer", int(ans))
			return nil
		})

		// Cancel-cascade: CancelParent's CP task subgraphs CancelChild, whose CI task parks via flow.Interrupt.
		// Running the parent parks the whole tree at interrupted (the interrupt surfaces up the surgraph chain);
		// cancelling the parent must then cascade cancellation into the live subgraph child. flow.Interrupt is
		// the deterministic pause that lets the test cancel a flow with a genuinely in-flight subgraph.
		subscribeGraph(c, "CancelParentGraph", ":428/cancel-parent", func() *workflow.Graph {
			g := workflow.NewGraph("CancelParent")
			g.SetEndpoint("CP", host+":428/cp")
			g.AddTransition("CP", workflow.END)
			return g
		})
		subscribeGraph(c, "CancelChildGraph", ":428/cancel-child", func() *workflow.Graph {
			g := workflow.NewGraph("CancelChild")
			g.SetEndpoint("CI", host+":428/ci")
			g.AddTransition("CI", workflow.END)
			return g
		})
		subscribeTask(c, "CP", ":428/cp", func(f *workflow.Flow) error {
			var out map[string]any
			yield, err := f.Subgraph(host+":428/cancel-child", nil, &out)
			if yield || err != nil {
				return err
			}
			return nil
		})
		subscribeTask(c, "CI", ":428/ci", func(f *workflow.Flow) error {
			var resume map[string]any
			f.Interrupt(map[string]any{"need": "info"}, &resume)
			return nil
		})

		// Cancel a running flow: Sl -> SlDone -> END, where Sl arms an hour-long flow.Sleep. The sleep delays
		// the successor step's not_before (SlDone stays pending for an hour), so the flow sits running rather
		// than completing - a sleep is only dropped when its step transitions straight to END. Because Create
		// auto-runs, the flow is running the moment Create returns; the hour-long delay guarantees it cannot
		// reach a terminal state during the test, so Cancel is deterministic without a pause/sync handshake.
		subscribeGraph(c, "SleepGraph", ":428/sleep", func() *workflow.Graph {
			g := workflow.NewGraph("Sleep")
			g.SetEndpoint("Sl", host+":428/sl")
			g.SetEndpoint("SlDone", host+":428/sl-done")
			g.AddTransition("Sl", "SlDone")
			g.AddTransition("SlDone", workflow.END)
			return g
		})
		subscribeTask(c, "Sl", ":428/sl", func(f *workflow.Flow) error {
			f.Sleep(time.Hour)
			return nil
		})
		subscribeTask(c, "SlDone", ":428/sl-done", func(f *workflow.Flow) error { return nil })

		// Task-owned retry: Rt -> END. The foreman no longer classifies errors into engine dispositions, so a
		// task that wants to recover from a transient failure arms flow.Retry itself. The task "fails" on its
		// first dispatch by arming a retry, then succeeds on re-dispatch.
		subscribeGraph(c, "RetryGraph", ":428/retry", func() *workflow.Graph {
			g := workflow.NewGraph("Retry")
			g.SetEndpoint("Rt", host+":428/rt")
			g.AddTransition("Rt", workflow.END)
			return g
		})
		subscribeTask(c, "Rt", ":428/rt", func(f *workflow.Flow) error {
			if retryAttempts.Add(1) == 1 {
				f.Sleep(10 * time.Millisecond)
				f.Retry(0, 1.0, 0, 0)
				return nil
			}
			f.SetBool("served", true)
			return nil
		})

		// Ack-timeout retry: Gone -> END, where Gone's endpoint is on a host no microservice serves, so the
		// dispatch ack-times-out (404). The foreman arms the retry itself and re-dispatches with backoff until
		// the give-up horizon, then fails the step.
		subscribeGraph(c, "AbsentGraph", ":428/absent", func() *workflow.Graph {
			g := workflow.NewGraph("Absent")
			g.SetEndpoint("Gone", "absent.host:428/gone")
			g.AddTransition("Gone", workflow.END)
			return g
		})
		return nil
	})

	tester := connector.New("inttest.client")

	fmn := NewService()
	// The ack-timeout retry horizon is the task's time budget, and the re-probe interval is budget/N - so a
	// small TimeBudget tightens the whole cadence. 1s budget => ~125ms probes, give-up in ~1s, no test seam.
	fmn.SetTimeBudget(time.Second)

	app := application.New()
	app.Add(fmn, wf, tester)
	app.RunInTest(t)

	client := foremanapi.NewClient(tester)

	t.Run("linear_happy_path", func(t *testing.T) {
		assert := testarossa.For(t)
		out, err := client.Run(ctx, host+":428/linear", map[string]any{"n": 1}, nil)
		if !assert.NoError(err) {
			return
		}
		assert.Equal(workflow.StatusCompleted, out.Status)
		assert.Equal(25, out.State.GetInt("n")) // ((1+1)*10)+5
	})

	t.Run("subgraph_call_and_return", func(t *testing.T) {
		assert := testarossa.For(t)
		out, err := client.Run(ctx, host+":428/parent", map[string]any{"x": 3}, nil)
		if !assert.NoError(err) {
			return
		}
		assert.Equal(workflow.StatusCompleted, out.Status)
		assert.Equal(6, out.State.GetInt("result")) // child computed w = v*2 = 6, parent adopted it
	})

	t.Run("interrupt_then_resume", func(t *testing.T) {
		assert := testarossa.For(t)
		// foreman.Run does not return the flow key; use Create (which auto-runs) + Await so we have the key
		// to Resume.
		flowKey, err := client.Create(ctx, host+":428/interrupt", nil, nil)
		if !assert.NoError(err) {
			return
		}
		out, err := client.Await(ctx, flowKey)
		if !assert.NoError(err) {
			return
		}
		// Await returns when the flow stops, and an interrupt is a stop: interrupted with the payload.
		assert.Equal(workflow.StatusInterrupted, out.Status)
		assert.Equal("info", out.InterruptPayload.GetString("need"))

		if !assert.NoError(client.Resume(ctx, flowKey, map[string]any{"answer": 42})) {
			return
		}
		final, err := client.Await(ctx, flowKey)
		if !assert.NoError(err) {
			return
		}
		assert.Equal(workflow.StatusCompleted, final.Status)
		assert.Equal(42, final.State.GetInt("answer"))
	})

	t.Run("cancel_running_flow", func(t *testing.T) {
		assert := testarossa.For(t)
		// The entry step sleeps an hour, so the auto-running flow stays running and Cancel is deterministic.
		flowKey, err := client.Create(ctx, host+":428/sleep", nil, nil)
		if !assert.NoError(err) {
			return
		}
		if !assert.NoError(client.Cancel(ctx, flowKey, "operator")) {
			return
		}
		out, err := client.Await(ctx, flowKey)
		if !assert.NoError(err) {
			return
		}
		assert.Equal(workflow.StatusCancelled, out.Status)
		assert.Equal("operator", out.CancelReason)
	})

	t.Run("cancel_cascades_into_subgraph", func(t *testing.T) {
		assert := testarossa.For(t)
		flowKey, err := client.Create(ctx, host+":428/cancel-parent", nil, nil)
		if !assert.NoError(err) {
			return
		}
		// The interrupt raised inside the subgraph child surfaces up and parks the parent.
		out, err := client.Await(ctx, flowKey)
		if !assert.NoError(err) {
			return
		}
		if !assert.Equal(workflow.StatusInterrupted, out.Status) {
			return
		}
		// Cancelling the parent cascades into the still-live subgraph child.
		if !assert.NoError(client.Cancel(ctx, flowKey, "operator")) {
			return
		}
		out, err = client.Await(ctx, flowKey)
		if !assert.NoError(err) {
			return
		}
		assert.Equal(workflow.StatusCancelled, out.Status)
	})

	t.Run("task_owned_retry_recovers", func(t *testing.T) {
		assert := testarossa.For(t)
		out, err := client.Run(ctx, host+":428/retry", nil, nil)
		if !assert.NoError(err) {
			return
		}
		// The first dispatch armed flow.Retry itself; the re-dispatch succeeded, so the flow recovers to
		// completed. The engine did no classification - retry is entirely the task's doing.
		assert.Equal(workflow.StatusCompleted, out.Status)
		assert.True(out.State.GetBool("served"))
		assert.True(retryAttempts.Load() >= 2, "expected the task's own retry to re-dispatch")
	})

	t.Run("ack_timeout_retries_then_gives_up", func(t *testing.T) {
		assert := testarossa.For(t)
		flowKey, err := client.Create(ctx, host+":428/absent", map[string]any{}, nil)
		if !assert.NoError(err) {
			return
		}
		outcome, err := client.Await(ctx, flowKey)
		if !assert.NoError(err) {
			return
		}
		// No microservice hosts the task, so every dispatch ack-times-out; the foreman retries with backoff
		// until the give-up horizon and then fails the step (rather than looping forever).
		assert.Equal(workflow.StatusFailed, outcome.Status)
		// The horizon is wide enough that at least one retry was armed before giving up: the step's attempt
		// counter advanced past the initial dispatch.
		steps, err := client.History(ctx, flowKey)
		if !assert.NoError(err) {
			return
		}
		maxAttempt := 0
		for _, s := range steps {
			if s.Attempt > maxAttempt {
				maxAttempt = s.Attempt
			}
		}
		assert.True(maxAttempt >= 1, "expected at least one ack-timeout retry, got max attempt %d", maxAttempt)
	})
}

// TestForemanIntegration_CrossReplica runs a flow against two foreman replicas sharing one plane (hence one
// set of shard databases). Work created via one replica's client is dispatched and completed across the
// pair, with the replicas coordinating entirely through the shared databases - no message passing between
// them at any point.
func TestForemanIntegration_CrossReplica(t *testing.T) {
	ctx := context.Background()
	const host = "inttest.xr"

	wf := connector.New(host).Init(func(c *connector.Connector) error {
		registerLinear(c, host)
		return nil
	})
	tester := connector.New("inttest.xr.client")

	app := application.New()
	app.Add(NewService(), NewService(), wf, tester) // two foreman.core replicas
	app.RunInTest(t)

	client := foremanapi.NewClient(tester)
	out, err := client.Run(ctx, host+":428/linear", map[string]any{"n": 1}, nil)
	if !testarossa.For(t).NoError(err) {
		return
	}
	assert := testarossa.For(t)
	assert.Equal(workflow.StatusCompleted, out.Status)
	assert.Equal(25, out.State.GetInt("n"))
}
