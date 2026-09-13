package transferdebit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// AuthVoidQueryRequest is the request body for API Void Query (Service
// Code 68, path .../{version}/auth/void-query). OriginalReferenceNo,
// MerchantID, and PartnerVoidNo are Mandatory.
type AuthVoidQueryRequest struct {
	OriginalReferenceNo        string          `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	MerchantID                 string          `json:"merchantId"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	VoidNo                     string          `json:"voidNo,omitempty"`
	PartnerVoidNo              string          `json:"partnerVoidNo"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthVoidQueryResponse is the response body for API Void Query.
// VoidAmount is Mandatory; PartnerVoidNo is Optional here — unlike
// AuthVoidResponse, where PartnerVoidNo is Mandatory. LatestVoidStatus
// is the shared 3-value enum (research §3: INIT/SUCCESS/FAILED).
type AuthVoidQueryResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	VoidNo                     string          `json:"voidNo,omitempty"`
	VoidAmount                 snap.Money      `json:"voidAmount"`
	VoidTime                   string          `json:"voidTime,omitempty"`
	LatestVoidStatus           string          `json:"latestVoidStatus,omitempty"`
	PartnerVoidNo              string          `json:"partnerVoidNo,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthVoidQuery calls the SNAP Void Query endpoint (Service Code 68,
// HTTP POST — no method override; research's Overview tab marks this
// GET, but every worked example across all 7 Auth Payment endpoints is
// POST-shaped with a mandatory JSON body, and this package's signing
// path has no representation for a body-carrying GET — see
// AuthPayment's doc comment for the full reasoning). hb must already carry every field
// snap.HeaderBuilder needs except Body, which AuthVoidQuery sets itself so
// the exact marshaled bytes are used for both signing and the wire
// body.
func AuthVoidQuery(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req AuthVoidQueryRequest) (AuthVoidQueryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AuthVoidQueryResponse{}, fmt.Errorf("snap: auth void query: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AuthVoidQueryResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AuthVoidQueryResponse{}, fmt.Errorf("snap: auth void query: %w", err)
	}

	var resp AuthVoidQueryResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AuthVoidQueryResponse{}, fmt.Errorf("snap: auth void query: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AuthVoidQueryResponse{}, errors.New("snap: auth void query: response has no responseCode")
	}
	return resp, nil
}
