package transferdebit

import (
	"encoding/json"

	snap "github.com/koriebruh/go-snap-bi"
)

// DirectDebitPaymentNotificationRequest is the request body for API
// Direct Debit Payment Notification (Service Code 56, path
// .../{version}/debit/notify). This is a settlement callback the PJP
// receives, not a call this package makes — no calling function is
// provided, matching the package's established convention for inbound
// notification endpoints (QRMPMPaymentNotificationRequest, Phase 23;
// NotifyBulkCashInRequest, Phase 18): a caller wires their own HTTP
// handler for this path, authenticates the inbound call with
// ServerVerifier.VerifyTransactionRequest, and json.Unmarshals the body
// into this type.
//
// OriginalReferenceNo and LatestTransactionStatus are Mandatory per
// research §5.1 line 143; every other field is Optional. Nothing in
// that line states the PJP sends this call outward, so the inbound
// reading is the default, not a guess.
type DirectDebitPaymentNotificationRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	MerchantID                 string          `json:"merchantId,omitempty"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	Amount                     *snap.Money     `json:"amount,omitempty"`
	LatestTransactionStatus    string          `json:"latestTransactionStatus"`
	TransactionStatusDesc      string          `json:"transactionStatusDesc,omitempty"`
	CreatedTime                string          `json:"createdTime,omitempty"`
	FinishedTime               string          `json:"finishedTime,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitPaymentNotificationResponse is the response body a
// caller sends back for API Direct Debit Payment Notification.
// ApprovalCode is the only field beyond the envelope, and is Optional.
// Research's worked example shows a non-standard field order
// (responseCode, approvalCode, responseMessage) — recorded here since
// JSON field order carries no semantic meaning; this struct keeps the
// package's conventional envelope-first order.
type DirectDebitPaymentNotificationResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	ApprovalCode    string `json:"approvalCode,omitempty"`
}
