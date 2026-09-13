package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// AuthRefundRequest is the request body for API Refund (Service Code
// 69, path .../{version}/auth/refund). Reverses an amount already
// captured by Auth Capture (65).
//
// OriginalCaptureNo is Conditional, "must be filled upon unsuccessful
// transaction" — the opposite condition-sense from most Conditional
// fields in this package, which are typically filled on success
// (research §5.3). Recorded here verbatim, not adjusted; this package
// enforces no field-level business-rule validation, so the direction
// of the trigger has no code consequence beyond this note.
type AuthRefundRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	PartnerRefundNo            string          `json:"partnerRefundNo"`
	MerchantID                 string          `json:"merchantId,omitempty"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	OriginalCaptureNo          string          `json:"originalCaptureNo,omitempty"`
	RefundAmount               *Money          `json:"refundAmount,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	Reason                     string          `json:"reason,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthRefundResponse is the response body for API Refund. RefundNo and
// RefundTime are Mandatory. PartnerRefundNo is Optional here — unlike
// Direct Debit Payment Refund's own response, where PartnerRefundNo is
// Mandatory (Phase 27) — recorded per-occurrence, not harmonized.
//
// OriginalCaptureNo and OriginalReferenceNo are both Conditional, but
// with opposite success/failure triggers from each other:
// OriginalCaptureNo is filled on an unsuccessful transaction (same
// condition as the request field of the same name), while
// OriginalReferenceNo is filled on a successful one. Research §6 item
// 3 flags this as not yet independently verified — it reads as a
// genuine distinction rather than a copy-paste duplicate of the same
// note, but research itself leaves it open pending confirmation. This
// package models the wire shape as documented either way (both fields
// Conditional → omitempty), since it enforces no field-level
// business-rule validation regardless of which direction the trigger
// actually runs.
type AuthRefundResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalCaptureNo          string          `json:"originalCaptureNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	PartnerRefundNo            string          `json:"partnerRefundNo,omitempty"`
	RefundNo                   string          `json:"refundNo"`
	RefundAmount               *Money          `json:"refundAmount,omitempty"`
	RefundTime                 string          `json:"refundTime"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthRefund calls the SNAP Refund endpoint (Service Code 69, HTTP
// POST — no method override, same GET/POST resolution as AuthPayment,
// Phase 31). hb must already carry every field HeaderBuilder needs
// except Body, which AuthRefund sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func AuthRefund(ctx context.Context, t *Transport, hb HeaderBuilder, req AuthRefundRequest) (AuthRefundResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AuthRefundResponse{}, fmt.Errorf("snap: auth refund: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AuthRefundResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AuthRefundResponse{}, fmt.Errorf("snap: auth refund: %w", err)
	}

	var resp AuthRefundResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AuthRefundResponse{}, fmt.Errorf("snap: auth refund: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AuthRefundResponse{}, errors.New("snap: auth refund: response has no responseCode")
	}
	return resp, nil
}
