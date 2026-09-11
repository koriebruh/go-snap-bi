package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// IntrabankTransferRequest is the request body for API Intrabank Transfer
// (Service Code 17). PartnerReferenceNo, Amount, BeneficiaryAccountNo,
// SourceAccountNo, and TransactionDate are mandatory per the Guides tab.
// OriginatorInfos is Conditional.
type IntrabankTransferRequest struct {
	PartnerReferenceNo   string                   `json:"partnerReferenceNo"`
	Amount               TransferAmount           `json:"amount"`
	BeneficiaryAccountNo string                   `json:"beneficiaryAccountNo"`
	BeneficiaryEmail     string                   `json:"beneficiaryEmail,omitempty"`
	Currency             string                   `json:"currency,omitempty"`
	CustomerReference    string                   `json:"customerReference,omitempty"`
	FeeType              string                   `json:"feeType,omitempty"`
	Remark               string                   `json:"remark,omitempty"`
	SourceAccountNo      string                   `json:"sourceAccountNo"`
	TransactionDate      string                   `json:"transactionDate"`
	OriginatorInfos      []TransferOriginatorInfo `json:"originatorInfos,omitempty"`
	AdditionalInfo       json.RawMessage          `json:"additionalInfo,omitempty"`
}

// IntrabankTransferResponse is the response body for API Intrabank
// Transfer.
type IntrabankTransferResponse struct {
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
	OriginatorInfos      []TransferOriginatorInfo `json:"originatorInfos,omitempty"`
	AdditionalInfo       json.RawMessage          `json:"additionalInfo,omitempty"`
}

// IntrabankTransfer calls the SNAP Intrabank Transfer endpoint (Service
// Code 17, path .../{version}/transfer-intrabank). hb must already carry
// every field HeaderBuilder needs except Body, which IntrabankTransfer
// sets itself so the exact marshaled bytes are used for both signing and
// the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it —
// a fresh X-EXTERNAL-ID on retry risks a duplicate transfer.
func IntrabankTransfer(ctx context.Context, t *Transport, hb HeaderBuilder, req IntrabankTransferRequest) (IntrabankTransferResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return IntrabankTransferResponse{}, fmt.Errorf("snap: intrabank transfer: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return IntrabankTransferResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return IntrabankTransferResponse{}, fmt.Errorf("snap: intrabank transfer: %w", err)
	}

	var resp IntrabankTransferResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return IntrabankTransferResponse{}, fmt.Errorf("snap: intrabank transfer: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return IntrabankTransferResponse{}, errors.New("snap: intrabank transfer: response has no responseCode")
	}
	return resp, nil
}
