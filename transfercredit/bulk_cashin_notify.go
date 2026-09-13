package transfercredit

import (
	"encoding/json"

	snap "github.com/koriebruh/go-snap-bi"
)

// BulkCashInNotificationItem is one entry in Notify Bulk Cash In's
// request "bulkObject[]" array — a settlement-result shape, distinct
// from BulkCashInItem's transfer-instruction shape (per the "distinct
// types per service code" convention: this type has 8 fields vs.
// Phase 12's InterbankBulkTransferNotificationItem's 3, so it is not a
// reuse of that type either). CustomerNumber, ReferenceNo,
// PartnerReferenceNo, ResponseCode, and ResponseMessage are all
// mandatory per the Guides tab (research §5.6 line 174's slash
// notation lists each pair as jointly Mandatory, not either/or — see
// the Phase 18 design doc).
type BulkCashInNotificationItem struct {
	CustomerNumber     string          `json:"customerNumber"`
	CustomerName       string          `json:"customerName,omitempty"`
	Amount             *snap.Money     `json:"amount,omitempty"`
	ReferenceNo        string          `json:"referenceNo"`
	PartnerReferenceNo string          `json:"partnerReferenceNo"`
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// NotifyBulkCashInRequest is the request body for API Notify Bulk Cash
// In (Service Code 41, path .../{version}/emoney/bulk-cashin-notify —
// the path was previously misstated here as notify-bulk-cash-in with
// no recorded justification, corrected against research §1's own path
// table). This is a settlement callback the PJP receives, not a call this package
// makes — no calling function is provided, matching the package's
// established convention for inbound notification endpoints
// (InterbankBulkTransferNotificationRequest, Phase 12): a caller wires
// their own HTTP handler for this path, authenticates the inbound call
// with ServerVerifier.VerifyTransactionRequest, and json.Unmarshals the
// body into this type.
//
// The nearest research evidence points at this direction, not the
// outbound direction Phase 18 originally shipped (a santa-loop finding
// caught this): SubmitBulkCashInResponse.BulkID is bank-issued
// (mandatory in the Submit response), and research §5.6 line 174's
// shape — bulkId/partnerBulkId plus per-item responseCode/
// responseMessage — mirrors line 116's Service 21 (Interbank Bulk
// Transfer - Notification), which research explicitly annotates
// "settlement callback shape" and which this package already models
// as inbound-only. Nothing in §5.6 says the PJP sends this call
// outward (contrast research §5.6 line 154's explicit "Callback the
// PJP sends outward" for VA Notify Payment Intrabank, which is why
// that endpoint alone gets a calling function). BulkID and
// PartnerBulkID are mandatory per the Guides tab.
type NotifyBulkCashInRequest struct {
	BulkID        string                       `json:"bulkId"`
	PartnerBulkID string                       `json:"partnerBulkId"`
	BulkObject    []BulkCashInNotificationItem `json:"bulkObject"`
}

// NotifyBulkCashInResponse is the response body a caller sends back
// for API Notify Bulk Cash In. BulkID and PartnerBulkID are mandatory
// per the Guides tab (research §5.6 line 174's "Resp: envelope +
// bulkId/partnerBulkId M" — stronger, endpoint-specific evidence than
// Phase 12's Service 21 precedent, whose response fields carry no
// letter and are Optional by the package's default). BulkID's JSON tag
// is camelCase "bulkId" here — distinct from SubmitBulkCashInResponse's
// lowercase-d "bulkid" (see that type's doc comment for the unresolved
// research contradiction).
type NotifyBulkCashInResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	BulkID          string `json:"bulkId"`
	PartnerBulkID   string `json:"partnerBulkId"`
}
