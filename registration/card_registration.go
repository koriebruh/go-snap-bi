package registration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// CardRegistrationRequest is the request body for API Card Registration
// (Service Code 01, path .../{version}/registration-card-bind).
// BankCardNo, CardData, and CustIDMerchant are the three mandatory
// fields per the Guides tab — CardData's own Mandatory marker was
// missed by an earlier version of this type (its doc comment listed
// only BankCardNo/CustIDMerchant, and the Go tag carried an omitempty
// that dropped a caller's CardData entirely if left unset); corrected
// against a direct portal re-verification recorded in
// docs/research/2026-09-13-registrasi-informasi-saldo-riwayat-transaksi-portal-research.md.
//
// CardData and Limit are typed json.RawMessage: the Guides tab labels
// CardData "Encrypted Object" and Limit "decimal", each permitting a
// non-string JSON representation (a bare object, a bare number), even
// though the portal's one worked example renders both as a quoted string.
// A caller assigning a plain Go string must quote it first — e.g.
// json.RawMessage(`"1000000"`), not json.RawMessage(limitStr), and
// json.RawMessage(`"`+cardDataBase64+`"`), not
// json.RawMessage(cardDataBase64) — to match the worked examples' wire
// shape. Getting this wrong is NOT always caught by json.Marshal: if the
// unquoted value happens to itself be a complete, valid JSON value (e.g.
// Limit set to the bare digits "1000000", with no leading zero — JSON
// numbers forbid leading zeros, so "01000000" fails instead), it marshals
// fine as that JSON type and is sent as a bare number/object/string —
// silently diverging from the worked example's quoted-string shape, with
// no error. Only a value that is not itself a valid JSON value (a
// comma-formatted decimal, a base64 blob starting with a letter, a
// digit string with a leading zero) fails json.Marshal and returns an
// "encode request" error before any request is sent — this is a
// narrower, less reliable safety net than it looks, not a guarantee.
// Two other shapes: an empty json.RawMessage is dropped by omitempty
// (the field is absent from the wire body, not sent empty), while
// json.RawMessage("null") is NOT dropped — it is sent as an explicit
// "limit":null / "cardData":null, since omitempty only skips a
// zero-length slice.
type CardRegistrationRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	AccountName        string          `json:"accountName,omitempty"`
	CardData           json.RawMessage `json:"cardData"`
	BankAccountNo      string          `json:"bankAccountNo,omitempty"`
	BankCardNo         string          `json:"bankCardNo"`
	BankCardType       string          `json:"bankCardType,omitempty"`
	DateOfBirth        string          `json:"dateOfBirth,omitempty"`
	Email              string          `json:"email,omitempty"`
	ExpiredDatetime    string          `json:"expiredDatetime,omitempty"`
	ExpiryDate         string          `json:"expiryDate,omitempty"`
	IdentificationNo   string          `json:"identificationNo,omitempty"`
	IdentificationType string          `json:"identificationType,omitempty"`
	CustIDMerchant     string          `json:"custIdMerchant"`
	IsBindAndPay       string          `json:"isBindAndPay,omitempty"`
	MerchantID         string          `json:"merchantId,omitempty"`
	TerminalID         string          `json:"terminalId,omitempty"`
	JourneyID          string          `json:"journeyId,omitempty"`
	SubMerchantID      string          `json:"subMerchantId,omitempty"`
	ExternalStoreID    string          `json:"externalStoreId,omitempty"`
	Limit              json.RawMessage `json:"limit,omitempty"`
	MerchantLogoURL    string          `json:"merchantLogoUrl,omitempty"`
	PhoneNo            string          `json:"phoneNo,omitempty"`
	SendOTPFlag        string          `json:"sendOtpFlag,omitempty"`
	Type               string          `json:"type,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// CardRegistrationResponse is the response body for API Card Registration.
type CardRegistrationResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	BankCardToken      string          `json:"bankCardToken"`
	ChargeToken        string          `json:"chargeToken,omitempty"`
	RandomString       string          `json:"randomString,omitempty"`
	TokenExpiryTime    string          `json:"tokenExpiryTime,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// CardRegistration calls the SNAP Card Registration endpoint (Service Code
// 01). It mints a BankCardToken, so this operation is not idempotent and
// this package does not retry. Callers that retry a failed or timed-out
// call should reuse the same X-EXTERNAL-ID, since the server's own
// duplicate-detection keys on it — a fresh X-EXTERNAL-ID on retry risks a
// duplicate card bind.
//
// hb must already carry every field snap.HeaderBuilder needs except Body,
// which CardRegistration sets itself so the exact marshaled bytes are used
// for both signing and the wire body.
func CardRegistration(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req CardRegistrationRequest) (CardRegistrationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CardRegistrationResponse{}, fmt.Errorf("snap: card registration: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CardRegistrationResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CardRegistrationResponse{}, fmt.Errorf("snap: card registration: %w", err)
	}

	var resp CardRegistrationResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CardRegistrationResponse{}, fmt.Errorf("snap: card registration: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CardRegistrationResponse{}, errors.New("snap: card registration: response has no responseCode")
	}
	return resp, nil
}
