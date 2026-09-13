package snap

import "encoding/json"

// DirectDebitBIFASTNotificationRequest is the request body for API
// Notify (Service Code 72, path .../{version}/debit/fast-notify).
// This is a settlement callback the PJP receives, not a call this
// package makes — no calling function is provided, matching the
// package's established convention for inbound notification endpoints
// (QRMPMPaymentNotificationRequest, Phase 23; DirectDebitPaymentNotificationRequest,
// Phase 27; CPMPaymentNotificationRequest, Phase 29): a caller wires
// their own HTTP handler for this path, authenticates the inbound call
// with ServerVerifier.VerifyTransactionRequest, and json.Unmarshals
// the body into this type.
//
// OriginalReferenceNo, TransactionStatus, EMandateReffID,
// SourceAccountNo, and SourceAccountName are Mandatory per research
// §5.4; every other field is Optional.
//
// The status field is named TransactionStatus, not
// LatestTransactionStatus as used everywhere else in both Transfer
// Kredit and the rest of Transfer Debit — research §6 item 5 records
// this as an unexplained naming inconsistency in the portal itself and
// explicitly instructs recording it "as shown, not silently
// normalized." This type follows that instruction literally rather
// than renaming the field to match the rest of the package.
type DirectDebitBIFASTNotificationRequest struct {
	OriginalReferenceNo        string          `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	TransactionStatus          string          `json:"transactionStatus"`
	TransactionStatusDesc      string          `json:"transactionStatusDesc,omitempty"`
	EMandateReffID             string          `json:"eMandateReffId"`
	SourceAccountNo            string          `json:"sourceAccountNo"`
	SourceAccountName          string          `json:"sourceAccountName"`
	Amount                     *Money          `json:"amount,omitempty"`
	TraceNo                    string          `json:"traceNo,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// DirectDebitBIFASTNotificationResponse is the response body a caller
// sends back for API Notify. Research documents this response as
// envelope-only — no fields beyond the standard
// responseCode/responseMessage envelope.
type DirectDebitBIFASTNotificationResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}
