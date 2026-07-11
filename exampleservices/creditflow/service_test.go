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

package creditflow

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/fabric/application"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/fabric/pub"
	"github.com/microbus-io/fabric/sub"
	"github.com/microbus-io/testarossa"

	"github.com/microbus-io/fabric/coreservices/foreman"
	"github.com/microbus-io/fabric/coreservices/foreman/foremanapi"
	"github.com/microbus-io/fabric/exampleservices/creditflow/creditflowapi"
)

var (
	_ context.Context
	_ io.Reader
	_ *http.Request
	_ *testing.T
	_ jwt.MapClaims
	_ application.Application
	_ connector.Connector
	_ frame.Frame
	_ pub.Option
	_ sub.Option
	_ *workflow.Flow
	_ testarossa.Asserter
	_ creditflowapi.Client
)

// MARKER: SubmitCreditApplication

// MARKER: VerifyCredit

// MARKER: VerifyEmployment

// MARKER: VerifySSN

// MARKER: VerifyAddress

// MARKER: VerifyPhoneNumber

// MARKER: IdentityDecision

// MARKER: HandleCreditError

// MARKER: Decision

// MARKER: RequestMoreInfo

// MARKER: ReviewCredit

// MARKER: CreditApproval

// Mock the workflow behavior

// Graph endpoint should now return a valid graph

// MARKER: IdentityVerification

// Mock the workflow behavior

// Graph endpoint should now return a valid graph

// MARKER: Demo

func TestCreditFlow_SubmitCreditApplication(t *testing.T) { // MARKER: SubmitCreditApplication
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			_, _, _, _, _, err := exec.WithOutputFlow(&outFlow).SubmitCreditApplication(ctx, creditflowapi.Applicant{...})
			assert.NoError(err)
		})
	*/

	t.Run("submit", func(t *testing.T) {
		assert := testarossa.For(t)

		applicant := creditflowapi.Applicant{
			ApplicantName: "Alice",
			SSN:           "123-45-6789",
			Address:       "123 Main St",
			Phone:         "555-123-4567",
			Employers:     []string{"Acme Corp", "Globex"},
			CreditScore:   750,
		}
		applicantName, ssn, address, phone, employers, creditScore, err := exec.SubmitCreditApplication(ctx, applicant)
		if assert.NoError(err) {
			assert.Expect(
				applicantName, "Alice",
				ssn, "123-45-6789",
				address, "123 Main St",
				phone, "555-123-4567",
				employers, []string{"Acme Corp", "Globex"},
				creditScore, 750,
			)
		}
	})
}

func TestCreditFlow_VerifyCredit(t *testing.T) { // MARKER: VerifyCredit
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			creditVerified, err := exec.WithOutputFlow(&outFlow).VerifyCredit(ctx, creditScore)
			if assert.NoError(err) {
				assert.Expect(creditVerified, true)
			}
		})
	*/

	t.Run("good_score", func(t *testing.T) {
		assert := testarossa.For(t)

		creditVerified, err := exec.VerifyCredit(ctx, 750)
		if assert.NoError(err) {
			assert.Expect(creditVerified, true)
		}
	})

	t.Run("bad_score", func(t *testing.T) {
		assert := testarossa.For(t)

		creditVerified, err := exec.VerifyCredit(ctx, 540)
		if assert.NoError(err) {
			assert.Expect(creditVerified, false)
		}
	})
}

func TestCreditFlow_VerifyEmployment(t *testing.T) { // MARKER: VerifyEmployment
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			employmentFailuresOut, err := exec.WithOutputFlow(&outFlow).VerifyEmployment(ctx, applicantName, employerName)
			if assert.NoError(err) {
				assert.Expect(employmentFailuresOut, 0)
			}
		})
	*/

	t.Run("employed", func(t *testing.T) {
		assert := testarossa.For(t)

		employmentFailuresOut, err := exec.VerifyEmployment(ctx, "Alice", "Acme Corp")
		if assert.NoError(err) {
			assert.Expect(employmentFailuresOut, 0)
		}
	})

	t.Run("empty_name", func(t *testing.T) {
		assert := testarossa.For(t)

		employmentFailuresOut, err := exec.VerifyEmployment(ctx, "Alice", "")
		if assert.NoError(err) {
			assert.Expect(employmentFailuresOut, 1)
		}
	})
}

func TestCreditFlow_VerifySSN(t *testing.T) { // MARKER: VerifySSN
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			ssnVerified, err := exec.WithOutputFlow(&outFlow).VerifySSN(ctx, ssn)
			if assert.NoError(err) {
				assert.Expect(ssnVerified, true)
			}
		})
	*/

	t.Run("valid", func(t *testing.T) {
		assert := testarossa.For(t)
		ssnVerified, err := exec.VerifySSN(ctx, "123-45-6789")
		if assert.NoError(err) {
			assert.Expect(ssnVerified, true)
		}
	})

	t.Run("empty_ssn", func(t *testing.T) {
		assert := testarossa.For(t)
		ssnVerified, err := exec.VerifySSN(ctx, "")
		if assert.NoError(err) {
			assert.Expect(ssnVerified, false)
		}
	})
}

func TestCreditFlow_VerifyAddress(t *testing.T) { // MARKER: VerifyAddress
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			addressVerified, err := exec.WithOutputFlow(&outFlow).VerifyAddress(ctx, address)
			if assert.NoError(err) {
				assert.Expect(addressVerified, true)
			}
		})
	*/

	t.Run("valid", func(t *testing.T) {
		assert := testarossa.For(t)
		addressVerified, err := exec.VerifyAddress(ctx, "123 Main St")
		if assert.NoError(err) {
			assert.Expect(addressVerified, true)
		}
	})
}

func TestCreditFlow_VerifyPhoneNumber(t *testing.T) { // MARKER: VerifyPhoneNumber
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			phoneVerified, err := exec.WithOutputFlow(&outFlow).VerifyPhoneNumber(ctx, phone)
			if assert.NoError(err) {
				assert.Expect(phoneVerified, true)
			}
		})
	*/

	t.Run("valid", func(t *testing.T) {
		assert := testarossa.For(t)
		phoneVerified, err := exec.VerifyPhoneNumber(ctx, "555-123-4567")
		if assert.NoError(err) {
			assert.Expect(phoneVerified, true)
		}
	})
}

func TestCreditFlow_IdentityDecision(t *testing.T) { // MARKER: IdentityDecision
	t.Parallel()
	ctx := t.Context()

	svc := NewService()
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			identityVerified, err := exec.WithOutputFlow(&outFlow).IdentityDecision(ctx, ssnVerified, addressVerified, phoneVerified)
			if assert.NoError(err) {
				assert.Expect(identityVerified, true)
			}
		})
	*/

	t.Run("all_verified", func(t *testing.T) {
		assert := testarossa.For(t)
		identityVerified, err := exec.IdentityDecision(ctx, true, true, true)
		if assert.NoError(err) {
			assert.Expect(identityVerified, true)
		}
	})

	t.Run("ssn_failed", func(t *testing.T) {
		assert := testarossa.For(t)
		identityVerified, err := exec.IdentityDecision(ctx, false, true, true)
		if assert.NoError(err) {
			assert.Expect(identityVerified, false)
		}
	})
}

func TestCreditFlow_RequestMoreInfo(t *testing.T) { // MARKER: RequestMoreInfo
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			reviewAttemptsOut, err := exec.WithOutputFlow(&outFlow).RequestMoreInfo(ctx, reviewAttempts)
			if assert.NoError(err) {
				assert.Expect(reviewAttemptsOut, 1)
			}
		})
	*/

	t.Run("increments_counter", func(t *testing.T) {
		assert := testarossa.For(t)

		reviewAttemptsOut, err := exec.RequestMoreInfo(ctx, 0)
		if assert.NoError(err) {
			assert.Expect(reviewAttemptsOut, 1)
		}
	})

	t.Run("increments_again", func(t *testing.T) {
		assert := testarossa.For(t)

		reviewAttemptsOut, err := exec.RequestMoreInfo(ctx, 1)
		if assert.NoError(err) {
			assert.Expect(reviewAttemptsOut, 2)
		}
	})
}

func TestCreditFlow_ReviewCredit(t *testing.T) { // MARKER: ReviewCredit
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			creditVerifiedOut, err := exec.WithOutputFlow(&outFlow).ReviewCredit(ctx, creditScore, creditVerified, reviewAttempts)
			if assert.NoError(err) {
				assert.Expect(creditVerifiedOut, true)
			}
		})
	*/

	t.Run("high_borderline_approved", func(t *testing.T) {
		assert := testarossa.For(t)

		creditVerifiedOut, err := exec.ReviewCredit(ctx, 580, false, 0)
		if assert.NoError(err) {
			assert.Expect(creditVerifiedOut, true)
		}
	})

	t.Run("too_low_rejected", func(t *testing.T) {
		assert := testarossa.For(t)

		creditVerifiedOut, err := exec.ReviewCredit(ctx, 400, false, 0)
		if assert.NoError(err) {
			assert.Expect(creditVerifiedOut, false)
		}
	})
}

func TestCreditFlow_Decision(t *testing.T) { // MARKER: Decision
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			var outFlow workflow.Flow
			approved, err := exec.WithOutputFlow(&outFlow).Decision(ctx, creditVerified, employmentFailures, identityVerified)
			if assert.NoError(err) {
				assert.Expect(approved, true)
			}
		})
	*/

	t.Run("all_pass", func(t *testing.T) {
		assert := testarossa.For(t)

		approved, err := exec.Decision(ctx, true, 0, true)
		if assert.NoError(err) {
			assert.Expect(approved, true)
		}
	})

	t.Run("credit_fails", func(t *testing.T) {
		assert := testarossa.For(t)

		approved, err := exec.Decision(ctx, false, 0, true)
		if assert.NoError(err) {
			assert.Expect(approved, false)
		}
	})
}

func TestCreditFlow_CreditApproval(t *testing.T) { // MARKER: CreditApproval
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	foremanClient := foremanapi.NewClient(tester)
	exec := creditflowapi.NewExecutor(tester).WithWorkflowRunner(foremanClient)

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		foreman.NewService(),
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case.
		Use WithOutputState to also inspect the full state map if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			approved, creditVerified, employmentFailures, identityVerified, status, err := exec.CreditApproval(ctx, creditflowapi.Applicant{...})
			assert.Expect(
				err, nil,
				status, workflow.StatusCompleted,
				approved, expectedApproved,
			)
		})
	*/

	t.Run("good_score_approved", func(t *testing.T) {
		assert := testarossa.For(t)

		// Score 750: passes VerifyCredit, review passes through, approved
		approved, creditVerified, _, identityVerified, status, err := exec.CreditApproval(ctx, creditflowapi.Applicant{
			ApplicantName: "Alice",
			SSN:           "123-45-6789",
			Address:       "123 Main St",
			Phone:         "555-123-4567",
			Employers:     []string{"Acme Corp", "Globex"},
			CreditScore:   750,
		})
		if assert.NoError(err) {
			assert.Expect(
				status, workflow.StatusCompleted,
				approved, true,
				creditVerified, true,
				identityVerified, true,
			)
		}
	})

	t.Run("low_score_rejected", func(t *testing.T) {
		assert := testarossa.For(t)

		// Score 400: fails VerifyCredit, review cannot save it, rejected
		approved, _, _, _, status, err := exec.CreditApproval(ctx, creditflowapi.Applicant{
			ApplicantName: "Bob",
			SSN:           "987-65-4321",
			Address:       "456 Oak Ave",
			Phone:         "555-987-6543",
			Employers:     []string{"Globex"},
			CreditScore:   400,
		})
		if assert.NoError(err) {
			assert.Expect(
				status, workflow.StatusCompleted,
				approved, false,
			)
		}
	})

	t.Run("borderline_with_review_approved", func(t *testing.T) {
		assert := testarossa.For(t)

		// Score 580: passes VerifyCredit, routed to ReviewCredit which approves 580+
		approved, creditVerified, _, _, status, err := exec.CreditApproval(ctx, creditflowapi.Applicant{
			ApplicantName: "Charlie",
			SSN:           "111-22-3333",
			Address:       "789 Elm Dr",
			Phone:         "(555) 111-2222",
			Employers:     []string{"Initech"},
			CreditScore:   580,
		})
		if assert.NoError(err) {
			assert.Expect(
				status, workflow.StatusCompleted,
				approved, true,
				creditVerified, true,
			)
		}
	})

	t.Run("very_borderline_goto_loop", func(t *testing.T) {
		assert := testarossa.For(t)

		// Score 560: passes VerifyCredit, review requests more info twice via goto, then approves
		approved, creditVerified, _, _, status, err := exec.CreditApproval(ctx, creditflowapi.Applicant{
			ApplicantName: "Diana",
			SSN:           "444-55-6666",
			Address:       "321 Pine Ln",
			Phone:         "555-444-3333",
			Employers:     []string{"Umbrella Corp"},
			CreditScore:   560,
		})
		if assert.NoError(err) {
			assert.Expect(
				status, workflow.StatusCompleted,
				approved, true,
				creditVerified, true,
			)
		}
	})

	t.Run("employment_failure", func(t *testing.T) {
		assert := testarossa.For(t)

		// Good credit but one empty employer name causes employment failure
		approved, _, _, _, status, err := exec.CreditApproval(ctx, creditflowapi.Applicant{
			ApplicantName: "Eve",
			SSN:           "555-66-7777",
			Address:       "654 Maple Ct",
			Phone:         "555-555-5555",
			Employers:     []string{"Acme Corp", ""},
			CreditScore:   750,
		})
		if assert.NoError(err) {
			assert.Expect(
				status, workflow.StatusCompleted,
				approved, false,
			)
		}
	})

	t.Run("empty_applicant_rejected", func(t *testing.T) {
		assert := testarossa.For(t)

		// Empty applicant name, invalid address, and bad phone fail identity verification
		approved, _, _, identityVerified, status, err := exec.CreditApproval(ctx, creditflowapi.Applicant{
			SSN:         "000-00-0000",
			Address:     "Nowhere",
			Phone:       "bad",
			Employers:   []string{"Acme Corp"},
			CreditScore: 750,
		})
		if assert.NoError(err) {
			assert.Expect(
				status, workflow.StatusCompleted,
				approved, false,
				identityVerified, false,
			)
		}
	})

}

func TestCreditFlow_Await(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	foremanClient := foremanapi.NewClient(tester)

	// Run the testing app
	app := application.New()
	app.Add(svc, foreman.NewService(), tester)
	app.RunInTest(t)

	goodApplicant := creditflowapi.Applicant{
		ApplicantName: "Alice",
		SSN:           "123-45-6789",
		Address:       "123 Main St",
		Phone:         "555-123-4567",
		Employers:     []string{"Acme Corp"},
		CreditScore:   750,
	}

	t.Run("completed", func(t *testing.T) {
		assert := testarossa.For(t)

		// Create auto-runs; Await blocks until the flow stops and returns its outcome. This is the
		// replacement for the removed stop-notification event: the caller learns the outcome by awaiting it.
		flowKey, err := foremanClient.Create(ctx, creditflowapi.CreditApproval.URL(), creditflowapi.CreditApprovalIn{
			Applicant: goodApplicant,
		}, nil)
		assert.NoError(err)

		outcome, err := foremanClient.Await(ctx, flowKey)
		if assert.NoError(err) {
			assert.Expect(outcome.Status, workflow.StatusCompleted)
			assert.NotNil(outcome.State)
		}
	})
}

func TestCreditFlow_Demo(t *testing.T) { // MARKER: Demo
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester client
	tester := connector.New("tester.client")
	client := creditflowapi.NewClient(tester)
	_ = client

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		foreman.NewService(),
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Use the following pattern for each test case

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			actor := jwt.MapClaims{}
			res, err := client.WithOptions(pub.Actor(actor)).Demo(ctx, "GET", "", payload)
			if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
				body, err := io.ReadAll(res.Body)
				if assert.NoError(err) {
					assert.HTMLMatch(body, "DIV.class > DIV#id", "substring")
					assert.Contains(body, "substring")
				}
			}
		})
	*/

	t.Run("get_form", func(t *testing.T) {
		assert := testarossa.For(t)

		res, err := client.Demo(ctx, "GET", "", nil)
		if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
			body, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Contains(body, "Credit Approval Workflow Demo")
				assert.Contains(body, "Alice")
			}
		}
	})

	t.Run("post_happy_path", func(t *testing.T) {
		assert := testarossa.For(t)

		form := "name=Alice&ssn=123-45-6789&address=123+Main+St&phone=555-123-4567&employers=Acme+Corp&score=750&fault="
		res, err := client.Demo(ctx, "POST", "", form)
		if assert.NoError(err) && assert.Expect(res.StatusCode, http.StatusOK) {
			body, err := io.ReadAll(res.Body)
			if assert.NoError(err) {
				assert.Contains(body, "completed")
				assert.Contains(body, "SubmitCreditApplication")
			}
		}
	})
}

func BenchmarkCreditFlow_CreditApproval(b *testing.B) {
	svc := NewService()
	tester := connector.New("tester.client")
	foremanClient := foremanapi.NewClient(tester)

	app := application.New()
	app.Add(svc, foreman.NewService(), tester)
	app.RunInTest(b)

	ctx := b.Context()
	applicant := creditflowapi.CreditApprovalIn{
		Applicant: creditflowapi.Applicant{
			ApplicantName: "Alice",
			SSN:           "123-45-6789",
			Address:       "123 Main St",
			Phone:         "555-123-4567",
			Employers:     []string{"Acme Corp"},
			CreditScore:   750,
		},
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := foremanClient.Run(ctx, creditflowapi.CreditApproval.URL(), applicant, nil)
		if err != nil {
			b.Fatal(err)
		}
	}

	/*
		Postgres: approx 12.8 workflows/sec or 141 tasks/sec

		goos: darwin
		goarch: arm64
		pkg: github.com/microbus-io/fabric/exampleservices/creditflow
		cpu: Apple M1 Pro
		BenchmarkCreditFlow_CreditApproval-10    	      13	  78078000 ns/op	 1831651 B/op	   23091 allocs/op
	*/
}

func BenchmarkCreditFlow_CreditApprovalParallel(b *testing.B) {
	svc := NewService()
	tester := connector.New("tester.client")
	foremanClient := foremanapi.NewClient(tester)

	app := application.New()
	app.Add(svc, foreman.NewService(), tester)
	app.RunInTest(b)

	ctx := b.Context()
	applicant := creditflowapi.CreditApprovalIn{
		Applicant: creditflowapi.Applicant{
			ApplicantName: "Alice",
			SSN:           "123-45-6789",
			Address:       "123 Main St",
			Phone:         "555-123-4567",
			Employers:     []string{"Acme Corp"},
			CreditScore:   750,
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := foremanClient.Run(ctx, creditflowapi.CreditApproval.URL(), applicant, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	/*
		Postgres: approx 38.5 workflows/sec or 424 tasks/sec

		goos: darwin
		goarch: arm64
		pkg: github.com/microbus-io/fabric/exampleservices/creditflow
		cpu: Apple M1 Pro
		BenchmarkCreditFlow_CreditApprovalParallel-10    	     39	  25952910 ns/op	 1589837 B/op	   20449 allocs/op

		SQLite: approx 110 workflows/sec or 1206 tasks/sec

		goos: darwin
		goarch: arm64
		pkg: github.com/microbus-io/fabric/exampleservices/creditflow
		cpu: Apple M1 Pro
		BenchmarkCreditFlow_CreditApprovalParallel-10    	    150	   9121310 ns/op	 1395101 B/op	   19108 allocs/op
	*/
}

func TestCreditFlow_InitIdentityVerification(t *testing.T) { // MARKER: InitIdentityVerification
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)
	_ = exec

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			err := exec.InitIdentityVerification(ctx, applicantName, ssn, address, phone)
			assert.NoError(err)
		})
	*/
}

func TestCreditFlow_RunIdentityVerification(t *testing.T) { // MARKER: RunIdentityVerification
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)
	_ = exec

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			identityVerified, err := exec.RunIdentityVerification(ctx, applicantName, ssn, address, phone)
			assert.Expect(
				identityVerified, expectedIdentityVerified,
				err, nil,
			)
		})
	*/
}

func TestCreditFlow_HandleCreditError(t *testing.T) { // MARKER: HandleCreditError
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the tester
	tester := connector.New("tester.client")
	exec := creditflowapi.NewExecutor(tester)
	_ = exec

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern.
		Use WithOutputFlow to also verify control signals (Goto, Retry, Interrupt, Sleep) if applicable.

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			creditVerified, err := exec.HandleCreditError(ctx, onErr)
			assert.Expect(
				creditVerified, expectedCreditVerified,
				err, nil,
			)
		})
	*/
}

func TestCreditFlow_IdentityVerification(t *testing.T) { // MARKER: IdentityVerification
	t.Parallel()
	ctx := t.Context()
	_ = ctx

	// Initialize the microservice under test
	svc := NewService()

	// Initialize the testers
	tester := connector.New("tester.client")
	foremanClient := foremanapi.NewClient(tester)
	exec := creditflowapi.NewExecutor(tester).WithWorkflowRunner(foremanClient)
	_ = exec

	// Run the testing app
	app := application.New()
	app.Add(
		// HINT: Add microservices or mocks required for this test
		svc,
		foreman.NewService(),
		tester,
	)
	app.RunInTest(t)

	/*
		HINT: Fill in test cases using the following pattern

		t.Run("test_case_name", func(t *testing.T) {
			assert := testarossa.For(t)

			identityVerified, status, err := exec.IdentityVerification(ctx, applicantName, ssn, address, phone)
			assert.Expect(
				err, nil,
				status, workflow.StatusCompleted,
				identityVerified, expectedIdentityVerified,
			)
		})
	*/
}
