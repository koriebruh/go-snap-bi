package snap

// QRMPMPaymentNotificationRequest is the request body for API Payment
// Notification (Service Code 52, path .../{version}/qr/qr-mpm-notify).
// This is a settlement callback the PJP receives, not a call this
// package makes — no calling function is provided, matching the
// package's established convention for inbound notification endpoints
// (NotifyBulkCashInRequest, Phase 18): a caller wires their own HTTP
// handler for this path, authenticates the inbound call with
// ServerVerifier.VerifyTransactionRequest, and json.Unmarshals the body
// into this type.
//
// OriginalReferenceNo and LatestTransactionStatus are mandatory per the
// Guides tab; every other field is Optional. Nothing in research §5.9
// line 202 states the PJP sends this call outward (contrast VA Notify
// Payment Intrabank's explicit "Callback the PJP sends outward"
// language, §5.3 line 154 — the one confirmed exception to this
// default), so the inbound reading is the default, not a guess.
type QRMPMPaymentNotificationRequest struct {
	OriginalReferenceNo        string `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string `json:"originalPartnerReferenceNo,omitempty"`
	LatestTransactionStatus    string `json:"latestTransactionStatus"`
	CustomerNumber             string `json:"customerNumber,omitempty"`
	AccountType                string `json:"accountType,omitempty"`
	DestinationNumber          string `json:"destinationNumber,omitempty"`
	DestinationAccountName     string `json:"destinationAccountName,omitempty"`
	Amount                     *Money `json:"amount,omitempty"`
	SessionID                  string `json:"sessionId,omitempty"`
	BankCode                   string `json:"bankCode,omitempty"`
	ExternalStoreID            string `json:"externalStoreId,omitempty"`
}

// QRMPMPaymentNotificationResponse is the response body a caller sends
// back for API Payment Notification. Research documents this response
// as "envelope-only" (§5.9 line 202) — no fields beyond the standard
// responseCode/responseMessage envelope.
type QRMPMPaymentNotificationResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}
