package transferdebit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// DirectDebitPaymentCancelRequest is the request body for API Direct
// Debit Payment Cancel (Service Code 57). OriginalPartnerReferenceNo
// is the only Mandatory field per research §5.1.
type DirectDebitPaymentCancelRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	ApprovalCode               string          `json:"approvalCode,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	MerchantID                 string          `json:"merchantId,omitempty"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	Reason                     string          `json:"reason,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	Amount                     *snap.Money     `json:"amount,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitPaymentCancelResponse is the response body for API Direct
// Debit Payment Cancel. OriginalReferenceNo is Conditional (success
// only). CancelTime is Conditional per research §5.1 ("required if
// successful"), matching QRMPMCancelPaymentResponse.CancelTime's
// identical treatment (Phase 24). No field is Mandatory beyond the
// envelope.
type DirectDebitPaymentCancelResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	CancelTime                 string          `json:"cancelTime,omitempty"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitPaymentCancel calls the SNAP Direct Debit Payment Cancel
// endpoint (Service Code 57, path .../{version}/debit/cancel, HTTP
// POST — no method override). hb must already carry every field
// snap.HeaderBuilder needs except Body, which DirectDebitPaymentCancel sets
// itself so the exact marshaled bytes are used for both signing and
// the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func DirectDebitPaymentCancel(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req DirectDebitPaymentCancelRequest) (DirectDebitPaymentCancelResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return DirectDebitPaymentCancelResponse{}, fmt.Errorf("snap: direct debit payment cancel: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return DirectDebitPaymentCancelResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return DirectDebitPaymentCancelResponse{}, fmt.Errorf("snap: direct debit payment cancel: %w", err)
	}

	var resp DirectDebitPaymentCancelResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return DirectDebitPaymentCancelResponse{}, fmt.Errorf("snap: direct debit payment cancel: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return DirectDebitPaymentCancelResponse{}, errors.New("snap: direct debit payment cancel: response has no responseCode")
	}
	return resp, nil
}
