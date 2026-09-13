package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CPMGenerateQRRequest is the request body for API Generate QR CPM
// (Service Code 59). PartnerTrxDate is the only Mandatory field per
// research §5.2.
type CPMGenerateQRRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	UserAccessToken    string          `json:"userAccessToken,omitempty"`
	MerchantID         string          `json:"merchantId,omitempty"`
	SubMerchantID      string          `json:"subMerchantId,omitempty"`
	PartnerTrxDate     string          `json:"partnerTrxDate"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMGenerateQRResponse is the response body for API Generate QR CPM.
// ExpiryTime is the only Mandatory field beyond the envelope. Unlike
// GenerateQRMPMResponse (Transfer Kredit, Service 47), QRContent/QRURL
// here carry no one-of-three conditional rule — both are plain
// Optional, and there is no QRImage field in this sub-group at all;
// this is independently derived from this endpoint's own field table,
// not a copy of that sibling.
type CPMGenerateQRResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	QRContent          string          `json:"qrContent,omitempty"`
	QRURL              string          `json:"qrUrl,omitempty"`
	ExpiryTime         string          `json:"expiryTime"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMGenerateQR calls the SNAP Generate QR CPM endpoint (Service Code
// 59, path .../{version}/qr/qr-cpm-generate, HTTP POST — no method
// override). hb must already carry every field HeaderBuilder needs
// except Body, which CPMGenerateQR sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func CPMGenerateQR(ctx context.Context, t *Transport, hb HeaderBuilder, req CPMGenerateQRRequest) (CPMGenerateQRResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CPMGenerateQRResponse{}, fmt.Errorf("snap: cpm generate qr: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CPMGenerateQRResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CPMGenerateQRResponse{}, fmt.Errorf("snap: cpm generate qr: %w", err)
	}

	var resp CPMGenerateQRResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CPMGenerateQRResponse{}, fmt.Errorf("snap: cpm generate qr: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CPMGenerateQRResponse{}, errors.New("snap: cpm generate qr: response has no responseCode")
	}
	return resp, nil
}
