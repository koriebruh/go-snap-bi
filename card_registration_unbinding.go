package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CardRegistrationUnbindingRequest is the request body for API Card
// Registration Unbinding (Service Code 05, path
// .../{version}/registration-card-unbind). Token is the only mandatory
// field per the Guides tab.
//
// Part is a source artifact, not reconciled: the Guides tab's
// description text for Part is identical to MerchantID's description on
// the portal, and the worked example sets both fields to the same
// value. Kept as its own field since the Guides tab lists it as a
// distinct field name.
type CardRegistrationUnbindingRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	Token              string          `json:"token"`
	BankCardNo         string          `json:"bankCardNo,omitempty"`
	Type               string          `json:"type,omitempty"`
	Part               string          `json:"part,omitempty"`
	MerchantID         string          `json:"merchantId,omitempty"`
	SubMerchantID      string          `json:"subMerchantId,omitempty"`
	TerminalID         string          `json:"terminalId,omitempty"`
	TokenRequestorID   string          `json:"tokenRequestorId,omitempty"`
	JourneyID          string          `json:"journeyId,omitempty"`
	TransactionDate    string          `json:"transactionDate,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// CardRegistrationUnbindingResponse is the response body for API Card
// Registration Unbinding. UnsubscribeDate is typed string despite being
// the only field in the researched Registrasi group labeled "Datetime"
// (every other date-like field is labeled "String"): JSON has no
// datetime primitive, and the worked example is a plain quoted ISO-8601
// string, so string is unambiguous regardless of the unique label.
type CardRegistrationUnbindingResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	Message            string          `json:"message,omitempty"`
	CustomerID         string          `json:"customerId,omitempty"`
	UnsubscribeDate    string          `json:"unsubscribeDate,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// CardRegistrationUnbinding calls the SNAP Card Registration Unbinding
// endpoint (Service Code 05). hb must already carry every field
// HeaderBuilder needs except Body, which CardRegistrationUnbinding sets
// itself so the exact marshaled bytes are used for both signing and the
// wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it
// — matching AccountUnbinding's stance, which carries the same note
// despite also minting nothing: the note is about server-side duplicate
// detection on a mutating call, not specifically about minting a new
// resource.
func CardRegistrationUnbinding(ctx context.Context, t *Transport, hb HeaderBuilder, req CardRegistrationUnbindingRequest) (CardRegistrationUnbindingResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CardRegistrationUnbindingResponse{}, fmt.Errorf("snap: card registration unbinding: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CardRegistrationUnbindingResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CardRegistrationUnbindingResponse{}, fmt.Errorf("snap: card registration unbinding: %w", err)
	}

	var resp CardRegistrationUnbindingResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CardRegistrationUnbindingResponse{}, fmt.Errorf("snap: card registration unbinding: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CardRegistrationUnbindingResponse{}, errors.New("snap: card registration unbinding: response has no responseCode")
	}
	return resp, nil
}
