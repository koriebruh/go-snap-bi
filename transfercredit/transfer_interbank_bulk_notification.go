package transfercredit

import "encoding/json"

// InterbankBulkTransferNotificationRequest is the request body for API
// Interbank Bulk Transfer - Notification (Service Code 21, path
// .../{version}/transfer-interbank-bulk/notify). This is a settlement
// callback the PJP receives, not a call this package makes — no
// calling function is provided. A caller wires their own HTTP handler
// for this path, authenticates the inbound call with
// ServerVerifier.VerifyTransactionRequest, and json.Unmarshals the body
// into this type. BulkID, PartnerBulkID, and BulkObject are mandatory
// per the Guides tab.
type InterbankBulkTransferNotificationRequest struct {
	BulkID        string                                  `json:"bulkId"`
	PartnerBulkID string                                  `json:"partnerBulkId"`
	BulkObject    []InterbankBulkTransferNotificationItem `json:"bulkObject"`
}

// InterbankBulkTransferNotificationResponse is the response body a
// caller sends back for API Interbank Bulk Transfer - Notification.
type InterbankBulkTransferNotificationResponse struct {
	ResponseCode    string          `json:"responseCode"`
	ResponseMessage string          `json:"responseMessage"`
	BulkID          string          `json:"bulkId,omitempty"`
	PartnerBulkID   string          `json:"partnerBulkId,omitempty"`
	AdditionalInfo  json.RawMessage `json:"additionalInfo,omitempty"`
}
