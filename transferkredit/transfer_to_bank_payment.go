package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// TransferToBankPaymentRequest is the request body for API Transfer To
// Bank - Payment Transaction (Service Code 43). PartnerReferenceNo,
// CustomerNumber, BeneficiaryAccountNumber, and Amount are mandatory
// per the Guides tab. CustomerNumber's JSON tag is the standard
// lowercase "customerNumber" here — this endpoint's own field listing
// uses the normal casing, unlike endpoint 42's "CustomerNumber" (see
// TransferToBankAccountInquiryRequest's doc comment).
type TransferToBankPaymentRequest struct {
	PartnerReferenceNo       string     `json:"partnerReferenceNo"`
	CustomerNumber           string     `json:"customerNumber"`
	AccountType              string     `json:"accountType,omitempty"`
	BeneficiaryAccountNumber string     `json:"beneficiaryAccountNumber"`
	BeneficiaryBankCode      string     `json:"beneficiaryBankCode,omitempty"`
	Amount                   snap.Money `json:"amount"`
	SessionID                string     `json:"sessionId,omitempty"`
	FeeType                  string     `json:"feeType,omitempty"`
}

// TransferToBankPaymentResponse is the response body for API Transfer
// To Bank - Payment Transaction. ReferenceNo and ReferenceNumber are
// both present and distinct — research §5.7 explicitly notes
// ReferenceNumber "is a distinct field from referenceNo, both
// present," the same coexistence pattern already established for
// CustomerTopUpResponse (Phase 17).
type TransferToBankPaymentResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	ReferenceNo     string `json:"referenceNo,omitempty"`
	TransactionDate string `json:"transactionDate,omitempty"`
	ReferenceNumber string `json:"referenceNumber"`
}

// TransferToBankPayment calls the SNAP Transfer To Bank - Payment
// Transaction endpoint (Service Code 43, path
// .../{version}/emoney/transfer-bank, HTTP POST — no method override).
// hb must already carry every field snap.HeaderBuilder needs except Body,
// which TransferToBankPayment sets itself so the exact marshaled bytes
// are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func TransferToBankPayment(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req TransferToBankPaymentRequest) (TransferToBankPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TransferToBankPaymentResponse{}, fmt.Errorf("snap: transfer to bank payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return TransferToBankPaymentResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return TransferToBankPaymentResponse{}, fmt.Errorf("snap: transfer to bank payment: %w", err)
	}

	var resp TransferToBankPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return TransferToBankPaymentResponse{}, fmt.Errorf("snap: transfer to bank payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return TransferToBankPaymentResponse{}, errors.New("snap: transfer to bank payment: response has no responseCode")
	}
	return resp, nil
}
