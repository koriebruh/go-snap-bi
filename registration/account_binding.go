package registration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// BindingSuccessParams is the request-side "successParams" object for Account
// Binding.
type BindingSuccessParams struct {
	AccountID        string `json:"accountId,omitempty"`
	TerminalID       string `json:"terminalId,omitempty"`
	TokenRequestorID string `json:"tokenRequestorId,omitempty"`
}

// BindingAccessTokenInfo is the response-side "accessTokenInfo" object for
// Account Binding. ExpiresIn/ReExpiresIn here are ISO 8601 datetime strings
// per the Guides tab ("Datetime of token expiration. Format: ISO 8601") —
// deliberately not the same shape as Token.ExpiresIn (a time.Duration
// parsed from a seconds-count string, per the separate B2B/B2B2C
// access-token endpoints in token.go). Same field name, different
// endpoint, different meaning.
type BindingAccessTokenInfo struct {
	AccessToken  string `json:"accessToken,omitempty"`
	ExpiresIn    string `json:"expiresIn,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ReExpiresIn  string `json:"reExpiresIn,omitempty"`
	TokenStatus  string `json:"tokenStatus,omitempty"`
}

// BindingUserInfo is the response-side "userInfo" object for Account Binding.
type BindingUserInfo struct {
	PublicUserID string `json:"publicUserId,omitempty"`
}

// AccountBindingRequest is the request body for API Account Binding
// (Service Code 07). MerchantID is the only mandatory field.
type AccountBindingRequest struct {
	PartnerReferenceNo string                `json:"partnerReferenceNo,omitempty"`
	Action             string                `json:"action,omitempty"`
	AdditionalData     json.RawMessage       `json:"additionalData,omitempty"`
	UserID             string                `json:"userId,omitempty"`
	Email              string                `json:"email,omitempty"`
	PostalAddress      string                `json:"postalAddress,omitempty"`
	AuthCode           string                `json:"authCode,omitempty"`
	GrantType          string                `json:"grantType,omitempty"`
	IsBindAndPay       string                `json:"isBindAndPay,omitempty"`
	Lang               string                `json:"lang,omitempty"`
	Locale             string                `json:"locale,omitempty"`
	MerchantID         string                `json:"merchantId"`
	SubMerchantID      string                `json:"subMerchantId,omitempty"`
	Msisdn             string                `json:"msisdn,omitempty"`
	OTP                string                `json:"otp,omitempty"`
	PhoneNo            string                `json:"phoneNo,omitempty"`
	PlatformType       string                `json:"platformType,omitempty"`
	RedirectURL        string                `json:"redirectUrl,omitempty"`
	ReferenceID        string                `json:"referenceId,omitempty"`
	RefreshToken       string                `json:"refreshToken,omitempty"`
	SuccessParams      *BindingSuccessParams `json:"successParams,omitempty"`
	AdditionalInfo     json.RawMessage       `json:"additionalInfo,omitempty"`
}

// AccountBindingResponse is the response body for API Account Binding.
type AccountBindingResponse struct {
	ResponseCode       string                  `json:"responseCode"`
	ResponseMessage    string                  `json:"responseMessage"`
	ReferenceNo        string                  `json:"referenceNo,omitempty"`
	PartnerReferenceNo string                  `json:"partnerReferenceNo,omitempty"`
	AccountToken       string                  `json:"accountToken,omitempty"`
	AccessTokenInfo    *BindingAccessTokenInfo `json:"accessTokenInfo,omitempty"`
	LinkID             string                  `json:"linkId,omitempty"`
	NextAction         string                  `json:"nextAction,omitempty"`
	LinkageToken       string                  `json:"linkageToken,omitempty"`
	Params             json.RawMessage         `json:"params,omitempty"`
	PinWebViewURL      string                  `json:"pinWebViewUrl,omitempty"`
	RedirectToDeeplink string                  `json:"redirectToDeeplink,omitempty"`
	RedirectURL        string                  `json:"redirectUrl,omitempty"`
	UserInfo           *BindingUserInfo        `json:"userInfo,omitempty"`
	AdditionalInfo     json.RawMessage         `json:"additionalInfo,omitempty"`
}

// AccountBinding calls the SNAP Account Binding endpoint (Service Code 07,
// path .../{version}/registration-account-binding). hb must already carry
// every field snap.HeaderBuilder needs except Body, which AccountBinding sets
// itself so the exact marshaled bytes are used for both signing and the
// wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it —
// a fresh X-EXTERNAL-ID on retry risks a duplicate binding or a second
// token issuance.
func AccountBinding(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req AccountBindingRequest) (AccountBindingResponse, error) {
	body, err := json.Marshal(req) // #nosec G117 -- refreshToken is a mandatory SNAP wire field here, not a leaked secret
	if err != nil {
		return AccountBindingResponse{}, fmt.Errorf("snap: account binding: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AccountBindingResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AccountBindingResponse{}, fmt.Errorf("snap: account binding: %w", err)
	}

	var resp AccountBindingResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AccountBindingResponse{}, fmt.Errorf("snap: account binding: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AccountBindingResponse{}, errors.New("snap: account binding: response has no responseCode")
	}
	return resp, nil
}
