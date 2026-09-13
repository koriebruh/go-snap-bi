package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// TransferToOTCCreatePaymentRequest is the request body for API
// Transfer To OTC - Create Payment (Service Code 44).
// PartnerReferenceNo, CustomerNumber, OTP, and Amount are mandatory
// per the Guides tab.
type TransferToOTCCreatePaymentRequest struct {
	PartnerReferenceNo string     `json:"partnerReferenceNo"`
	CustomerNumber     string     `json:"customerNumber"`
	OTP                string     `json:"otp"`
	Amount             snap.Money `json:"amount"`
	FeeType            string     `json:"feeType,omitempty"`
}

// TransferToOTCCreatePaymentResponse is the response body for API
// Transfer To OTC - Create Payment.
type TransferToOTCCreatePaymentResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	ReferenceNo     string `json:"referenceNo,omitempty"`
	TransactionDate string `json:"transactionDate,omitempty"`
}

// TransferToOTCCreatePayment calls the SNAP Transfer To OTC - Create
// Payment endpoint (Service Code 44, path
// .../{version}/emoney/otc-cashout, HTTP POST — no method override).
// hb must already carry every field snap.HeaderBuilder needs except Body,
// which TransferToOTCCreatePayment sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func TransferToOTCCreatePayment(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req TransferToOTCCreatePaymentRequest) (TransferToOTCCreatePaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TransferToOTCCreatePaymentResponse{}, fmt.Errorf("snap: transfer to otc create payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return TransferToOTCCreatePaymentResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return TransferToOTCCreatePaymentResponse{}, fmt.Errorf("snap: transfer to otc create payment: %w", err)
	}

	var resp TransferToOTCCreatePaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return TransferToOTCCreatePaymentResponse{}, fmt.Errorf("snap: transfer to otc create payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return TransferToOTCCreatePaymentResponse{}, errors.New("snap: transfer to otc create payment: response has no responseCode")
	}
	return resp, nil
}
