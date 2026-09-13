package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// DirectDebitBIFASTEMandateRegistrationRequest is the request body for
// API Registrasi e-Mandate (Service Code 70). BankCode, SourceAccountNo,
// SourceAccountName, BillerID, BillerName, CustomerID, and
// ExpiredDatetime are Mandatory per research §5.4.
type DirectDebitBIFASTEMandateRegistrationRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	BankCode           string          `json:"bankCode"`
	SourceAccountNo    string          `json:"sourceAccountNo"`
	SourceAccountName  string          `json:"sourceAccountName"`
	MaxAmount          *Money          `json:"maxAmount,omitempty"`
	BillerID           string          `json:"billerId"`
	BillerName         string          `json:"billerName"`
	CustomerID         string          `json:"customerId"`
	ExpiredDatetime    string          `json:"expiredDatetime"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitBIFASTEMandateRegistrationResponse is the response body
// for API Registrasi e-Mandate. ReferenceNo is Conditional (success
// only). EMandateReffID is Mandatory per research §5.4.
type DirectDebitBIFASTEMandateRegistrationResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	EMandateReffID     string          `json:"eMandateReffId"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitBIFASTEMandateRegistration calls the SNAP Registrasi
// e-Mandate endpoint (Service Code 70, path
// .../{version}/debit/fast-emandate, HTTP POST — no method override).
// hb must already carry every field HeaderBuilder needs except Body,
// which DirectDebitBIFASTEMandateRegistration sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func DirectDebitBIFASTEMandateRegistration(ctx context.Context, t *Transport, hb HeaderBuilder, req DirectDebitBIFASTEMandateRegistrationRequest) (DirectDebitBIFASTEMandateRegistrationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return DirectDebitBIFASTEMandateRegistrationResponse{}, fmt.Errorf("snap: direct debit bifast emandate registration: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return DirectDebitBIFASTEMandateRegistrationResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return DirectDebitBIFASTEMandateRegistrationResponse{}, fmt.Errorf("snap: direct debit bifast emandate registration: %w", err)
	}

	var resp DirectDebitBIFASTEMandateRegistrationResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return DirectDebitBIFASTEMandateRegistrationResponse{}, fmt.Errorf("snap: direct debit bifast emandate registration: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return DirectDebitBIFASTEMandateRegistrationResponse{}, errors.New("snap: direct debit bifast emandate registration: response has no responseCode")
	}
	return resp, nil
}
