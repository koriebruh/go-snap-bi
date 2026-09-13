package transferdebit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// DirectDebitPaymentRefundRequest is the request body for API Direct
// Debit Payment Refund (Service Code 58). OriginalPartnerReferenceNo
// and PartnerRefundNo are Mandatory per research §5.1.
type DirectDebitPaymentRefundRequest struct {
	MerchantID                 string          `json:"merchantId,omitempty"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	PartnerRefundNo            string          `json:"partnerRefundNo"`
	RefundAmount               *snap.Money     `json:"refundAmount,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	Reason                     string          `json:"reason,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitPaymentRefundResponse is the response body for API Direct
// Debit Payment Refund. OriginalReferenceNo is Conditional (success
// only). RefundNo, PartnerRefundNo, and RefundTime are Mandatory per
// research §5.1 — not a byte-for-byte copy of
// QRMPMRefundPaymentResponse, which has no Mandatory marker on
// PartnerRefundNo; independently derived from this endpoint's own
// field table.
type DirectDebitPaymentRefundResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	PartnerTrxID               string          `json:"partnerTrxId,omitempty"`
	RefundNo                   string          `json:"refundNo"`
	PartnerRefundNo            string          `json:"partnerRefundNo"`
	RefundAmount               *snap.Money     `json:"refundAmount,omitempty"`
	RefundTime                 string          `json:"refundTime"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitPaymentRefund calls the SNAP Direct Debit Payment Refund
// endpoint (Service Code 58, path .../{version}/debit/refund, HTTP
// POST — no method override). hb must already carry every field
// snap.HeaderBuilder needs except Body, which DirectDebitPaymentRefund sets
// itself so the exact marshaled bytes are used for both signing and
// the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func DirectDebitPaymentRefund(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req DirectDebitPaymentRefundRequest) (DirectDebitPaymentRefundResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return DirectDebitPaymentRefundResponse{}, fmt.Errorf("snap: direct debit payment refund: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return DirectDebitPaymentRefundResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return DirectDebitPaymentRefundResponse{}, fmt.Errorf("snap: direct debit payment refund: %w", err)
	}

	var resp DirectDebitPaymentRefundResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return DirectDebitPaymentRefundResponse{}, fmt.Errorf("snap: direct debit payment refund: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return DirectDebitPaymentRefundResponse{}, errors.New("snap: direct debit payment refund: response has no responseCode")
	}
	return resp, nil
}
