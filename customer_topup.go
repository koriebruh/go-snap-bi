package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CustomerTopUpRequest is the request body for API Customer Top Up
// (Service Code 38). PartnerReferenceNo is the only mandatory field
// per the Guides tab. CategoryID is documented Numeric but shown
// quoted on the wire; per the package's ambiguous-type rule it is
// json.RawMessage rather than string, matching CustomerMonthlyInLimit
// on the sibling Account Inquiry - Customer Top Up endpoint (37).
type CustomerTopUpRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo"`
	CustomerNumber     string          `json:"customerNumber,omitempty"`
	CustomerName       string          `json:"customerName,omitempty"`
	Amount             *Money          `json:"amount,omitempty"`
	FeeAmount          *Money          `json:"feeAmount,omitempty"`
	TransactionDate    string          `json:"transactionDate,omitempty"`
	SessionID          string          `json:"sessionId,omitempty"`
	CategoryID         json.RawMessage `json:"categoryId,omitempty"`
	Notes              string          `json:"notes,omitempty"`
}

// CustomerTopUpResponse is the response body for API Customer Top Up.
// ReferenceNumber is not listed in the Guides tab's field table for
// this endpoint, but is present in its own worked example
// ("REF993883") — an empirically observed field, not a guess, per the
// package's practice of trusting worked examples over an incomplete
// table.
type CustomerTopUpResponse struct {
	ResponseCode       string `json:"responseCode"`
	ResponseMessage    string `json:"responseMessage"`
	ReferenceNo        string `json:"referenceNo,omitempty"`
	PartnerReferenceNo string `json:"partnerReferenceNo,omitempty"`
	SessionID          string `json:"sessionId,omitempty"`
	CustomerNumber     string `json:"customerNumber,omitempty"`
	Amount             *Money `json:"amount,omitempty"`
	ReferenceNumber    string `json:"referenceNumber,omitempty"`
}

// CustomerTopUp calls the SNAP Customer Top Up endpoint (Service Code
// 38, path .../{version}/customer-top-up, HTTP POST — no method
// override). hb must already carry every field HeaderBuilder needs
// except Body, which CustomerTopUp sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func CustomerTopUp(ctx context.Context, t *Transport, hb HeaderBuilder, req CustomerTopUpRequest) (CustomerTopUpResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CustomerTopUpResponse{}, fmt.Errorf("snap: customer top up: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CustomerTopUpResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CustomerTopUpResponse{}, fmt.Errorf("snap: customer top up: %w", err)
	}

	var resp CustomerTopUpResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CustomerTopUpResponse{}, fmt.Errorf("snap: customer top up: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CustomerTopUpResponse{}, errors.New("snap: customer top up: response has no responseCode")
	}
	return resp, nil
}
