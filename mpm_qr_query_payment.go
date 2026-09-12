package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// QRMPMQueryPaymentRequest is the request body for API Query Payment
// (Service Code 51). It is a superset of TransactionStatusInquiryBankRequest
// (Phase 16) — same "originalX/serviceCode" base fields, since research
// names that same base pattern here, plus MerchantID, SubMerchantID,
// and ExternalStoreID. This is a genuine superset, not an identical
// shape, so per Phase 20's established practice it is a distinct type
// documented in prose rather than reflection-drift-guarded against the
// base. ServiceCode is the only mandatory field.
type QRMPMQueryPaymentRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	ServiceCode                string          `json:"serviceCode"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	Amount                     *Money          `json:"amount,omitempty"`
	MerchantID                 string          `json:"merchantId,omitempty"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// QRMPMQueryPaymentResponse is the response body for API Query
// Payment. It reuses TransactionStatusInquiryBankResponse's full field
// set verbatim (same superset relationship as the request; see that
// type's doc comment for why fields like BeneficiaryAccountNo apply
// here despite not being QR-specific — the research documents this as
// a generic reusable status-response schema) plus PaidTime and
// TerminalID.
type QRMPMQueryPaymentResponse struct {
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
	PaidTime                   string          `json:"paidTime,omitempty"`
	TerminalID                 string          `json:"terminalId,omitempty"`
}

// QRMPMQueryPayment calls the SNAP Query Payment endpoint (Service
// Code 51, path .../{version}/qr/qr-mpm-query, HTTP POST — no method
// override). hb must already carry every field HeaderBuilder needs
// except Body, which QRMPMQueryPayment sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
//
// This is a read-only status query; unlike the package's mutating
// calls it carries no non-idempotency note, matching
// TransactionStatusInquiryBank's precedent.
func QRMPMQueryPayment(ctx context.Context, t *Transport, hb HeaderBuilder, req QRMPMQueryPaymentRequest) (QRMPMQueryPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return QRMPMQueryPaymentResponse{}, fmt.Errorf("snap: qr mpm query payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return QRMPMQueryPaymentResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return QRMPMQueryPaymentResponse{}, fmt.Errorf("snap: qr mpm query payment: %w", err)
	}

	var resp QRMPMQueryPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return QRMPMQueryPaymentResponse{}, fmt.Errorf("snap: qr mpm query payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return QRMPMQueryPaymentResponse{}, errors.New("snap: qr mpm query payment: response has no responseCode")
	}
	return resp, nil
}
