package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// TransactionStatusInquiryBankRequest is the request body for API
// Transaction Status Inquiry Bank (Service Code 36). ServiceCode is
// the only mandatory field per the Guides tab — it points at the
// original transaction's service code (e.g. "17" for Intrabank).
type TransactionStatusInquiryBankRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	ServiceCode                string          `json:"serviceCode"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	Amount                     *snap.Money     `json:"amount,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// TransactionStatusInquiryBankResponse is the response body for API
// Transaction Status Inquiry Bank. Unlike the Virtual Account
// sub-group, this response has no nested data object — fields sit
// directly on the response, matching the flat-response precedent
// already used by AccountInquiryInternalResponse. The original request
// fields are echoed back (Optional here, regardless of their request
// cardinality), matching AccountInquiryInternalResponse's
// PartnerReferenceNo echo. LatestTransactionStatus is the shared
// transactionStatus 2-digit enum (research §3: 00 Success, 01
// Initiated, 02 Paying, 03 Pending, 04 Refunded, 05 Canceled,
// 06 Failed, 07 Not found).
type TransactionStatusInquiryBankResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	ServiceCode                string          `json:"serviceCode,omitempty"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	Amount                     *snap.Money     `json:"amount,omitempty"`
	BeneficiaryAccountNo       string          `json:"beneficiaryAccountNo"`
	BeneficiaryBankCode        string          `json:"beneficiaryBankCode,omitempty"`
	PreviousResponseCode       string          `json:"previousResponseCode,omitempty"`
	ReferenceNumber            string          `json:"referenceNumber"`
	SourceAccountNo            string          `json:"sourceAccountNo"`
	TransactionID              string          `json:"transactionId,omitempty"`
	LatestTransactionStatus    string          `json:"latestTransactionStatus"`
	TransactionStatusDesc      string          `json:"transactionStatusDesc,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// TransactionStatusInquiryBank calls the SNAP Transaction Status
// Inquiry Bank endpoint (Service Code 36, path .../{version}/transfer/status
// — the path was previously misstated here as
// transaction-status-inquiry-bank with no recorded justification,
// corrected against research §1's own path table, HTTP POST — no
// method override). hb must already carry every field snap.HeaderBuilder needs
// except Body, which TransactionStatusInquiryBank sets itself so the
// exact marshaled bytes are used for both signing and the wire body.
func TransactionStatusInquiryBank(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req TransactionStatusInquiryBankRequest) (TransactionStatusInquiryBankResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TransactionStatusInquiryBankResponse{}, fmt.Errorf("snap: transaction status inquiry bank: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return TransactionStatusInquiryBankResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return TransactionStatusInquiryBankResponse{}, fmt.Errorf("snap: transaction status inquiry bank: %w", err)
	}

	var resp TransactionStatusInquiryBankResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return TransactionStatusInquiryBankResponse{}, fmt.Errorf("snap: transaction status inquiry bank: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return TransactionStatusInquiryBankResponse{}, errors.New("snap: transaction status inquiry bank: response has no responseCode")
	}
	return resp, nil
}
