package transferdebit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// DirectDebitBIFASTPaymentRequest is the request body for API Trigger
// Direct Debit Transfer / Payment (Service Code 71). SourceAccountNo
// is documented String(34) in this endpoint's own field table, unlike
// Registrasi e-Mandate's String(19) for what is presumably the same
// logical field (research §6 item 4) — this package enforces no
// field-length validation anywhere, so the discrepancy has no code
// consequence.
type DirectDebitBIFASTPaymentRequest struct {
	PartnerReferenceNo     string          `json:"partnerReferenceNo"`
	Currency               string          `json:"currency,omitempty"`
	CustomerReference      string          `json:"customerReference"`
	FeeType                string          `json:"feeType,omitempty"`
	Remark                 string          `json:"remark,omitempty"`
	BeneficiaryAccountNo   string          `json:"beneficiaryAccountNo"`
	BeneficiaryAccountName string          `json:"beneficiaryAccountName"`
	TransactionDate        string          `json:"transactionDate"`
	BankCode               string          `json:"bankCode"`
	SourceAccountNo        string          `json:"sourceAccountNo"`
	SourceAccountName      string          `json:"sourceAccountName"`
	Amount                 *snap.Money     `json:"amount,omitempty"`
	EMandateReffID         string          `json:"eMandateReffId"`
	AdditionalInfo         json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitBIFASTPaymentResponse is the response body for API
// Trigger Direct Debit Transfer / Payment. ReferenceNo is Conditional
// (success only). No field is Mandatory beyond the envelope.
type DirectDebitBIFASTPaymentResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitBIFASTPayment calls the SNAP Trigger Direct Debit
// Transfer / Payment endpoint (Service Code 71, path
// .../{version}/debit/fast-payment, HTTP POST — no method override).
// hb must already carry every field snap.HeaderBuilder needs except Body,
// which DirectDebitBIFASTPayment sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func DirectDebitBIFASTPayment(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req DirectDebitBIFASTPaymentRequest) (DirectDebitBIFASTPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return DirectDebitBIFASTPaymentResponse{}, fmt.Errorf("snap: direct debit bifast payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return DirectDebitBIFASTPaymentResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return DirectDebitBIFASTPaymentResponse{}, fmt.Errorf("snap: direct debit bifast payment: %w", err)
	}

	var resp DirectDebitBIFASTPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return DirectDebitBIFASTPaymentResponse{}, fmt.Errorf("snap: direct debit bifast payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return DirectDebitBIFASTPaymentResponse{}, errors.New("snap: direct debit bifast payment: response has no responseCode")
	}
	return resp, nil
}
