package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// SKNBITransferRequest is the request body for API Transfer SKNBI
// (Service Code 23) — field-for-field identical shape to
// RTGSTransferRequest, with its own service code and path.
// PartnerReferenceNo, Amount, BeneficiaryAccountNo,
// BeneficiaryAccountName, BeneficiaryBankCode, SourceAccountNo,
// TransactionDate, BeneficiaryCustomerResidence, and
// BeneficiaryCustomerType are mandatory per the Guides tab.
// OriginatorInfos is Conditional.
type SKNBITransferRequest struct {
	PartnerReferenceNo           string                   `json:"partnerReferenceNo"`
	Amount                       snap.Money               `json:"amount"`
	BeneficiaryAccountNo         string                   `json:"beneficiaryAccountNo"`
	BeneficiaryAccountName       string                   `json:"beneficiaryAccountName"`
	BeneficiaryAddress           string                   `json:"beneficiaryAddress,omitempty"`
	BeneficiaryBankCode          string                   `json:"beneficiaryBankCode"`
	BeneficiaryBankName          string                   `json:"beneficiaryBankName,omitempty"`
	BeneficiaryEmail             string                   `json:"beneficiaryEmail,omitempty"`
	Currency                     string                   `json:"currency,omitempty"`
	CustomerReference            string                   `json:"customerReference,omitempty"`
	FeeType                      string                   `json:"feeType,omitempty"`
	Remark                       string                   `json:"remark,omitempty"`
	SourceAccountNo              string                   `json:"sourceAccountNo"`
	TransactionDate              string                   `json:"transactionDate"`
	BeneficiaryCustomerResidence string                   `json:"beneficiaryCustomerResidence"`
	BeneficiaryCustomerType      string                   `json:"beneficiaryCustomerType"`
	Kodepos                      string                   `json:"kodepos,omitempty"`
	ReceiverPhone                string                   `json:"receiverPhone,omitempty"`
	SenderCustomerResidence      string                   `json:"senderCustomerResidence,omitempty"`
	SenderCustomerType           string                   `json:"senderCustomerType,omitempty"`
	SenderPhone                  string                   `json:"senderPhone,omitempty"`
	OriginatorInfos              []TransferOriginatorInfo `json:"originatorInfos,omitempty"`
	AdditionalInfo               json.RawMessage          `json:"additionalInfo,omitempty"`
}

// SKNBITransferResponse is the response body for API Transfer SKNBI —
// field-for-field identical shape to RTGSTransferResponse.
type SKNBITransferResponse struct {
	ResponseCode           string                   `json:"responseCode"`
	ResponseMessage        string                   `json:"responseMessage"`
	ReferenceNo            string                   `json:"referenceNo,omitempty"`
	PartnerReferenceNo     string                   `json:"partnerReferenceNo,omitempty"`
	Amount                 *snap.Money              `json:"amount,omitempty"`
	BeneficiaryAccountNo   string                   `json:"beneficiaryAccountNo,omitempty"`
	Currency               string                   `json:"currency,omitempty"`
	CustomerReference      string                   `json:"customerReference,omitempty"`
	SourceAccountNo        string                   `json:"sourceAccountNo,omitempty"`
	TransactionDate        string                   `json:"transactionDate,omitempty"`
	TraceNo                string                   `json:"traceNo,omitempty"`
	TransactionStatus      string                   `json:"transactionStatus,omitempty"`
	TransactionStatusDesc  string                   `json:"transactionStatusDesc,omitempty"`
	BeneficiaryAccountType string                   `json:"beneficiaryAccountType,omitempty"`
	OriginatorInfos        []TransferOriginatorInfo `json:"originatorInfos,omitempty"`
	AdditionalInfo         json.RawMessage          `json:"additionalInfo,omitempty"`
}

// SKNBITransfer calls the SNAP Transfer SKNBI endpoint (Service Code
// 23, path .../{version}/transfer-skn). hb must already carry every
// field snap.HeaderBuilder needs except Body, which SKNBITransfer sets
// itself so the exact marshaled bytes are used for both signing and the
// wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it —
// a fresh X-EXTERNAL-ID on retry risks a duplicate transfer.
func SKNBITransfer(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req SKNBITransferRequest) (SKNBITransferResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return SKNBITransferResponse{}, fmt.Errorf("snap: sknbi transfer: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return SKNBITransferResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return SKNBITransferResponse{}, fmt.Errorf("snap: sknbi transfer: %w", err)
	}

	var resp SKNBITransferResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return SKNBITransferResponse{}, fmt.Errorf("snap: sknbi transfer: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return SKNBITransferResponse{}, errors.New("snap: sknbi transfer: response has no responseCode")
	}
	return resp, nil
}
