package registration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// OTPRequest is the request body for API OTP (Service Code 81, path
// v1.0/otp — note this endpoint's path is not templated by {version}
// like every other endpoint in this package; the portal's own path is
// the literal "v1.0/otp"). JourneyID is the only mandatory field per the
// Guides tab.
//
// This endpoint's worked example is the package's first to include an
// Authorization-Customer header — a B2B2C-shaped call. This type takes
// no special parameter for that: a caller sets snap.HeaderBuilder.B2B2C
// (and AuthorizationCustomer/DeviceID, which snap.HeaderBuilder.Build already
// requires when B2B2C is true) the same way as for any other B2B2C call.
//
// SubMerchant is named without an "Id" suffix, unlike subMerchantId in
// every other endpoint in this package — the standard's own
// inconsistency, kept as-is. BankCardToken is Conditional per the Guides
// tab (the only Conditional-marked field found across the researched
// Registrasi group); it carries omitempty the same as every Optional
// field, since the field's presence rather than its JSON tag is what
// makes it conditional. TrxDateTime is typed string despite being
// labeled "Date" (a unique type label in this whole document — every
// other date-like field elsewhere is labeled "String"): JSON has no
// date primitive, and the worked example is a plain quoted ISO-8601
// string, so string is unambiguous regardless of the unique label —
// same reasoning as CardRegistrationUnbindingResponse.UnsubscribeDate.
type OTPRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	JourneyID          string          `json:"journeyId"`
	MerchantID         string          `json:"merchantId,omitempty"`
	SubMerchant        string          `json:"subMerchant,omitempty"`
	ExternalStoreID    string          `json:"externalStoreId,omitempty"`
	TrxDateTime        string          `json:"trxDateTime,omitempty"`
	BankCardToken      string          `json:"bankCardToken,omitempty"`
	OTPTrxCode         string          `json:"otpTrxCode,omitempty"`
	OTPReasonCode      string          `json:"otpReasonCode,omitempty"`
	OTPReasonMessage   string          `json:"otpReasonMessage,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// OTPResponse is the response body for API OTP. ChargeToken is mandatory
// here, unlike the Optional ChargeToken on CardRegistrationResponse
// (Service Code 01) — same field name, different endpoint, different
// mandatory-ness.
type OTPResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	ChargeToken        string          `json:"chargeToken"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// OTP calls the SNAP API OTP endpoint (Service Code 81). hb must already
// carry every field snap.HeaderBuilder needs except Body, which OTP sets
// itself so the exact marshaled bytes are used for both signing and the
// wire body.
//
// This operation triggers a real external side effect (an OTP
// delivery, e.g. SMS) and this package does not retry. Callers that
// retry a failed or timed-out call should reuse the same X-EXTERNAL-ID,
// since the server's own duplicate-detection keys on it — a fresh
// X-EXTERNAL-ID on retry risks a duplicate OTP delivery.
func OTP(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req OTPRequest) (OTPResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return OTPResponse{}, fmt.Errorf("snap: otp: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return OTPResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return OTPResponse{}, fmt.Errorf("snap: otp: %w", err)
	}

	var resp OTPResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return OTPResponse{}, fmt.Errorf("snap: otp: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return OTPResponse{}, errors.New("snap: otp: response has no responseCode")
	}
	return resp, nil
}
