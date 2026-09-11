package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// TransferToBankAccountInquiryRequest is the request body for API
// Transfer To Bank - Account Inquiry (Service Code 42). CustomerNumber
// and Amount are mandatory per the Guides tab. CustomerNumber's JSON
// tag is "CustomerNumber" (capital C) — this endpoint's own field
// table documents it that way, unlike every other endpoint in the
// package, which uses lowercase-first "customerNumber." No worked
// example exists to resolve whether this is a typo, so this package
// models the literal, only-available documented casing rather than
// guessing (research §6 item 7).
type TransferToBankAccountInquiryRequest struct {
	PartnerReferenceNo       string `json:"partnerReferenceNo,omitempty"`
	CustomerNumber           string `json:"CustomerNumber"`
	Amount                   Money  `json:"amount"`
	BeneficiaryAccountNumber string `json:"beneficiaryAccountNumber,omitempty"`
}

// TransferToBankAccountInquiryResponse is the response body for API
// Transfer To Bank - Account Inquiry.
type TransferToBankAccountInquiryResponse struct {
	ResponseCode             string `json:"responseCode"`
	ResponseMessage          string `json:"responseMessage"`
	AccountType              string `json:"accountType,omitempty"`
	BeneficiaryAccountNumber string `json:"beneficiaryAccountNumber"`
	BeneficiaryAccountName   string `json:"beneficiaryAccountName"`
	BeneficiaryBankCode      string `json:"beneficiaryBankCode,omitempty"`
	BeneficiaryBankShortName string `json:"beneficiaryBankShortName,omitempty"`
	BeneficiaryBankName      string `json:"beneficiaryBankName,omitempty"`
	Amount                   Money  `json:"amount"`
	SessionID                string `json:"sessionId,omitempty"`
}

// TransferToBankAccountInquiry calls the SNAP Transfer To Bank -
// Account Inquiry endpoint (Service Code 42, path
// .../{version}/emoney/bank-account-inquiry, HTTP POST — no method
// override). hb must already carry every field HeaderBuilder needs
// except Body, which TransferToBankAccountInquiry sets itself so the
// exact marshaled bytes are used for both signing and the wire body.
func TransferToBankAccountInquiry(ctx context.Context, t *Transport, hb HeaderBuilder, req TransferToBankAccountInquiryRequest) (TransferToBankAccountInquiryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TransferToBankAccountInquiryResponse{}, fmt.Errorf("snap: transfer to bank account inquiry: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return TransferToBankAccountInquiryResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return TransferToBankAccountInquiryResponse{}, fmt.Errorf("snap: transfer to bank account inquiry: %w", err)
	}

	var resp TransferToBankAccountInquiryResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return TransferToBankAccountInquiryResponse{}, fmt.Errorf("snap: transfer to bank account inquiry: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return TransferToBankAccountInquiryResponse{}, errors.New("snap: transfer to bank account inquiry: response has no responseCode")
	}
	return resp, nil
}
