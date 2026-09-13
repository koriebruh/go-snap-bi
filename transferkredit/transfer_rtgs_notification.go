package transferkredit

import (
	"encoding/json"

	snap "github.com/koriebruh/go-snap-bi"
)

// RTGSNotificationRequest is the request body for API RTGS -
// Notification (Service Code 76, path
// .../{version}/transfer-rtgs/notify). This is a settlement callback
// the PJP receives, not a call this package makes — no calling function
// is provided. A caller wires their own HTTP handler for this path,
// authenticates the inbound call with
// ServerVerifier.VerifyTransactionRequest, and json.Unmarshals the body
// into this type. LatestTransactionStatus, BeneficiaryAccountName,
// BeneficiaryAccountNo, BeneficiaryBankCode, SourceAccountNo, and
// TransactionDate are mandatory per the Guides tab.
type RTGSNotificationRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	LatestTransactionStatus    string          `json:"latestTransactionStatus"`
	Amount                     *snap.Money     `json:"amount,omitempty"`
	BeneficiaryAccountName     string          `json:"beneficiaryAccountName"`
	BeneficiaryAccountNo       string          `json:"beneficiaryAccountNo"`
	BeneficiaryBankCode        string          `json:"beneficiaryBankCode"`
	SourceAccountNo            string          `json:"sourceAccountNo"`
	TransactionDate            string          `json:"transactionDate"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// RTGSNotificationResponse is the response body a caller sends back for
// API RTGS - Notification. Envelope-only per the Guides tab — no other
// fields.
type RTGSNotificationResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}
