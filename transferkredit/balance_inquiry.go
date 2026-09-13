package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// AccountInfo is one entry in a BalanceInquiryResponse's accountInfos array.
type AccountInfo struct {
	BalanceType              string          `json:"balanceType"`
	Amount                   snap.Money      `json:"amount"`
	FloatAmount              snap.Money      `json:"floatAmount"`
	HoldAmount               snap.Money      `json:"holdAmount"`
	AvailableBalance         snap.Money      `json:"availableBalance"`
	LedgerBalance            snap.Money      `json:"ledgerBalance"`
	CurrentMultilateralLimit snap.Money      `json:"currentMultilateralLimit"`
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
// snap.HeaderBuilder needs except Body, which BalanceInquiry sets itself so the
// exact marshaled bytes are used for both signing and the wire body.
func BalanceInquiry(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req BalanceInquiryRequest) (BalanceInquiryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return BalanceInquiryResponse{}, fmt.Errorf("snap: balance inquiry: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return BalanceInquiryResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		// snap.CheckResponseStatus is authoritative on the transport status: a
		// non-2xx HTTP status is never treated as success, even if the body
		// claims a 2xx-class responseCode (e.g. a stale cached body from a
		// misbehaving intermediary).
		return BalanceInquiryResponse{}, fmt.Errorf("snap: balance inquiry: %w", err)
	}

	var resp BalanceInquiryResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return BalanceInquiryResponse{}, fmt.Errorf("snap: balance inquiry: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		// responseCode is mandatory in the standard's response shape. By
		// this point a non-2xx status with no responseCode was already
		// caught above (or, for a non-JSON body, in snap.Transport.Do itself) —
		// the only way to reach here is a 2xx HTTP status whose JSON body
		// still omits responseCode, a malformed "successful" response that
		// must not be returned as a zero-value success.
		return BalanceInquiryResponse{}, errors.New("snap: balance inquiry: response has no responseCode")
	}
	return resp, nil
}
