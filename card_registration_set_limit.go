package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CardRegistrationSetLimitRequest is the request body for API Card
// Registration – Set Limit (Service Code 02, path
// .../{version}/registration/card-bind-limit — note the "/" where sibling
// endpoints use "-"; this is the standard's own path). BankCardToken is
// the only mandatory field per the Guides tab.
//
// Limit is typed json.RawMessage for the same reason as
// CardRegistrationRequest.Limit: the Guides tab labels it "decimal", which
// permits a non-string JSON representation even though the portal's one
// worked example renders it as a quoted string. A caller assigning a
// plain Go string must quote it first — e.g. json.RawMessage(`"1000000"`),
// not json.RawMessage(limitStr) — to match the worked example's wire
// shape. Getting this wrong is not silent: a formatted decimal like
// "1,000,000" is not valid JSON on its own, so an unquoted assignment
// fails json.Marshal and CardRegistrationSetLimit returns an "encode
// request" error before any request is sent. Two other shapes are worth
// knowing: an empty json.RawMessage is dropped by omitempty (the field is
// absent from the wire body, not sent empty), while json.RawMessage("null")
// is NOT dropped — it is sent as an explicit "limit":null, since
// omitempty only skips a zero-length slice.
type CardRegistrationSetLimitRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	BankAccountNo      string          `json:"bankAccountNo,omitempty"`
	BankCardNo         string          `json:"bankCardNo,omitempty"`
	Limit              json.RawMessage `json:"limit,omitempty"`
	BankCardToken      string          `json:"bankCardToken"`
	OTP                string          `json:"otp,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// CardRegistrationSetLimitResponse is the response body for API Card
// Registration – Set Limit.
type CardRegistrationSetLimitResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// CardRegistrationSetLimit calls the SNAP Card Registration – Set Limit
// endpoint (Service Code 02). hb must already carry every field
// HeaderBuilder needs except Body, which CardRegistrationSetLimit sets
// itself so the exact marshaled bytes are used for both signing and the
// wire body.
func CardRegistrationSetLimit(ctx context.Context, t *Transport, hb HeaderBuilder, req CardRegistrationSetLimitRequest) (CardRegistrationSetLimitResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CardRegistrationSetLimitResponse{}, fmt.Errorf("snap: card registration set limit: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CardRegistrationSetLimitResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CardRegistrationSetLimitResponse{}, fmt.Errorf("snap: card registration set limit: %w", err)
	}

	var resp CardRegistrationSetLimitResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CardRegistrationSetLimitResponse{}, fmt.Errorf("snap: card registration set limit: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CardRegistrationSetLimitResponse{}, errors.New("snap: card registration set limit: response has no responseCode")
	}
	return resp, nil
}
