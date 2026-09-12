package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// TransactionStatusInquiryNonBankRequest is the request body for API
// Transaction Status Inquiry (non-bank, Service Code 53). It is a
// superset of TransactionStatusInquiryBankRequest (Phase 16) — same 7
// base fields verbatim — plus OriginalResponseCode, OriginalResponseMessage,
// SessionID, and RequestID.
//
// Research §5.10 line 210 says only "same shape... plus [these four
// fields]" with no Resp: segment naming a response addition, unlike
// every §5.9 row that adds response fields explicitly. Placement on
// the request is inferred from that absence, not stated outright — see
// the Phase 25 design doc.
type TransactionStatusInquiryNonBankRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	ServiceCode                string          `json:"serviceCode"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	Amount                     *Money          `json:"amount,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
	OriginalResponseCode       string          `json:"originalResponseCode,omitempty"`
	OriginalResponseMessage    string          `json:"originalResponseMessage,omitempty"`
	SessionID                  string          `json:"sessionId,omitempty"`
	RequestID                  string          `json:"requestId,omitempty"`
}

// TransactionStatusInquiryNonBankResponse is the response body for API
// Transaction Status Inquiry (non-bank) — field-identical to
// TransactionStatusInquiryBankResponse under its own name, per the
// package's "distinct types per service code even for identical
// shapes" convention. Research names no response-side additions for
// this endpoint (see the request doc comment). Guarded against drift
// by a full-equality reflection test in this package's test file.
type TransactionStatusInquiryNonBankResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	ServiceCode                string          `json:"serviceCode,omitempty"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	Amount                     *Money          `json:"amount,omitempty"`
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

// TransactionStatusInquiryNonBank calls the SNAP Transaction Status
// Inquiry (non-bank) endpoint (Service Code 53, path
// .../{version}/qr/qr-mpm-status, HTTP POST — no method override). The
// shared qr/ path prefix does not mean this belongs to the MPM/QR
// sub-group: research §5.10 is its own sub-group, distinct from §5.9
// MPM/QR, and the path naming is the portal's own inconsistency, not a
// signal to prefix this type QRMPM*. hb must already carry every field
// HeaderBuilder needs except Body, which TransactionStatusInquiryNonBank
// sets itself so the exact marshaled bytes are used for both signing
// and the wire body.
//
// This is a read-only status query and carries no non-idempotency
// note, matching TransactionStatusInquiryBank's precedent.
func TransactionStatusInquiryNonBank(ctx context.Context, t *Transport, hb HeaderBuilder, req TransactionStatusInquiryNonBankRequest) (TransactionStatusInquiryNonBankResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TransactionStatusInquiryNonBankResponse{}, fmt.Errorf("snap: transaction status inquiry non-bank: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return TransactionStatusInquiryNonBankResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return TransactionStatusInquiryNonBankResponse{}, fmt.Errorf("snap: transaction status inquiry non-bank: %w", err)
	}

	var resp TransactionStatusInquiryNonBankResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return TransactionStatusInquiryNonBankResponse{}, fmt.Errorf("snap: transaction status inquiry non-bank: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return TransactionStatusInquiryNonBankResponse{}, errors.New("snap: transaction status inquiry non-bank: response has no responseCode")
	}
	return resp, nil
}
