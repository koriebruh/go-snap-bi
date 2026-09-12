package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// QRMPMRefundPaymentRequest is the request body for API Refund Payment
// (Service Code 78). OriginalPartnerReferenceNo and PartnerRefundNo are
// Mandatory; every other field is Optional per the Guides tab.
type QRMPMRefundPaymentRequest struct {
	MerchantID                 string `json:"merchantId,omitempty"`
	SubMerchantID              string `json:"subMerchantId,omitempty"`
	ExternalStoreID            string `json:"externalStoreId,omitempty"`
	OriginalPartnerReferenceNo string `json:"originalPartnerReferenceNo"`
	OriginalReferenceNo        string `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string `json:"originalExternalId,omitempty"`
	PartnerRefundNo            string `json:"partnerRefundNo"`
	RefundAmount               *Money `json:"refundAmount,omitempty"`
	Reason                     string `json:"reason,omitempty"`
}

// QRMPMRefundPaymentResponse is the response body for API Refund
// Payment. RefundNo and RefundTime are Mandatory; PartnerRefundNo and
// RefundAmount are Optional (echoed back, not guaranteed).
type QRMPMRefundPaymentResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	RefundNo        string `json:"refundNo"`
	PartnerRefundNo string `json:"partnerRefundNo,omitempty"`
	RefundAmount    *Money `json:"refundAmount,omitempty"`
	RefundTime      string `json:"refundTime"`
}

// QRMPMRefundPayment calls the SNAP Refund Payment endpoint (Service
// Code 78, path .../{version}/qr/qr-mpm-refund, HTTP POST — no method
// override). hb must already carry every field HeaderBuilder needs
// except Body, which QRMPMRefundPayment sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func QRMPMRefundPayment(ctx context.Context, t *Transport, hb HeaderBuilder, req QRMPMRefundPaymentRequest) (QRMPMRefundPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return QRMPMRefundPaymentResponse{}, fmt.Errorf("snap: qr mpm refund payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return QRMPMRefundPaymentResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return QRMPMRefundPaymentResponse{}, fmt.Errorf("snap: qr mpm refund payment: %w", err)
	}

	var resp QRMPMRefundPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return QRMPMRefundPaymentResponse{}, fmt.Errorf("snap: qr mpm refund payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return QRMPMRefundPaymentResponse{}, errors.New("snap: qr mpm refund payment: response has no responseCode")
	}
	return resp, nil
}
