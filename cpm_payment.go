package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CPMPaymentScannerInfo is the CPM Payment request's optional
// "scannerInfo" nested object. All four members are Optional per
// research §5.2.
type CPMPaymentScannerInfo struct {
	DeviceID      string `json:"deviceId,omitempty"`
	DeviceVersion string `json:"deviceVersion,omitempty"`
	DeviceModel   string `json:"deviceModel,omitempty"`
	DeviceIP      string `json:"deviceIp,omitempty"`
}

// CPMPaymentRequest is the request body for API CPM Payment (Service
// Code 60). PartnerReferenceNo, QRContent, and MerchantID are
// Mandatory per research §5.2.
//
// Items is documented as an unstructured "Object" with no item-level
// field table given anywhere in research — modeled as json.RawMessage,
// matching the package's existing treatment of genuinely untyped
// fields (same rationale as AdditionalInfo, not a new convention).
type CPMPaymentRequest struct {
	PartnerReferenceNo string                 `json:"partnerReferenceNo"`
	QRContent          string                 `json:"qrContent"`
	Amount             *Money                 `json:"amount,omitempty"`
	FeeAmount          *Money                 `json:"feeAmount,omitempty"`
	MerchantID         string                 `json:"merchantId"`
	SubMerchantID      string                 `json:"subMerchantId,omitempty"`
	Title              string                 `json:"title,omitempty"`
	ExpiryTime         string                 `json:"expiryTime,omitempty"`
	Items              json.RawMessage        `json:"items,omitempty"`
	ExternalStoreID    string                 `json:"externalStoreId,omitempty"`
	MerchantName       string                 `json:"merchantName,omitempty"`
	MerchantLocation   string                 `json:"merchantLocation,omitempty"`
	AcquirerName       string                 `json:"acquirerName,omitempty"`
	TerminalID         string                 `json:"terminalId,omitempty"`
	ScannerInfo        *CPMPaymentScannerInfo `json:"scannerInfo,omitempty"`
	AdditionalInfo     json.RawMessage        `json:"additionalInfo,omitempty"`
}

// CPMPaymentResponse is the response body for API CPM Payment.
// ReferenceNo is Conditional (success only). No field is Mandatory
// beyond the envelope.
type CPMPaymentResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	TransactionDate    string          `json:"transactionDate,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMPayment calls the SNAP CPM Payment endpoint (Service Code 60,
// path .../{version}/qr/qr-cpm-payment, HTTP POST — no method
// override). hb must already carry every field HeaderBuilder needs
// except Body, which CPMPayment sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func CPMPayment(ctx context.Context, t *Transport, hb HeaderBuilder, req CPMPaymentRequest) (CPMPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CPMPaymentResponse{}, fmt.Errorf("snap: cpm payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CPMPaymentResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CPMPaymentResponse{}, fmt.Errorf("snap: cpm payment: %w", err)
	}

	var resp CPMPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CPMPaymentResponse{}, fmt.Errorf("snap: cpm payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CPMPaymentResponse{}, errors.New("snap: cpm payment: response has no responseCode")
	}
	return resp, nil
}
