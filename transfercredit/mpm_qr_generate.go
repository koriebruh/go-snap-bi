package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// GenerateQRMPMRequest is the request body for API Generate QR MPM
// (Service Code 47). Every field is Optional per the Guides tab — no
// field in this request is documented Mandatory.
type GenerateQRMPMRequest struct {
	PartnerReferenceNo string      `json:"partnerReferenceNo,omitempty"`
	Amount             *snap.Money `json:"amount,omitempty"`
	FeeAmount          *snap.Money `json:"feeAmount,omitempty"`
	MerchantID         string      `json:"merchantId,omitempty"`
	SubMerchantID      string      `json:"subMerchantId,omitempty"`
	StoreID            string      `json:"storeId,omitempty"`
	TerminalID         string      `json:"terminalId,omitempty"`
	ValidityPeriod     string      `json:"validityPeriod,omitempty"`
}

// GenerateQRMPMResponse is the response body for API Generate QR MPM.
//
// QRContent, QRURL, and QRImage form a one-of-three condition the type
// system cannot express: per the Guides tab, "if [qrContent is] null,
// qrUrl or qrImage must be filled." All three are Optional here; callers
// must check which of the three came back non-empty.
type GenerateQRMPMResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	QRContent       string `json:"qrContent,omitempty"`
	QRURL           string `json:"qrUrl,omitempty"`
	QRImage         string `json:"qrImage,omitempty"` // base64, String(unlimited) — bounded only by the shared transport read cap
	RedirectURL     string `json:"redirectUrl,omitempty"`
	MerchantName    string `json:"merchantName,omitempty"`
	StoreID         string `json:"storeId,omitempty"`
	TerminalID      string `json:"terminalId,omitempty"`
}

// GenerateQRMPM calls the SNAP Generate QR MPM endpoint (Service Code
// 47, path .../{version}/qr/qr-mpm-generate, HTTP POST — no method
// override). hb must already carry every field snap.HeaderBuilder needs
// except Body, which GenerateQRMPM sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func GenerateQRMPM(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req GenerateQRMPMRequest) (GenerateQRMPMResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return GenerateQRMPMResponse{}, fmt.Errorf("snap: generate qr mpm: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return GenerateQRMPMResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return GenerateQRMPMResponse{}, fmt.Errorf("snap: generate qr mpm: %w", err)
	}

	var resp GenerateQRMPMResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return GenerateQRMPMResponse{}, fmt.Errorf("snap: generate qr mpm: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return GenerateQRMPMResponse{}, errors.New("snap: generate qr mpm: response has no responseCode")
	}
	return resp, nil
}
