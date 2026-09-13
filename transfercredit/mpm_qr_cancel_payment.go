package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// QRMPMCancelPaymentRequest is the request body for API Cancel Payment
// (Service Code 77). Unlike the package's other originalX-pattern
// endpoints, this row documents no serviceCode field and all three
// originalX fields as Optional — MerchantID and Reason are the only
// Mandatory fields.
type QRMPMCancelPaymentRequest struct {
	OriginalPartnerReferenceNo string      `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string      `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string      `json:"originalExternalId,omitempty"`
	MerchantID                 string      `json:"merchantId"`
	SubMerchantID              string      `json:"subMerchantId,omitempty"`
	ExternalStoreID            string      `json:"externalStoreId,omitempty"`
	Reason                     string      `json:"reason"`
	Amount                     *snap.Money `json:"amount,omitempty"`
}

// QRMPMCancelPaymentResponse is the response body for API Cancel
// Payment. CancelTime is Conditional per research §5.9 line 204 (no
// further condition given in that row, unlike Transfer To OTC Cancel
// Payment's documented "must be filled if cancelled transaction
// success", Phase 20) — modeled Optional, matching the package's
// standard handling of Conditional fields. TransactionDate is Optional.
type QRMPMCancelPaymentResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	CancelTime      string `json:"cancelTime,omitempty"`
	TransactionDate string `json:"transactionDate,omitempty"`
}

// QRMPMCancelPayment calls the SNAP Cancel Payment endpoint (Service
// Code 77, path .../{version}/qr/qr-mpm-cancel, HTTP POST — no method
// override). hb must already carry every field snap.HeaderBuilder needs
// except Body, which QRMPMCancelPayment sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func QRMPMCancelPayment(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req QRMPMCancelPaymentRequest) (QRMPMCancelPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return QRMPMCancelPaymentResponse{}, fmt.Errorf("snap: qr mpm cancel payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return QRMPMCancelPaymentResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return QRMPMCancelPaymentResponse{}, fmt.Errorf("snap: qr mpm cancel payment: %w", err)
	}

	var resp QRMPMCancelPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return QRMPMCancelPaymentResponse{}, fmt.Errorf("snap: qr mpm cancel payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return QRMPMCancelPaymentResponse{}, errors.New("snap: qr mpm cancel payment: response has no responseCode")
	}
	return resp, nil
}
