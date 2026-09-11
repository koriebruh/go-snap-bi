package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// DeviceInfo describes the device initiating a registration request.
type DeviceInfo struct {
	OS           string `json:"os,omitempty"`
	OSVersion    string `json:"osVersion,omitempty"`
	Model        string `json:"model,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
}

// AccountCreationRequest is the request body for API Account Creation
// (Service Code 06). Every field is optional per the standard — this
// endpoint supports several onboarding flows (seamless data, OAuth
// redirect, direct creation) that each use a different subset of fields.
type AccountCreationRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	CountryCode        string          `json:"countryCode,omitempty"`
	CustomerID         string          `json:"customerId,omitempty"`
	DeviceInfo         *DeviceInfo     `json:"deviceInfo,omitempty"`
	Email              string          `json:"email,omitempty"`
	Lang               string          `json:"lang,omitempty"`
	Locale             string          `json:"locale,omitempty"`
	Name               string          `json:"name,omitempty"`
	OnboardingPartner  string          `json:"onboardingPartner,omitempty"`
	PhoneNo            string          `json:"phoneNo,omitempty"`
	RedirectURL        string          `json:"redirectUrl,omitempty"`
	Scopes             string          `json:"scopes,omitempty"`
	SeamlessData       string          `json:"seamlessData,omitempty"`
	SeamlessSign       string          `json:"seamlessSign,omitempty"`
	State              string          `json:"state,omitempty"`
	MerchantID         string          `json:"merchantId,omitempty"`
	SubMerchantID      string          `json:"subMerchantId,omitempty"`
	TerminalType       json.RawMessage `json:"terminalType,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountCreationResponse is the response body for API Account Creation.
// APIKey is typed string, not a numeric type, even though the standard's
// Guides table labels it "Numeric" — no length or range is given, and
// treating an opaque identifier as a numeric type risks silent precision
// loss or a false range assumption for what is not an arithmetic value.
type AccountCreationResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	AuthCode           string          `json:"authCode,omitempty"`
	APIKey             string          `json:"apiKey,omitempty"`
	AccountID          string          `json:"accountId,omitempty"`
	State              string          `json:"state,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountCreation calls the SNAP Account Creation endpoint (Service Code
// 06, path .../{version}/registration-account-creation). hb must already
// carry every field HeaderBuilder needs except Body, which AccountCreation
// sets itself so the exact marshaled bytes are used for both signing and
// the wire body.
func AccountCreation(ctx context.Context, t *Transport, hb HeaderBuilder, req AccountCreationRequest) (AccountCreationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AccountCreationResponse{}, fmt.Errorf("snap: account creation: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AccountCreationResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AccountCreationResponse{}, fmt.Errorf("snap: account creation: %w", err)
	}

	var resp AccountCreationResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AccountCreationResponse{}, fmt.Errorf("snap: account creation: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AccountCreationResponse{}, errors.New("snap: account creation: response has no responseCode")
	}
	return resp, nil
}
