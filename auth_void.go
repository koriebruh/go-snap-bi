package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// AuthVoidRequest is the request body for API Void (Service Code 67,
// path .../{version}/auth/void). Releases held-but-uncaptured funds
// from a hold previously placed by Auth Payment (63).
//
// VoidRemainingAmount is documented String(8) but carries the
// worked-example value "TRUE", same convention as
// AuthCaptureRequest.LastCapture (research §6 item 6) — modeled as
// plain string.
type AuthVoidRequest struct {
	OriginalReferenceNo        string          `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo"`
	MerchantID                 string          `json:"merchantId"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	VoidAmount                 *Money          `json:"voidAmount,omitempty"`
	PartnerVoidNo              string          `json:"partnerVoidNo"`
	VoidRemainingAmount        string          `json:"voidRemainingAmount,omitempty"`
	Reason                     string          `json:"reason,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthVoidResponse is the response body for API Void. VoidNo and
// VoidTime are Conditional (success only). PartnerVoidNo and
// VoidAmount are Mandatory here — research states both plainly, no
// unmarked-container ambiguity (unlike AuthCaptureResponse.CaptureAmount).
type AuthVoidResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	VoidNo                     string          `json:"voidNo,omitempty"`
	PartnerVoidNo              string          `json:"partnerVoidNo"`
	VoidAmount                 Money           `json:"voidAmount"`
	VoidTime                   string          `json:"voidTime,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthVoid calls the SNAP Void endpoint (Service Code 67, HTTP POST —
// no method override). hb must already carry every field
// HeaderBuilder needs except Body, which AuthVoid sets itself so the
// exact marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func AuthVoid(ctx context.Context, t *Transport, hb HeaderBuilder, req AuthVoidRequest) (AuthVoidResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AuthVoidResponse{}, fmt.Errorf("snap: auth void: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AuthVoidResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AuthVoidResponse{}, fmt.Errorf("snap: auth void: %w", err)
	}

	var resp AuthVoidResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AuthVoidResponse{}, fmt.Errorf("snap: auth void: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AuthVoidResponse{}, errors.New("snap: auth void: response has no responseCode")
	}
	return resp, nil
}
