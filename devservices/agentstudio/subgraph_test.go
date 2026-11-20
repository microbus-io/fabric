package agentstudio

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/foreman"
	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
	"github.com/microbus-io/fabric/devservices/agentstudio/agentstudioapi"
)

func TestAgentStudio_SubgraphChildActions(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Parent runs P, which calls Child as a subgraph, then W, which interrupts. So the child completes while its
	// root is still open.
	const host = "subgraph.actions.agentstudio"
	graph := func(name, route string, build func() *workflow.Graph) func(c *connector.Connector) {
		return func(c *connector.Connector) {
			c.Subscribe(name, func(w http.ResponseWriter, r *http.Request) error {
				w.Header().Set("Content-Type", "application/json")
				return errors.Trace(json.NewEncoder(w).Encode(struct {
					Graph *workflow.Graph `json:"graph"`
				}{Graph: build()}))
			}, sub.At("GET", route), sub.Workflow(struct{}{}, struct{}{}))
		}
	}
	task := func(name, route string, body func(f *workflow.Flow) error) func(c *connector.Connector) {
		return func(c *connector.Connector) {
			c.Subscribe(name, func(w http.ResponseWriter, r *http.Request) error {
				var flow workflow.Flow
				err := json.NewDecoder(r.Body).Decode(workflow.RequestCarrier(&flow))
				if err != nil {
					return errors.Trace(err)
				}
				err = body(&flow)
				if err != nil {
					return err // No trace
				}
				w.Header().Set("Content-Type", "application/json")
				return errors.Trace(json.NewEncoder(w).Encode(workflow.ResultCarrier(&flow)))
			}, sub.At("POST", route), sub.Task(struct{}{}, struct{}{}))
		}
	}
	wf := connector.New(host).Init(func(c *connector.Connector) error {
		graph("ParentGraph", ":428/parent", func() *workflow.Graph {
			g := workflow.NewGraph("Parent")
			g.SetURL("P", host+":428/p")
			g.SetURL("W", host+":428/w")
			g.AddTransition("P", "W")
			g.AddTransition("W", workflow.End)
			return g
		})(c)
		graph("ChildGraph", ":428/child", func() *workflow.Graph {
			g := workflow.NewGraph("Child")
			g.SetURL("K", host+":428/k")
			g.AddTransition("K", workflow.End)
			return g
		})(c)
		task("P", ":428/p", func(f *workflow.Flow) error {
			var out map[string]any
			_, err := f.Subgraph(host+":428/child", nil, &out)
			return err
		})(c)
		task("W", ":428/w", func(f *workflow.Flow) error {
			var resume map[string]any
			f.Interrupt(map[string]any{"need": "input"}, &resume)
			return nil
		})(c)
		task("K", ":428/k", func(f *workflow.Flow) error {
			f.SetInt("k", 1)
			return nil
		})(c)
		return nil
	})

	tester := connector.New("tester.client")
	client := agentstudioapi.NewClient(tester)
	fm := foremanapi.NewClient(tester)

	app := application.New()
	app.Add(
		NewService(),
		foreman.NewService(),
		wf,
		tester,
	)
	app.RunInTest(t)

	assert := testarossa.For(t)
	rootKey, err := fm.Create(ctx, host+":428/parent", nil, nil)
	if !assert.NoError(err) {
		return
	}
	out, err := fm.Await(ctx, rootKey)
	if !assert.NoError(err) || !assert.Equal(workflow.StatusInterrupted, out.Status) {
		return
	}
	rootSteps, err := fm.History(ctx, rootKey)
	if !assert.NoError(err) || !assert.True(len(rootSteps) > 0) {
		return
	}
	childKey := rootSteps[0].SubgraphKey
	childSteps, err := fm.History(ctx, childKey)
	if !assert.NoError(err) || !assert.True(len(childSteps) > 0) {
		return
	}
	childStepKey := childSteps[0].StepKey

	flowPage := func(t *testing.T, flowKey string) string {
		res, err := client.FlowDetail(ctx, "GET", "?flowKey="+url.QueryEscape(flowKey), nil)
		if !testarossa.For(t).NoError(err) {
			return ""
		}
		body, _ := io.ReadAll(res.Body)
		return string(body)
	}
	stepPage := func(t *testing.T, stepKey string, flowKey string) string {
		res, err := client.StepDetail(ctx, "GET", "?stepKey="+url.QueryEscape(stepKey)+"&flow="+url.QueryEscape(flowKey), nil)
		if !testarossa.For(t).NoError(err) {
			return ""
		}
		body, _ := io.ReadAll(res.Body)
		return string(body)
	}

	t.Run("child_of_an_open_root", func(t *testing.T) {
		assert := testarossa.For(t)

		// The child completed, but Resume, Terminate and Continue take the root's key, and Fork needs the root closed
		page := flowPage(t, childKey)
		assert.Contains(page, "Root flow")
		assert.Contains(page, `name="continue" disabled`)
		assert.NotContains(page, ">Resume<")
		assert.NotContains(page, "stop_circle")
		assert.NotContains(page, "fork_right")

		assert.NotContains(stepPage(t, childStepKey, childKey), "fork_right")
	})

	t.Run("open_root", func(t *testing.T) {
		assert := testarossa.For(t)

		page := flowPage(t, rootKey)
		assert.NotContains(page, "Root flow")
		assert.Contains(page, ">Resume<")
		assert.Contains(page, "stop_circle")
		assert.NotContains(page, "fork_right")

		// A completed step of an open flow cannot be forked yet
		assert.NotContains(stepPage(t, rootSteps[0].StepKey, rootKey), "fork_right")
	})

	t.Run("closed_root", func(t *testing.T) {
		assert := testarossa.For(t)

		err := fm.Resume(ctx, rootKey, map[string]any{"answer": 1})
		if !assert.NoError(err) {
			return
		}
		out, err := fm.Await(ctx, rootKey)
		if !assert.NoError(err) || !assert.Equal(workflow.StatusCompleted, out.Status) {
			return
		}

		// Once the root is closed, any step of the tree can be forked
		assert.Contains(flowPage(t, rootKey), "fork_right")
		page := flowPage(t, childKey)
		assert.Contains(page, "fork_right")
		assert.Contains(page, `name="continue" disabled`)
		assert.Contains(stepPage(t, childStepKey, childKey), "fork_right")
		assert.Contains(stepPage(t, rootSteps[0].StepKey, rootKey), "fork_right")
	})
}
