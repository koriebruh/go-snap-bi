package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// DirectDebitPaymentURLParam is one entry in Direct Debit Payment's
// request "urlParams[]" array. Type is documented as PAY_RETURN or
// PAY_NOTIFY but modeled as a plain string since research gives no
// exhaustive closed list. IsDeeplink is a documented String(1) Y/N
// flag, not the ambiguous-numeric-type case — plain string, not
// json.RawMessage.
type DirectDebitPaymentURLParam struct {
	URL        string `json:"url"`
	Type       string `json:"type"`
	IsDeeplink string `json:"isDeeplink"`
}

// DirectDebitPayOptionDetail is one entry in Direct Debit Payment's
// request "payOptionDetails[]" array. TransAmount and FeeAmount are
// documented as Optional containers with Mandatory members — modeled
// as *Money per the package's established handling (Optional
// container wins over internal-member markers).
type DirectDebitPayOptionDetail struct {
	PayMethod      string          `json:"payMethod"`
	PayOption      string          `json:"payOption"`
	TransAmount    *Money          `json:"transAmount,omitempty"`
	FeeAmount      *Money          `json:"feeAmount,omitempty"`
	CardToken      string          `json:"cardToken,omitempty"`
	MerchantToken  string          `json:"merchantToken,omitempty"`
	AdditionalInfo json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitPaymentRequest is the request body for API Direct Debit
// Payment (Service Code 54). PartnerReferenceNo is the only mandatory
// field per research §5.1.
type DirectDebitPaymentRequest struct {
	PartnerReferenceNo string                       `json:"partnerReferenceNo"`
	BankCardToken      string                       `json:"bankCardToken,omitempty"`
	ChargeToken        string                       `json:"chargeToken,omitempty"`
	OTP                string                       `json:"otp,omitempty"`
	OTPTrxCode         string                       `json:"otpTrxCode,omitempty"`
	MerchantID         string                       `json:"merchantId,omitempty"`
	TerminalID         string                       `json:"terminalId,omitempty"`
	JourneyID          string                       `json:"journeyId,omitempty"`
	SubMerchantID      string                       `json:"subMerchantId,omitempty"`
	Amount             *Money                       `json:"amount,omitempty"`
	URLParams          []DirectDebitPaymentURLParam `json:"urlParams,omitempty"`
	ExternalStoreID    string                       `json:"externalStoreId,omitempty"`
	ValidUpTo          string                       `json:"validUpTo,omitempty"`
	PointOfInitiation  string                       `json:"pointOfInitiation,omitempty"`
	FeeType            string                       `json:"feeType,omitempty"`
	DisabledPayMethods string                       `json:"disabledPayMethods,omitempty"`
	PayOptionDetails   []DirectDebitPayOptionDetail `json:"payOptionDetails,omitempty"`
	AdditionalInfo     json.RawMessage              `json:"additionalInfo,omitempty"`
}

// DirectDebitPaymentResponse is the response body for API Direct Debit
// Payment. ReferenceNo is Conditional (success only). No response
// field is Mandatory beyond the envelope.
type DirectDebitPaymentResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	ApprovalCode       string          `json:"approvalCode,omitempty"`
	AppRedirectURL     string          `json:"appRedirectUrl,omitempty"`
	WebRedirectURL     string          `json:"webRedirectUrl,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitPayment calls the SNAP Direct Debit Payment endpoint
// (Service Code 54, path .../{version}/debit/payment-host-to-host,
// HTTP POST — no method override). hb must already carry every field
// HeaderBuilder needs except Body, which DirectDebitPayment sets
// itself so the exact marshaled bytes are used for both signing and
// the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func DirectDebitPayment(ctx context.Context, t *Transport, hb HeaderBuilder, req DirectDebitPaymentRequest) (DirectDebitPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return DirectDebitPaymentResponse{}, fmt.Errorf("snap: direct debit payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return DirectDebitPaymentResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return DirectDebitPaymentResponse{}, fmt.Errorf("snap: direct debit payment: %w", err)
	}

	var resp DirectDebitPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return DirectDebitPaymentResponse{}, fmt.Errorf("snap: direct debit payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return DirectDebitPaymentResponse{}, errors.New("snap: direct debit payment: response has no responseCode")
	}
	return resp, nil
}
