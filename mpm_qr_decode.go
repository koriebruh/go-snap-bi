package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// MPMMerchantInfo is one entry in a DecodeQRMPMResponse's merchantInfos
// array.
//
// MerchantPAN is documented Numeric(19) but quoted as a string on the
// wire (research §5.9 line 194) — the same ambiguous-type shape as
// customerNo (Phase 14/15) and CustomerMonthlyInLimit (Phase 17), so it
// is json.RawMessage rather than string or a numeric Go type.
type MPMMerchantInfo struct {
	MerchantPAN  json.RawMessage `json:"merchantPAN"`
	AcquirerName string          `json:"acquirerName"`
}

// DecodeQRMPMRequest is the request body for API Decode QR MPM
// (Service Code 48). QRContent and ScanTime are Mandatory per the
// Guides tab.
type DecodeQRMPMRequest struct {
	PartnerReferenceNo string `json:"partnerReferenceNo,omitempty"`
	QRContent          string `json:"qrContent"`
	Amount             *Money `json:"amount,omitempty"`
	MerchantID         string `json:"merchantId,omitempty"`
	SubMerchantID      string `json:"subMerchantId,omitempty"`
	ScanTime           string `json:"scanTime"`
}

// DecodeQRMPMResponse is the response body for API Decode QR MPM.
//
// ReferenceNo and RedirectURL carry contradictory conditional labels in
// the source documentation — "Mandatory if redirect" and "Mandatory if
// H2H mode" respectively, which read as opposite branches of the same
// condition (research §6 item 8, unresolved). Both are modeled Optional;
// neither reading is asserted here.
type DecodeQRMPMResponse struct {
	ResponseCode      string            `json:"responseCode"`
	ResponseMessage   string            `json:"responseMessage"`
	ReferenceNo       string            `json:"referenceNo,omitempty"`
	RedirectURL       string            `json:"redirectUrl,omitempty"`
	MerchantName      string            `json:"merchantName,omitempty"`
	MerchantCategory  string            `json:"merchantCategory,omitempty"`
	MerchantLocation  string            `json:"merchantLocation,omitempty"`
	MerchantInfos     []MPMMerchantInfo `json:"merchantInfos"`
	TransactionAmount *Money            `json:"transactionAmount,omitempty"`
	FeeAmount         *Money            `json:"feeAmount,omitempty"`
}

// DecodeQRMPM calls the SNAP Decode QR MPM endpoint (Service Code 48,
// path .../{version}/qr/qr-mpm-decode, HTTP POST — no method override).
// hb must already carry every field HeaderBuilder needs except Body,
// which DecodeQRMPM sets itself so the exact marshaled bytes are used
// for both signing and the wire body.
//
// This is a read-only decode; unlike GenerateQRMPM it carries no
// non-idempotency note.
func DecodeQRMPM(ctx context.Context, t *Transport, hb HeaderBuilder, req DecodeQRMPMRequest) (DecodeQRMPMResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return DecodeQRMPMResponse{}, fmt.Errorf("snap: decode qr mpm: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return DecodeQRMPMResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return DecodeQRMPMResponse{}, fmt.Errorf("snap: decode qr mpm: %w", err)
	}

	var resp DecodeQRMPMResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return DecodeQRMPMResponse{}, fmt.Errorf("snap: decode qr mpm: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return DecodeQRMPMResponse{}, errors.New("snap: decode qr mpm: response has no responseCode")
	}
	return resp, nil
}
