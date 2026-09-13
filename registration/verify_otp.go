package registration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// VerifyOTPRequest is the request body for API Verify OTP (Direct
// Integration) (Service Code 04, path
// .../{version}/otp-verification). Every field is Optional per the
// Guides tab.
type VerifyOTPRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	Action                     string          `json:"action,omitempty"`
	MerchantID                 string          `json:"merchantId,omitempty"`
	OTP                        string          `json:"otp,omitempty"`
	ChargeToken                string          `json:"chargeToken,omitempty"`
	Type                       string          `json:"type,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// VerifyOTPResponse is the response body for API Verify OTP (Direct
// Integration).
//
// TransactionTimestamp is a source artifact, not reconciled: the Guides
// tab's description for this field ("Random String to generate
// validation for webview") doesn't match its name, and the worked
// example holds a random-string-looking value, not an actual timestamp.
//
// QParamsURL's wire tag is "qParamsURL" (capital URL), per the portal's
// own worked example — unlike every other URL field elsewhere in this
// package (redirectUrl, merchantLogoUrl, pinWebViewUrl), which use
// lowercase "Url". Kept exactly as the source shows it, not "corrected"
// to match sibling endpoints' casing.
type VerifyOTPResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	AccountNo                  string          `json:"accountNo,omitempty"`
	BankCardToken              string          `json:"bankCardToken,omitempty"`
	CardPan                    string          `json:"cardPan,omitempty"`
	CustomerID                 string          `json:"customerId,omitempty"`
	Email                      string          `json:"email,omitempty"`
	ExpiredDatetime            string          `json:"expiredDatetime,omitempty"`
	ExpiryDate                 string          `json:"expiryDate,omitempty"`
	IdentificationNo           string          `json:"identificationNo,omitempty"`
	LinkageToken               string          `json:"linkageToken,omitempty"`
	PhoneNo                    string          `json:"phoneNo,omitempty"`
	QParamsURL                 string          `json:"qParamsURL,omitempty"`
	QParams                    json.RawMessage `json:"qParams,omitempty"`
	SendOTPFlag                string          `json:"sendOtpFlag,omitempty"`
	SubscribeDatetime          string          `json:"subscribeDatetime,omitempty"`
	TokenExpiryTime            string          `json:"tokenExpiryTime,omitempty"`
	TransactionTimestamp       string          `json:"transactionTimestamp,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// VerifyOTP calls the SNAP Verify OTP (Direct Integration) endpoint
// (Service Code 04). hb must already carry every field snap.HeaderBuilder
// needs except Body, which VerifyOTP sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation consumes server-side state — it invalidates the OTP
// being verified and, plausibly, increments an attempt counter — so
// this package treats it as non-idempotent and does not retry. Callers
// that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it;
// a fresh X-EXTERNAL-ID on retry risks burning an extra OTP attempt or
// re-processing a call the server already completed.
func VerifyOTP(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req VerifyOTPRequest) (VerifyOTPResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return VerifyOTPResponse{}, fmt.Errorf("snap: verify otp: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return VerifyOTPResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return VerifyOTPResponse{}, fmt.Errorf("snap: verify otp: %w", err)
	}

	var resp VerifyOTPResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return VerifyOTPResponse{}, fmt.Errorf("snap: verify otp: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return VerifyOTPResponse{}, errors.New("snap: verify otp: response has no responseCode")
	}
	return resp, nil
}
