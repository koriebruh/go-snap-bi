package transferdebit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// AuthCaptureRequest is the request body for API Capture (Service Code
// 65, path .../{version}/auth/capture). Charges some or all of an
// amount previously held by Auth Payment (63); may be called multiple
// times for partial captures.
//
// LastCapture is documented String(8) but carries the worked-example
// value "TRUE" — a boolean-shaped value in a string field. Research
// states the wire shape (a quoted string) is unambiguous here (§6 item
// 6), so it is modeled as plain string, not json.RawMessage.
type AuthCaptureRequest struct {
	OriginalReferenceNo        string          `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo"`
	MerchantID                 string          `json:"merchantId"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	PartnerCaptureNo           string          `json:"partnerCaptureNo"`
	CaptureAmount              *snap.Money     `json:"captureAmount,omitempty"`
	Title                      string          `json:"title"`
	LastCapture                string          `json:"lastCapture,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthCaptureResponse is the response body for API Capture. CaptureNo
// and CaptureTime are Conditional (success only). CaptureAmount's
// container is left unmarked in research alongside Mandatory members
// — the same ambiguity as AuthPaymentResponse.Amount (Phase 31) — and
// is modeled the same way, as a plain (non-pointer) snap.Money, since this
// is a success-response field.
type AuthCaptureResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	PartnerCaptureNo           string          `json:"partnerCaptureNo,omitempty"`
	CaptureNo                  string          `json:"captureNo,omitempty"`
	CaptureAmount              snap.Money      `json:"captureAmount"`
	CaptureTime                string          `json:"captureTime,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthCapture calls the SNAP Capture endpoint (Service Code 65, HTTP
// POST — no method override, same GET/POST resolution as AuthPayment,
// Phase 31). hb must already carry every field snap.HeaderBuilder needs
// except Body, which AuthCapture sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func AuthCapture(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req AuthCaptureRequest) (AuthCaptureResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AuthCaptureResponse{}, fmt.Errorf("snap: auth capture: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AuthCaptureResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AuthCaptureResponse{}, fmt.Errorf("snap: auth capture: %w", err)
	}

	var resp AuthCaptureResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AuthCaptureResponse{}, fmt.Errorf("snap: auth capture: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AuthCaptureResponse{}, errors.New("snap: auth capture: response has no responseCode")
	}
	return resp, nil
}
