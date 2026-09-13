package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// AccountInquiryCustomerTopUpRequest is the request body for API
// Account Inquiry - Customer Top Up (Service Code 37). Amount is the
// only mandatory field per the Guides tab. CustomerNumber is
// Conditional ("mandatory if B2B2C token null" — a data-dependent
// condition this package cannot express in the type system), so it is
// Optional here.
type AccountInquiryCustomerTopUpRequest struct {
	PartnerReferenceNo string     `json:"partnerReferenceNo,omitempty"`
	CustomerNumber     string     `json:"customerNumber,omitempty"`
	Amount             snap.Money `json:"amount"`
	TransactionDate    string     `json:"transactionDate,omitempty"`
}

// AccountInquiryCustomerTopUpResponse is the response body for API
// Account Inquiry - Customer Top Up. CustomerNumber is documented
// String(64) here (masked, e.g. "XXXXXXXXX1857") vs the request's
// String(32) — both are plain string, Go strings have no length
// constraint. CustomerMonthlyInLimit is documented Numeric but shown
// quoted on the wire; per the package's ambiguous-type rule it is
// json.RawMessage rather than string, since one worked example is thin
// evidence about every issuer's wire shape (BillReferenceNo precedent,
// Phase 13).
type AccountInquiryCustomerTopUpResponse struct {
	ResponseCode           string          `json:"responseCode"`
	ResponseMessage        string          `json:"responseMessage"`
	ReferenceNo            string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo     string          `json:"partnerReferenceNo,omitempty"`
	SessionID              string          `json:"sessionId,omitempty"`
	CustomerNumber         string          `json:"customerNumber,omitempty"`
	CustomerName           string          `json:"customerName"`
	CustomerMonthlyInLimit json.RawMessage `json:"customerMonthlyInLimit,omitempty"`
	MinAmount              *snap.Money     `json:"minAmount,omitempty"`
	MaxAmount              *snap.Money     `json:"maxAmount,omitempty"`
	Amount                 *snap.Money     `json:"amount,omitempty"`
	FeeAmount              *snap.Money     `json:"feeAmount,omitempty"`
	FeeType                string          `json:"feeType,omitempty"`
}

// AccountInquiryCustomerTopUp calls the SNAP Account Inquiry -
// Customer Top Up endpoint (Service Code 37, path
// .../{version}/account-inquiry-customer-top-up, HTTP POST — no method
// override). hb must already carry every field snap.HeaderBuilder needs
// except Body, which AccountInquiryCustomerTopUp sets itself so the
// exact marshaled bytes are used for both signing and the wire body.
func AccountInquiryCustomerTopUp(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req AccountInquiryCustomerTopUpRequest) (AccountInquiryCustomerTopUpResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AccountInquiryCustomerTopUpResponse{}, fmt.Errorf("snap: account inquiry customer top up: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AccountInquiryCustomerTopUpResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AccountInquiryCustomerTopUpResponse{}, fmt.Errorf("snap: account inquiry customer top up: %w", err)
	}

	var resp AccountInquiryCustomerTopUpResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AccountInquiryCustomerTopUpResponse{}, fmt.Errorf("snap: account inquiry customer top up: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AccountInquiryCustomerTopUpResponse{}, errors.New("snap: account inquiry customer top up: response has no responseCode")
	}
	return resp, nil
}
