package snap

import "encoding/json"

// CPMPaymentNotificationRequest is the request body for API Payment
// Notification (Service Code 79, path .../{version}/qr/qr-cpm-notify).
// This is a settlement callback the PJP receives, not a call this
// package makes — no calling function is provided, matching the
// package's established convention for inbound notification endpoints
// (QRMPMPaymentNotificationRequest, Phase 23; DirectDebitPaymentNotificationRequest,
// Phase 27): a caller wires their own HTTP handler for this path,
// authenticates the inbound call with ServerVerifier.VerifyTransactionRequest,
// and json.Unmarshals the body into this type.
//
// MerchantID and LatestTransactionStatus are Mandatory per research
// §5.2; every other field is Optional. Research explicitly confirms
// this endpoint matches QRMPMPaymentNotification's shape and
// direction (envelope-only response, inbound), unlike Cancel Payment
// (62)'s contradicted "structurally identical" claim in the same
// section — this citation held up under verification.
type CPMPaymentNotificationRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	MerchantID                 string          `json:"merchantId"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	Amount                     *Money          `json:"amount,omitempty"`
	LatestTransactionStatus    string          `json:"latestTransactionStatus"`
	TransactionStatusDesc      string          `json:"transactionStatusDesc,omitempty"`
	CustomerNumber             string          `json:"customerNumber,omitempty"`
	AccountType                string          `json:"accountType,omitempty"`
	DestinationNumber          string          `json:"destinationNumber,omitempty"`
	DestinationAccountName     string          `json:"destinationAccountName,omitempty"`
	SessionID                  string          `json:"sessionId,omitempty"`
	BankCode                   string          `json:"bankCode,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMPaymentNotificationResponse is the response body a caller sends
// back for API Payment Notification. Research documents this response
// as envelope-only — no fields beyond the standard
// responseCode/responseMessage envelope.
type CPMPaymentNotificationResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}
