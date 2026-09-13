package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// TransferToOTCCancelPaymentRequest is the request body for API
// Transfer To OTC - Cancel Payment (Service Code 46).
// OriginalPartnerReferenceNo, CustomerNumber, and Reason are mandatory
// per the Guides tab; OriginalReferenceNo is Conditional here (it
// flips to Mandatory in the response — see
// TransferToOTCCancelPaymentResponse's doc comment).
type TransferToOTCCancelPaymentRequest struct {
	OriginalReferenceNo        string `json:"originalReferenceNo,omitempty"`
	OriginalPartnerReferenceNo string `json:"originalPartnerReferenceNo"`
	OriginalExternalID         string `json:"originalExternalId,omitempty"`
	CustomerNumber             string `json:"customerNumber"`
	Reason                     string `json:"reason"`
}

// TransferToOTCCancelPaymentResponse is the response body for API
// Transfer To OTC - Cancel Payment. OriginalReferenceNo is Mandatory
// here, unlike Conditional on the request — research §5.8 explicitly
// notes this flip. CancelTime is Conditional ("must be filled if
// cancelled transaction success" — a data-dependent condition this
// package cannot express in the type system), so it is Optional here.
type TransferToOTCCancelPaymentResponse struct {
	ResponseCode        string `json:"responseCode"`
	ResponseMessage     string `json:"responseMessage"`
	OriginalReferenceNo string `json:"originalReferenceNo"`
	CancelTime          string `json:"cancelTime,omitempty"`
	TransactionDate     string `json:"transactionDate,omitempty"`
}

// TransferToOTCCancelPayment calls the SNAP Transfer To OTC - Cancel
// Payment endpoint (Service Code 46, HTTP POST — no method override).
// The path is documented two conflicting ways: the portal's Overview
// tab says .../{version}/emoney/otc-cancel, but the Code Snippet tab's
// own request line reads POST .../v1.0/otc/cashout/cancel — research
// explicitly flags this as unresolved, needing sandbox/Postman
// verification before hardcoding either reading (research §6 item 5).
// This package uses the Overview reading in this comment, consistent
// with the sub-group's other two endpoints' emoney/otc-* naming, but
// makes no runtime assumption either way: hb must already carry every
// field snap.HeaderBuilder needs, including EndpointURL, which the caller
// always supplies. TransferToOTCCancelPayment sets Body itself so the
// exact marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func TransferToOTCCancelPayment(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req TransferToOTCCancelPaymentRequest) (TransferToOTCCancelPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TransferToOTCCancelPaymentResponse{}, fmt.Errorf("snap: transfer to otc cancel payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return TransferToOTCCancelPaymentResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return TransferToOTCCancelPaymentResponse{}, fmt.Errorf("snap: transfer to otc cancel payment: %w", err)
	}

	var resp TransferToOTCCancelPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return TransferToOTCCancelPaymentResponse{}, fmt.Errorf("snap: transfer to otc cancel payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return TransferToOTCCancelPaymentResponse{}, errors.New("snap: transfer to otc cancel payment: response has no responseCode")
	}
	return resp, nil
}
