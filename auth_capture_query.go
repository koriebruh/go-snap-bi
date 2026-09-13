package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// AuthCaptureQueryRequest is the request body for API Capture Query
// (Service Code 66, path .../{version}/auth/capture-query).
// OriginalReferenceNo, MerchantID, and PartnerCaptureNo are Mandatory.
type AuthCaptureQueryRequest struct {
	OriginalReferenceNo        string          `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	MerchantID                 string          `json:"merchantId"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	CaptureNo                  string          `json:"captureNo,omitempty"`
	PartnerCaptureNo           string          `json:"partnerCaptureNo"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthCaptureQueryResponse is the response body for API Capture Query.
// CaptureAmount and PartnerCaptureNo are Mandatory here — unlike
// AuthCaptureResponse, where PartnerCaptureNo is Optional.
// LatestCaptureStatus is the shared 3-value enum (research §3:
// INIT/SUCCESS/FAILED).
type AuthCaptureQueryResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	CaptureNo                  string          `json:"captureNo,omitempty"`
	CaptureAmount              Money           `json:"captureAmount"`
	CaptureTime                string          `json:"captureTime,omitempty"`
	LatestCaptureStatus        string          `json:"latestCaptureStatus,omitempty"`
	PartnerCaptureNo           string          `json:"partnerCaptureNo"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthCaptureQuery calls the SNAP Capture Query endpoint (Service Code
// 66, HTTP POST — no method override). hb must already carry every
// field HeaderBuilder needs except Body, which AuthCaptureQuery sets
// itself so the exact marshaled bytes are used for both signing and
// the wire body.
func AuthCaptureQuery(ctx context.Context, t *Transport, hb HeaderBuilder, req AuthCaptureQueryRequest) (AuthCaptureQueryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AuthCaptureQueryResponse{}, fmt.Errorf("snap: auth capture query: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AuthCaptureQueryResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AuthCaptureQueryResponse{}, fmt.Errorf("snap: auth capture query: %w", err)
	}

	var resp AuthCaptureQueryResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AuthCaptureQueryResponse{}, fmt.Errorf("snap: auth capture query: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AuthCaptureQueryResponse{}, errors.New("snap: auth capture query: response has no responseCode")
	}
	return resp, nil
}
