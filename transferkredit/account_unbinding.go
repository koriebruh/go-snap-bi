package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// AccountUnbindingRequest is the request body for API Account Unbinding
// (Service Code 09, path .../{version}/registration-account-unbinding).
// MerchantID is the only mandatory field per the Guides tab; LinkID and
// TokenID — the fields that would actually identify which binding to
// remove — are both Optional. The Guides tab's M/O split is the standard's
// own ambiguity here, not a gap this package fills: this type does not
// enforce that at least one of LinkID/TokenID is present, matching this
// package's established stance (see BalanceInquiryRequest) that the
// server validates business rules the wire shape alone doesn't capture.
type AccountUnbindingRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	LinkID             string          `json:"linkId,omitempty"`
	MerchantID         string          `json:"merchantId"`
	SubMerchantID      string          `json:"subMerchantId,omitempty"`
	TokenID            string          `json:"tokenId,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountUnbindingResponse is the response body for API Account Unbinding.
// UnlinkResult is typed string, not a bool: the sample value ("success")
// reads status-like, but the Guides tab lists no enumerated values.
type AccountUnbindingResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	MerchantID         string          `json:"merchantId,omitempty"`
	SubMerchantID      string          `json:"subMerchantId,omitempty"`
	LinkID             string          `json:"linkId,omitempty"`
	UnlinkResult       string          `json:"unlinkResult,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountUnbinding calls the SNAP Account Unbinding endpoint (Service Code
// 09). hb must already carry every field snap.HeaderBuilder needs except Body,
// which AccountUnbinding sets itself so the exact marshaled bytes are used
// for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func AccountUnbinding(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req AccountUnbindingRequest) (AccountUnbindingResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AccountUnbindingResponse{}, fmt.Errorf("snap: account unbinding: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AccountUnbindingResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AccountUnbindingResponse{}, fmt.Errorf("snap: account unbinding: %w", err)
	}

	var resp AccountUnbindingResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AccountUnbindingResponse{}, fmt.Errorf("snap: account unbinding: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AccountUnbindingResponse{}, errors.New("snap: account unbinding: response has no responseCode")
	}
	return resp, nil
}
