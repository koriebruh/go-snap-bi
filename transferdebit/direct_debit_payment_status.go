package transferdebit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// DirectDebitRefundHistoryItem is one entry in Direct Debit Payment
// Status's response "refundHistory[]" array. RefundStatus is a
// documented String(2) 00/03/06 subset of the 8-value transaction
// enum, modeled as plain string, matching the package's existing
// convention of not creating enum types for status fields elsewhere.
type DirectDebitRefundHistoryItem struct {
	RefundNo        string      `json:"refundNo,omitempty"`
	PartnerRefundNo string      `json:"partnerRefundNo"`
	RefundAmount    *snap.Money `json:"refundAmount,omitempty"`
	RefundStatus    string      `json:"refundStatus"`
	RefundDate      string      `json:"refundDate,omitempty"`
	Reason          string      `json:"reason,omitempty"`
}

// DirectDebitPaymentStatusRequest is the request body for API Direct
// Debit Payment Status (Service Code 55). Uses the "originalX +
// serviceCode" core shape shared with TransactionStatusInquiryBankRequest,
// but with this sub-group's own distinct additions (merchantId,
// subMerchantId, externalStoreId) — kept as an independent type rather
// than reused/extended from Transfer Kredit's base type, since the two
// differ in field composition. ServiceCode is the only mandatory field.
type DirectDebitPaymentStatusRequest struct {
	OriginalPartnerReferenceNo string      `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string      `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string      `json:"originalExternalId,omitempty"`
	ServiceCode                string      `json:"serviceCode"`
	TransactionDate            string      `json:"transactionDate,omitempty"`
	Amount                     *snap.Money `json:"amount,omitempty"`
	MerchantID                 string      `json:"merchantId,omitempty"`
	SubMerchantID              string      `json:"subMerchantId,omitempty"`
	ExternalStoreID            string      `json:"externalStoreId,omitempty"`
}

// DirectDebitPaymentStatusResponse is the response body for API Direct
// Debit Payment Status. LatestTransactionStatus is the only mandatory
// field beyond the envelope, matching every other occurrence of this
// field package-wide.
type DirectDebitPaymentStatusResponse struct {
	ResponseCode               string                         `json:"responseCode"`
	ResponseMessage            string                         `json:"responseMessage"`
	OriginalPartnerReferenceNo string                         `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string                         `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string                         `json:"originalExternalId,omitempty"`
	ServiceCode                string                         `json:"serviceCode,omitempty"`
	TransactionDate            string                         `json:"transactionDate,omitempty"`
	Amount                     *snap.Money                    `json:"amount,omitempty"`
	ApprovalCode               string                         `json:"approvalCode,omitempty"`
	LatestTransactionStatus    string                         `json:"latestTransactionStatus"`
	TransactionStatusDesc      string                         `json:"transactionStatusDesc,omitempty"`
	OriginalResponseCode       string                         `json:"originalResponseCode,omitempty"`
	OriginalResponseMessage    string                         `json:"originalResponseMessage,omitempty"`
	SessionID                  string                         `json:"sessionId,omitempty"`
	RequestID                  string                         `json:"requestId,omitempty"`
	RefundHistory              []DirectDebitRefundHistoryItem `json:"refundHistory,omitempty"`
	TransAmount                *snap.Money                    `json:"transAmount,omitempty"`
	FeeAmount                  *snap.Money                    `json:"feeAmount,omitempty"`
	PaidTime                   string                         `json:"paidTime,omitempty"`
}

// DirectDebitPaymentStatus calls the SNAP Direct Debit Payment Status
// endpoint (Service Code 55, path .../{version}/debit/status, HTTP
// POST — no method override). hb must already carry every field
// snap.HeaderBuilder needs except Body, which DirectDebitPaymentStatus sets
// itself so the exact marshaled bytes are used for both signing and
// the wire body.
//
// This is a read-only status query and carries no non-idempotency
// note, matching TransactionStatusInquiryBank/TransactionStatusInquiryNonBank
// precedent.
func DirectDebitPaymentStatus(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req DirectDebitPaymentStatusRequest) (DirectDebitPaymentStatusResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return DirectDebitPaymentStatusResponse{}, fmt.Errorf("snap: direct debit payment status: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return DirectDebitPaymentStatusResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return DirectDebitPaymentStatusResponse{}, fmt.Errorf("snap: direct debit payment status: %w", err)
	}

	var resp DirectDebitPaymentStatusResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return DirectDebitPaymentStatusResponse{}, fmt.Errorf("snap: direct debit payment status: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return DirectDebitPaymentStatusResponse{}, errors.New("snap: direct debit payment status: response has no responseCode")
	}
	return resp, nil
}
