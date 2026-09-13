package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// QRMPMPaymentH2HRequest is the request body for API Payment - Host to
// Host (Service Code 50). PartnerReferenceNo is Mandatory; every other
// field is Optional per the Guides tab.
//
// VerificationID also appears on QRMPMPaymentH2HResponse under the same
// name but a different documented length (String(32) here, String(64)
// on the response) — both are plain string in Go (no length
// enforcement, matching package practice); this is a length note, not
// a type-shape difference, recorded so it isn't mistaken for a typo.
type QRMPMPaymentH2HRequest struct {
	PartnerReferenceNo string      `json:"partnerReferenceNo"`
	MerchantID         string      `json:"merchantId,omitempty"`
	SubMerchantID      string      `json:"subMerchantId,omitempty"`
	Amount             *snap.Money `json:"amount,omitempty"`
	FeeAmount          *snap.Money `json:"feeAmount,omitempty"`
	OTP                string      `json:"otp,omitempty"`
	VerificationID     string      `json:"verificationId,omitempty"`
}

// QRMPMPaymentH2HResponse is the response body for API Payment - Host
// to Host.
//
// The base ReferenceNo/TransactionDate shape is an interpretive
// modeling choice, not a literal transcription: research documents this
// response only as "adds verificationId String(64) O" without naming
// what it adds to. See the Phase 22 design doc for the full reasoning.
type QRMPMPaymentH2HResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	ReferenceNo     string `json:"referenceNo,omitempty"`
	TransactionDate string `json:"transactionDate,omitempty"`
	VerificationID  string `json:"verificationId,omitempty"`
}

// QRMPMPaymentH2H calls the SNAP Payment - Host to Host endpoint
// (Service Code 50, path .../{version}/qr/qr-mpm-payment, HTTP POST —
// no method override). hb must already carry every field snap.HeaderBuilder
// needs except Body, which QRMPMPaymentH2H sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func QRMPMPaymentH2H(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req QRMPMPaymentH2HRequest) (QRMPMPaymentH2HResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return QRMPMPaymentH2HResponse{}, fmt.Errorf("snap: qr mpm payment h2h: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return QRMPMPaymentH2HResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return QRMPMPaymentH2HResponse{}, fmt.Errorf("snap: qr mpm payment h2h: %w", err)
	}

	var resp QRMPMPaymentH2HResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return QRMPMPaymentH2HResponse{}, fmt.Errorf("snap: qr mpm payment h2h: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return QRMPMPaymentH2HResponse{}, errors.New("snap: qr mpm payment h2h: response has no responseCode")
	}
	return resp, nil
}
