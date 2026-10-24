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

package creditflowapi

import (
	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/define"
	"time"
)

// HINT: This file is the single source of truth for the microservice's API. After editing it, run
// cmd/genservice on the microservice's directory (the parent of this api package) to regenerate client.go,
// intermediate.go, mock.go, mock_test.go, and manifest.yaml. Do not hand-edit those generated files.

// Hostname is the default hostname of the microservice.
const Hostname = "creditflow.example"

// Name is the decorative PascalCase name of the microservice.
const Name = "CreditFlow"

// Version is a generation counter bumped on each regeneration, not a semantic version.
const Version = 8

// Description is the human-readable summary of the microservice, surfaced in OpenAPI and discovery.
const Description = `CreditFlow is an example microservice that demonstrates agentic workflow features.`

// Demo serves the demo page for the credit approval workflow.
var Demo = define.Web{ // MARKER: Demo
	Host: Hostname, Method: "ANY", Route: "/demo",
}

// SubmitCreditApplication receives a credit application and unpacks the applicant into individual workflow
// state fields.
var SubmitCreditApplication = define.Task{ // MARKER: SubmitCreditApplication
	Host: Hostname, Method: "POST", Route: ":428/submit-credit-application",
	In: SubmitCreditApplicationIn{}, Out: SubmitCreditApplicationOut{},
}

// SubmitCreditApplicationIn are the input arguments of SubmitCreditApplication.
type SubmitCreditApplicationIn struct { // MARKER: SubmitCreditApplication
	Applicant Applicant `json:"applicant,omitzero"`
}

// SubmitCreditApplicationOut are the output arguments of SubmitCreditApplication.
type SubmitCreditApplicationOut struct { // MARKER: SubmitCreditApplication
	ApplicantName string   `json:"applicantName,omitzero"`
	SSN           string   `json:"ssn,omitzero"`
	Address       string   `json:"address,omitzero"`
	Phone         string   `json:"phone,omitzero"`
	Employers     []string `json:"employers,omitzero"`
	CreditScore   int      `json:"creditScore,omitzero"`
}

// VerifyCredit checks the applicant's credit score.
var VerifyCredit = define.Task{ // MARKER: VerifyCredit
	Host: Hostname, Method: "POST", Route: ":428/verify-credit",
	In: VerifyCreditIn{}, Out: VerifyCreditOut{},
}

// VerifyCreditIn are the input arguments of VerifyCredit.
type VerifyCreditIn struct { // MARKER: VerifyCredit
	CreditScore int `json:"creditScore,omitzero"`
}

// VerifyCreditOut are the output arguments of VerifyCredit.
type VerifyCreditOut struct { // MARKER: VerifyCredit
	CreditVerified bool `json:"creditVerified,omitzero"`
}

// VerifyEmployment checks the applicant's employment status.
var VerifyEmployment = define.Task{ // MARKER: VerifyEmployment
	Host: Hostname, Method: "POST", Route: ":428/verify-employment",
	In: VerifyEmploymentIn{}, Out: VerifyEmploymentOut{},
}

// VerifyEmploymentIn are the input arguments of VerifyEmployment.
type VerifyEmploymentIn struct { // MARKER: VerifyEmployment
	ApplicantName string `json:"applicantName,omitzero"`
	EmployerName  string `json:"employerName,omitzero"`
}

// VerifyEmploymentOut are the output arguments of VerifyEmployment.
type VerifyEmploymentOut struct { // MARKER: VerifyEmployment
	EmploymentFailuresOut int `json:"employmentFailures,omitzero"`
}

// InitIdentityVerification is the entry point for the identity verification subgraph.
var InitIdentityVerification = define.Task{ // MARKER: InitIdentityVerification
	Host: Hostname, Method: "POST", Route: ":428/init-identity-verification",
	In: InitIdentityVerificationIn{}, Out: InitIdentityVerificationOut{},
}

// InitIdentityVerificationIn are the input arguments of InitIdentityVerification.
type InitIdentityVerificationIn struct { // MARKER: InitIdentityVerification
	ApplicantName string `json:"applicantName,omitzero"`
	SSN           string `json:"ssn,omitzero"`
	Address       string `json:"address,omitzero"`
	Phone         string `json:"phone,omitzero"`
}

// InitIdentityVerificationOut are the output arguments of InitIdentityVerification.
type InitIdentityVerificationOut struct { // MARKER: InitIdentityVerification
}

// VerifySSN checks the applicant's SSN.
var VerifySSN = define.Task{ // MARKER: VerifySSN
	Host: Hostname, Method: "POST", Route: ":428/verify-ssn",
	In: VerifySSNIn{}, Out: VerifySSNOut{},
}

// VerifySSNIn are the input arguments of VerifySSN.
type VerifySSNIn struct { // MARKER: VerifySSN
	SSN string `json:"ssn,omitzero"`
}

// VerifySSNOut are the output arguments of VerifySSN.
type VerifySSNOut struct { // MARKER: VerifySSN
	SsnVerified bool `json:"ssnVerified,omitzero"`
}

// VerifyAddress checks the applicant's address.
var VerifyAddress = define.Task{ // MARKER: VerifyAddress
	Host: Hostname, Method: "POST", Route: ":428/verify-address",
	In: VerifyAddressIn{}, Out: VerifyAddressOut{},
}

// VerifyAddressIn are the input arguments of VerifyAddress.
type VerifyAddressIn struct { // MARKER: VerifyAddress
	Address string `json:"address,omitzero"`
}

// VerifyAddressOut are the output arguments of VerifyAddress.
type VerifyAddressOut struct { // MARKER: VerifyAddress
	AddressVerified bool `json:"addressVerified,omitzero"`
}

// VerifyPhoneNumber checks the applicant's phone number.
var VerifyPhoneNumber = define.Task{ // MARKER: VerifyPhoneNumber
	Host: Hostname, Method: "POST", Route: ":428/verify-phone-number",
	TimeBudget: time.Second,
	In:         VerifyPhoneNumberIn{}, Out: VerifyPhoneNumberOut{},
}

// VerifyPhoneNumberIn are the input arguments of VerifyPhoneNumber.
type VerifyPhoneNumberIn struct { // MARKER: VerifyPhoneNumber
	Phone string `json:"phone,omitzero"`
}

// VerifyPhoneNumberOut are the output arguments of VerifyPhoneNumber.
type VerifyPhoneNumberOut struct { // MARKER: VerifyPhoneNumber
	PhoneVerified bool `json:"phoneVerified,omitzero"`
}

// IdentityDecision determines whether the applicant's identity is verified based on SSN, address, and phone checks.
var IdentityDecision = define.Task{ // MARKER: IdentityDecision
	Host: Hostname, Method: "POST", Route: ":428/identity-decision",
	In: IdentityDecisionIn{}, Out: IdentityDecisionOut{},
}

// IdentityDecisionIn are the input arguments of IdentityDecision.
type IdentityDecisionIn struct { // MARKER: IdentityDecision
	SsnVerified     bool `json:"ssnVerified,omitzero"`
	AddressVerified bool `json:"addressVerified,omitzero"`
	PhoneVerified   bool `json:"phoneVerified,omitzero"`
}

// IdentityDecisionOut are the output arguments of IdentityDecision.
type IdentityDecisionOut struct { // MARKER: IdentityDecision
	IdentityVerified bool `json:"identityVerified,omitzero"`
}

// RunIdentityVerification invokes the IdentityVerification subgraph via flow.Subgraph and adopts identityVerified.
var RunIdentityVerification = define.Task{ // MARKER: RunIdentityVerification
	Host: Hostname, Method: "POST", Route: ":428/run-identity-verification",
	In: RunIdentityVerificationIn{}, Out: RunIdentityVerificationOut{},
}

// RunIdentityVerificationIn are the input arguments of RunIdentityVerification.
type RunIdentityVerificationIn struct { // MARKER: RunIdentityVerification
	ApplicantName string `json:"applicantName,omitzero"`
	SSN           string `json:"ssn,omitzero"`
	Address       string `json:"address,omitzero"`
	Phone         string `json:"phone,omitzero"`
}

// RunIdentityVerificationOut are the output arguments of RunIdentityVerification.
type RunIdentityVerificationOut struct { // MARKER: RunIdentityVerification
	IdentityVerified bool `json:"identityVerified,omitzero"`
}

// RequestMoreInfo requests additional information for the credit review and increments the review attempt counter.
var RequestMoreInfo = define.Task{ // MARKER: RequestMoreInfo
	Host: Hostname, Method: "POST", Route: ":428/request-more-info",
	In: RequestMoreInfoIn{}, Out: RequestMoreInfoOut{},
}

// RequestMoreInfoIn are the input arguments of RequestMoreInfo.
type RequestMoreInfoIn struct { // MARKER: RequestMoreInfo
	ReviewAttempts int `json:"reviewAttempts,omitzero"`
}

// RequestMoreInfoOut are the output arguments of RequestMoreInfo.
type RequestMoreInfoOut struct { // MARKER: RequestMoreInfo
	ReviewAttemptsOut int `json:"reviewAttempts,omitzero"`
}

// ReviewCredit performs a manual review of borderline credit scores. For very borderline scores (550-579)
// it requests more info up to 2 times before deciding; scores of 580+ are approved and below 550 are
// rejected.
var ReviewCredit = define.Task{ // MARKER: ReviewCredit
	Host: Hostname, Method: "POST", Route: ":428/review-credit",
	In: ReviewCreditIn{}, Out: ReviewCreditOut{},
}

// ReviewCreditIn are the input arguments of ReviewCredit.
type ReviewCreditIn struct { // MARKER: ReviewCredit
	CreditScore    int  `json:"creditScore,omitzero"`
	CreditVerified bool `json:"creditVerified,omitzero"`
	ReviewAttempts int  `json:"reviewAttempts,omitzero"`
}

// ReviewCreditOut are the output arguments of ReviewCredit.
type ReviewCreditOut struct { // MARKER: ReviewCredit
	CreditVerifiedOut bool `json:"creditVerified,omitzero"`
}

// HandleCreditError handles a credit verification error by setting creditVerified to false.
var HandleCreditError = define.Task{ // MARKER: HandleCreditError
	Host: Hostname, Method: "POST", Route: ":428/handle-credit-error",
	In: HandleCreditErrorIn{}, Out: HandleCreditErrorOut{},
}

// HandleCreditErrorIn are the input arguments of HandleCreditError.
type HandleCreditErrorIn struct { // MARKER: HandleCreditError
	OnErr *errors.TracedError `json:"onErr,omitzero"`
}

// HandleCreditErrorOut are the output arguments of HandleCreditError.
type HandleCreditErrorOut struct { // MARKER: HandleCreditError
	CreditVerified bool `json:"creditVerified,omitzero"`
}

// Decision determines whether to approve the credit application based on verification results.
var Decision = define.Task{ // MARKER: Decision
	Host: Hostname, Method: "POST", Route: ":428/decision",
	In: DecisionIn{}, Out: DecisionOut{},
}

// DecisionIn are the input arguments of Decision.
type DecisionIn struct { // MARKER: Decision
	CreditVerified     bool `json:"creditVerified,omitzero"`
	EmploymentFailures int  `json:"employmentFailures,omitzero"`
	IdentityVerified   bool `json:"identityVerified,omitzero"`
}

// DecisionOut are the output arguments of Decision.
type DecisionOut struct { // MARKER: Decision
	Approved bool `json:"approved,omitzero"`
}

// IdentityVerification defines the workflow graph for the identity verification process.
var IdentityVerification = define.Workflow{ // MARKER: IdentityVerification
	Host: Hostname, Method: "GET", Route: ":428/identity-verification",
	In: IdentityVerificationIn{}, Out: IdentityVerificationOut{},
}

// IdentityVerificationIn are the input arguments of IdentityVerification.
type IdentityVerificationIn struct { // MARKER: IdentityVerification
	ApplicantName string `json:"applicantName,omitzero"`
	SSN           string `json:"ssn,omitzero"`
	Address       string `json:"address,omitzero"`
	Phone         string `json:"phone,omitzero"`
}

// IdentityVerificationOut are the output arguments of IdentityVerification.
type IdentityVerificationOut struct { // MARKER: IdentityVerification
	IdentityVerified bool `json:"identityVerified,omitzero"`
}

// CreditApproval defines the workflow graph for the full credit approval process.
var CreditApproval = define.Workflow{ // MARKER: CreditApproval
	Host: Hostname, Method: "GET", Route: ":428/credit-approval",
	In: CreditApprovalIn{}, Out: CreditApprovalOut{},
}

// CreditApprovalIn are the input arguments of CreditApproval.
type CreditApprovalIn struct { // MARKER: CreditApproval
	Applicant Applicant `json:"applicant,omitzero"`
}

// CreditApprovalOut are the output arguments of CreditApproval.
type CreditApprovalOut struct { // MARKER: CreditApproval
	Approved           bool `json:"approved,omitzero"`
	CreditVerified     bool `json:"creditVerified,omitzero"`
	EmploymentFailures int  `json:"employmentFailures,omitzero"`
	IdentityVerified   bool `json:"identityVerified,omitzero"`
}
