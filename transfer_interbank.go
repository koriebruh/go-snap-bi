package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// InterbankTransferRequest is the request body for API Interbank Transfer
// (Service Code 18). PartnerReferenceNo, Amount, BeneficiaryAccountNo,
// BeneficiaryAccountName, BeneficiaryBankCode, SourceAccountNo, and
// TransactionDate are mandatory per the Guides tab. OriginatorInfos is
// Conditional.
type InterbankTransferRequest struct {
	PartnerReferenceNo     string                   `json:"partnerReferenceNo"`
	Amount                 TransferAmount           `json:"amount"`
	BeneficiaryAccountNo   string                   `json:"beneficiaryAccountNo"`
	BeneficiaryAccountName string                   `json:"beneficiaryAccountName"`
	BeneficiaryAddress     string                   `json:"beneficiaryAddress,omitempty"`
	BeneficiaryBankCode    string                   `json:"beneficiaryBankCode"`
	BeneficiaryBankName    string                   `json:"beneficiaryBankName,omitempty"`
	BeneficiaryEmail       string                   `json:"beneficiaryEmail,omitempty"`
	Currency               string                   `json:"currency,omitempty"`
	CustomerReference      string                   `json:"customerReference,omitempty"`
	FeeType                string                   `json:"feeType,omitempty"`
	Remark                 string                   `json:"remark,omitempty"`
	SourceAccountNo        string                   `json:"sourceAccountNo"`
	TransactionDate        string                   `json:"transactionDate"`
	OriginatorInfos        []TransferOriginatorInfo `json:"originatorInfos,omitempty"`
	AdditionalInfo         json.RawMessage          `json:"additionalInfo,omitempty"`
}

// InterbankTransferResponse is the response body for API Interbank
// Transfer.
type InterbankTransferResponse struct {
	ResponseCode         string                   `json:"responseCode"`
	ResponseMessage      string                   `json:"responseMessage"`
	ReferenceNo          string                   `json:"referenceNo,omitempty"`
	PartnerReferenceNo   string                   `json:"partnerReferenceNo,omitempty"`
	Amount               *TransferAmount          `json:"amount,omitempty"`
	BeneficiaryAccountNo string                   `json:"beneficiaryAccountNo,omitempty"`
	Currency             string                   `json:"currency,omitempty"`
	CustomerReference    string                   `json:"customerReference,omitempty"`
	SourceAccountNo      string                   `json:"sourceAccountNo,omitempty"`
	TransactionDate      string                   `json:"transactionDate,omitempty"`
	TraceNo              string                   `json:"traceNo,omitempty"`
	OriginatorInfos      []TransferOriginatorInfo `json:"originatorInfos,omitempty"`
	AdditionalInfo       json.RawMessage          `json:"additionalInfo,omitempty"`
}

// InterbankTransfer calls the SNAP Interbank Transfer endpoint (Service
// Code 18, path .../{version}/transfer-interbank). hb must already carry
// every field HeaderBuilder needs except Body, which InterbankTransfer
// sets itself so the exact marshaled bytes are used for both signing and
// the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it —
// a fresh X-EXTERNAL-ID on retry risks a duplicate transfer.
func InterbankTransfer(ctx context.Context, t *Transport, hb HeaderBuilder, req InterbankTransferRequest) (InterbankTransferResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return InterbankTransferResponse{}, fmt.Errorf("snap: interbank transfer: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return InterbankTransferResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return InterbankTransferResponse{}, fmt.Errorf("snap: interbank transfer: %w", err)
	}

	var resp InterbankTransferResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return InterbankTransferResponse{}, fmt.Errorf("snap: interbank transfer: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return InterbankTransferResponse{}, errors.New("snap: interbank transfer: response has no responseCode")
	}
	return resp, nil
}
