package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Money is the shared {value, currency} amount shape used across every
// per-service response that carries a monetary value. value is a decimal
// string (e.g. "200000.00"); currency is ISO 4217 (e.g. "IDR").
type Money struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// AccountInfo is one entry in a BalanceInquiryResponse's accountInfos array.
type AccountInfo struct {
	BalanceType              string          `json:"balanceType"`
	Amount                   Money           `json:"amount"`
	FloatAmount              Money           `json:"floatAmount"`
	HoldAmount               Money           `json:"holdAmount"`
	AvailableBalance         Money           `json:"availableBalance"`
	LedgerBalance            Money           `json:"ledgerBalance"`
	CurrentMultilateralLimit Money           `json:"currentMultilateralLimit"`
	RegistrationStatusCode   string          `json:"registrationStatusCode"`
	Status                   string          `json:"status"`
	AdditionalInfo           json.RawMessage `json:"additionalInfo,omitempty"`
}

// BalanceInquiryRequest is the request body for API Balance Inquiry
// (Service Code 11). Per the standard, exactly one of BankCardToken or
// AccountNo is expected to be set (unless a B2B2C customer token supplies
// the account context instead) — this type does not enforce that itself,
// the server validates and rejects an inquiry that omits both.
type BalanceInquiryRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	BankCardToken      string          `json:"bankCardToken,omitempty"`
	AccountNo          string          `json:"accountNo,omitempty"`
	BalanceTypes       []string        `json:"balanceTypes,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// BalanceInquiryResponse is the response body for API Balance Inquiry.
type BalanceInquiryResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	AccountNo          string          `json:"accountNo,omitempty"`
	Name               string          `json:"name,omitempty"`
	AccountInfos       []AccountInfo   `json:"accountInfos,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// BalanceInquiry calls the SNAP Balance Inquiry endpoint (Service Code 11,
// path .../{version}/balance-inquiry). hb must already carry every field
// HeaderBuilder needs except Body, which BalanceInquiry sets itself so the
// exact marshaled bytes are used for both signing and the wire body.
func BalanceInquiry(ctx context.Context, t *Transport, hb HeaderBuilder, req BalanceInquiryRequest) (BalanceInquiryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return BalanceInquiryResponse{}, fmt.Errorf("snap: balance inquiry: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return BalanceInquiryResponse{}, err
	}
	if env.ResponseCode == "" && (env.StatusCode < 200 || env.StatusCode >= 300) {
		// The body didn't carry a responseCode (e.g. a proxy/WAF error page)
		// but the transport-level status still says this failed — fall back
		// to a status-derived sentinel so errors.Is still works, mirroring
		// TokenManager's identical fallback in token.go.
		return BalanceInquiryResponse{}, fmt.Errorf("snap: balance inquiry: %w: http status %d", sentinelForHTTPStatus(env.StatusCode), env.StatusCode)
	}
	if err := envelopeError(env.ResponseCode); err != nil {
		return BalanceInquiryResponse{}, fmt.Errorf("snap: balance inquiry: %w", err)
	}

	var resp BalanceInquiryResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return BalanceInquiryResponse{}, fmt.Errorf("snap: balance inquiry: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		// responseCode is mandatory in the standard's response shape. By
		// this point a non-2xx status with no responseCode was already
		// caught above (or, for a non-JSON body, in Transport.Do itself) —
		// the only way to reach here is a 2xx HTTP status whose JSON body
		// still omits responseCode, a malformed "successful" response that
		// must not be returned as a zero-value success.
		return BalanceInquiryResponse{}, errors.New("snap: balance inquiry: response has no responseCode")
	}
	return resp, nil
}
