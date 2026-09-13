package transferdebit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// CPMRefundPaymentRequest is the request body for API Refund Payment
// (Service Code 80). OriginalPartnerReferenceNo and PartnerRefundNo
// are Mandatory per research §5.2.
type CPMRefundPaymentRequest struct {
	MerchantID                 string          `json:"merchantId,omitempty"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	PartnerRefundNo            string          `json:"partnerRefundNo"`
	RefundAmount               *snap.Money     `json:"refundAmount,omitempty"`
	Reason                     string          `json:"reason,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMRefundPaymentResponse is the response body for API Refund
// Payment. RefundNo and RefundTime are Mandatory per research §5.2.
// OriginalReferenceNo carries no C/M marker in this endpoint's own
// research row (unlike every other occurrence of this field in the
// package) — modeled Optional per the package's default-to-omitempty
// handling for an unmarked field, recorded as shown rather than
// guessed. PartnerRefundNo is Optional here in the response, unlike
// Direct Debit Payment Refund's Mandatory PartnerRefundNo in its own
// response (Phase 27) — independently derived from this endpoint's
// own field table, not copied.
type CPMRefundPaymentResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	RefundNo                   string          `json:"refundNo"`
	PartnerRefundNo            string          `json:"partnerRefundNo,omitempty"`
	RefundAmount               *snap.Money     `json:"refundAmount,omitempty"`
	RefundTime                 string          `json:"refundTime"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMRefundPayment calls the SNAP Refund Payment endpoint (Service
// Code 80, path .../{version}/qr/qr-cpm-refund, HTTP POST — no method
// override). hb must already carry every field snap.HeaderBuilder needs
// except Body, which CPMRefundPayment sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func CPMRefundPayment(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req CPMRefundPaymentRequest) (CPMRefundPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CPMRefundPaymentResponse{}, fmt.Errorf("snap: cpm refund payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CPMRefundPaymentResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CPMRefundPaymentResponse{}, fmt.Errorf("snap: cpm refund payment: %w", err)
	}

	var resp CPMRefundPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CPMRefundPaymentResponse{}, fmt.Errorf("snap: cpm refund payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CPMRefundPaymentResponse{}, errors.New("snap: cpm refund payment: response has no responseCode")
	}
	return resp, nil
}
