package banksupport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/errors"

	"github.com/microbus-io/fabric/coreservices/bearertoken/bearertokenapi"
	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
	"github.com/microbus-io/fabric/coreservices/llm/llmapi"
	"github.com/microbus-io/fabric/exampleservices/banksupport/banksupportapi"
	"github.com/microbus-io/fabric/frame"
)

var (
	_ context.Context
	_ http.Request
	_ errors.TracedError
	_ = banksupportapi.Hostname
)

const authTokenCookieName = "Authorization"

/*
Service implements the banksupport.example agent. It authenticates a bank customer, then answers their
natural-language banking questions by running an LLM tool-calling loop over their own balance and transactions,
which it reads scoped to the customer's actor claim.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	// accounts is the in-memory demo store, keyed by username. It is built once in OnStartup and read-only
	// thereafter, so it needs no lock (see populate.go).
	accounts map[string]*demoAccount
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	svc.populateDemoData()
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	return nil
}

/*
Login renders the login screen and, on valid credentials, mints a customer bearer token and redirects to the demo.
*/
func (svc *Service) Login(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: Login
	ctx := r.Context()
	err = r.ParseForm()
	if err != nil {
		return errors.Trace(err)
	}
	u := strings.TrimSpace(strings.ToLower(r.FormValue("u")))
	submitted := r.FormValue("l") != ""

	cust, known := demoCustomers[u]
	ok := submitted && known
	if ok {
		signedJWT, err := bearertokenapi.NewClient(svc).Mint(ctx, map[string]any{
			"sub":   u,
			"name":  cust.holder,
			"roles": []string{"customer"},
		})
		if err != nil {
			return errors.Trace(err)
		}
		token, _ := jwt.Parse(signedJWT, nil)
		exp := time.Unix(int64(token.Claims.(jwt.MapClaims)["exp"].(float64)), 0)
		cookie := &http.Cookie{
			Name:     authTokenCookieName,
			Value:    signedJWT,
			MaxAge:   int(time.Until(exp).Round(time.Second).Seconds()),
			HttpOnly: true,
			Secure:   r.TLS != nil,
			Path:     "/",
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, svc.ExternalizeURL(ctx, banksupportapi.Demo.URL()), http.StatusTemporaryRedirect)
		return nil
	}

	data := struct {
		U      string
		Denied bool
	}{
		U:      u,
		Denied: submitted && !ok,
	}
	w.Header().Set("Content-Type", "text/html")
	err = svc.WriteResTemplate(w, "login.html", data)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
Logout clears the customer session cookie and returns to the login screen.
*/
func (svc *Service) Logout(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: Logout
	ctx := r.Context()
	cookie := &http.Cookie{
		Name:     authTokenCookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		Path:     "/",
	}
	w.Header().Add("Set-Cookie", cookie.String())
	http.Redirect(w, r, svc.ExternalizeURL(ctx, banksupportapi.Login.URL()), http.StatusTemporaryRedirect)
	return nil
}

// accountFor looks up the signed-in customer's account in the in-memory store by the username derived from the
// verified actor claim, never a caller-supplied identifier.
func (svc *Service) accountFor(ctx context.Context) (*demoAccount, error) {
	var actor banksupportapi.Actor
	_, err := frame.Of(ctx).ParseActor(&actor)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if actor.Subject == "" {
		return nil, errors.New("no customer identity in token", http.StatusForbidden)
	}
	acct := svc.accounts[actor.Subject]
	if acct == nil {
		return nil, errors.New("no account for customer '%s'", actor.Subject, http.StatusNotFound)
	}
	return acct, nil
}

/*
Balance returns the signed-in customer's current balance. It derives the customer from the actor claim, never
from an argument, and is exposed to the LLM as a tool.
*/
func (svc *Service) Balance(ctx context.Context) (balanceCents int, holder string, err error) { // MARKER: Balance
	acct, err := svc.accountFor(ctx)
	if err != nil {
		return 0, "", errors.Trace(err)
	}
	return acct.balanceCents, acct.holder, nil
}

/*
Transactions returns the signed-in customer's transactions in a date range. It derives the customer from the
actor claim, never from an argument, and is exposed to the LLM as a tool for spend analysis over time.
*/
func (svc *Service) Transactions(ctx context.Context, fromDate string, toDate string) (transactions []banksupportapi.TxnView, err error) { // MARKER: Transactions
	acct, err := svc.accountFor(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var from, to time.Time
	if fromDate != "" {
		from, err = time.ParseInLocation("2006-01-02", fromDate, time.Local)
		if err != nil {
			return nil, errors.New("invalid fromDate '%s', expected YYYY-MM-DD", fromDate, http.StatusBadRequest)
		}
	}
	if toDate != "" {
		to, err = time.ParseInLocation("2006-01-02", toDate, time.Local)
		if err != nil {
			return nil, errors.New("invalid toDate '%s', expected YYYY-MM-DD", toDate, http.StatusBadRequest)
		}
	}
	// The store holds transactions most-recent-first; filter to the [from, to) range (inclusive from, exclusive to).
	for i, view := range acct.txns {
		at := acct.txnDates[i]
		if !from.IsZero() && at.Before(from) {
			continue
		}
		if !to.IsZero() && !at.Before(to) {
			continue
		}
		transactions = append(transactions, view)
	}
	return transactions, nil
}

// parseVerdict extracts the JSON verdict object from the model's final message, tolerating code fences and
// surrounding prose. On failure it falls back to treating the whole reply as advice.
func parseVerdict(answer string) banksupportapi.SupportOut {
	trimmed := strings.TrimSpace(answer)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	start := strings.IndexByte(trimmed, '{')
	end := strings.LastIndexByte(trimmed, '}')
	if start >= 0 && end > start {
		var out banksupportapi.SupportOut
		if err := json.Unmarshal([]byte(trimmed[start:end+1]), &out); err == nil && out.Advice != "" {
			if out.Risk < 0 {
				out.Risk = 0
			}
			if out.Risk > 10 {
				out.Risk = 10
			}
			return out
		}
	}
	return banksupportapi.SupportOut{Advice: strings.TrimSpace(answer)}
}

/*
Support is the durable workflow that answers a customer's banking question. Its RunSupport task runs the LLM
tool-calling loop (ChatLoop) with the Balance and Transactions endpoints as tools and produces a structured
verdict: advice text, whether the card should be blocked, and a 0-10 risk score. It is a workflow rather than a
synchronous call because the multi-turn agent conversation routinely exceeds a single request's time budget;
each turn is its own durable, independently-budgeted foreman step.
*/
func (svc *Service) Support(ctx context.Context) (graph *workflow.Graph, err error) { // MARKER: Support
	graph = workflow.NewGraph("Support")
	graph.SetURL("RunSupport", banksupportapi.RunSupport.URL())
	graph.AddTransition("RunSupport", workflow.End)
	return graph, nil
}

/*
RunSupport is the single task of the Support workflow. It runs llm.core's ChatLoop as a subgraph with the
Balance and Transactions tools and parses the model's final message into the structured verdict.
*/
func (svc *Service) RunSupport(ctx context.Context, flow *workflow.Flow, query string) (advice string, blockCard bool, risk int, err error) { // MARKER: RunSupport
	today := time.Now().Format("2006-01-02")
	systemPrompt := fmt.Sprintf(`You are a bank's customer-support agent. Today's date is %s.
Answer the customer's question about their own account. Use the balance tool for the current balance and the
transactions tool (with a fromDate/toDate range in YYYY-MM-DD) to analyze spending over time; sum and group the
returned rows yourself. Amounts are in cents; negative amounts are debits.
When you are done, reply with ONLY a JSON object and nothing else, of the form:
{"advice": "<your natural-language answer>", "blockCard": <true|false>, "risk": <integer 0-10>}
Set blockCard to true and a high risk only when the account is deeply overdrawn or shows clearly abnormal spending.`, today)

	items := []llmapi.Item{
		llmapi.NewMessage("system", systemPrompt).AsItem(),
		llmapi.NewMessage("user", query).AsItem(),
	}
	tools := []string{
		banksupportapi.Balance.URL(),
		banksupportapi.Transactions.URL(),
	}
	result, _, yield, err := llmapi.NewSubgraph(flow).ChatLoop(ctx, llmapi.ProviderAny, llmapi.ModelDefault, items, tools, nil)
	if yield {
		return "", false, 0, nil
	}
	if err != nil {
		return "", false, 0, errors.Trace(err)
	}
	answer := llmapi.LastAssistantMessage(result)
	out := parseVerdict(answer)
	return out.Advice, out.BlockCard, out.Risk, nil
}

/*
Demo is the signed-in support console where a customer asks natural-language banking questions.
*/
func (svc *Service) Demo(w http.ResponseWriter, r *http.Request) (err error) { // MARKER: Demo
	ctx := r.Context()
	err = r.ParseForm()
	if err != nil {
		return errors.Trace(err, http.StatusBadRequest)
	}
	acct, err := svc.accountFor(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	data := struct {
		Holder         string
		BalanceDollars string
		Query          string
		FlowKey        string
		Error          string
	}{
		Holder:         acct.holder,
		BalanceDollars: fmt.Sprintf("%.2f", float64(acct.balanceCents)/100),
		Query:          r.FormValue("query"),
	}

	if r.Method == "POST" && strings.TrimSpace(data.Query) != "" {
		flowKey, cerr := foremanapi.NewClient(svc).Create(ctx, banksupportapi.Support.URL(), banksupportapi.SupportIn{Query: data.Query}, nil)
		if cerr != nil {
			data.Error = fmt.Sprintf("%+v", cerr)
		} else {
			data.FlowKey = flowKey
		}
	}

	w.Header().Set("Content-Type", "text/html")
	err = svc.WriteResTemplate(w, "demo.html", data)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
DemoStatus long-polls a launched Support workflow via the foreman's Poll and returns the structured verdict
when the flow has stopped, or a running status so the demo page can re-poll immediately with no client delay.
*/
func (svc *Service) DemoStatus(ctx context.Context, flowKey string) (status string, advice string, blockCard bool, risk int, errorMsg string, err error) { // MARKER: DemoStatus
	if strings.TrimSpace(flowKey) == "" {
		return "", "", false, 0, "", errors.New("flowKey is required", http.StatusBadRequest)
	}
	var out banksupportapi.SupportOut
	outcome, err := foremanapi.NewClient(svc).PollAndParse(ctx, flowKey, &out)
	if err != nil {
		return "", "", false, 0, "", errors.Trace(err)
	}
	if !outcome.Stopped() {
		return workflow.StatusRunning, "", false, 0, "", nil
	}
	if outcome.Status == workflow.StatusCompleted {
		return outcome.Status, out.Advice, out.BlockCard, out.Risk, "", nil
	}
	return outcome.Status, "", false, 0, outcome.Error, nil
}
