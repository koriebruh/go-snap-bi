package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// BulkCashInNotificationItem is one entry in Notify Bulk Cash In's
// request "bulkObject[]" array — a settlement-result shape, distinct
// from BulkCashInItem's transfer-instruction shape (per the "distinct
// types per service code" convention: this type has 7 fields vs.
// Phase 12's InterbankBulkTransferNotificationItem's 3, so it is not a
// reuse of that type either). CustomerNumber, ReferenceNo,
// PartnerReferenceNo, ResponseCode, and ResponseMessage are all
// mandatory per the Guides tab (research §5.6 line 174's slash
// notation lists each pair as jointly Mandatory, not either/or — see
// the Phase 18 design doc).
type BulkCashInNotificationItem struct {
	CustomerNumber     string          `json:"customerNumber"`
	CustomerName       string          `json:"customerName,omitempty"`
	Amount             *Money          `json:"amount,omitempty"`
	ReferenceNo        string          `json:"referenceNo"`
	PartnerReferenceNo string          `json:"partnerReferenceNo"`
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// NotifyBulkCashInRequest is the request body for API Notify Bulk Cash
// In (Service Code 41). BulkID and PartnerBulkID are both mandatory
// per the Guides tab (research §5.6 line 174's "bulkId/partnerBulkId
// M").
type NotifyBulkCashInRequest struct {
	BulkID        string                       `json:"bulkId"`
	PartnerBulkID string                       `json:"partnerBulkId"`
	BulkObject    []BulkCashInNotificationItem `json:"bulkObject,omitempty"`
}

// NotifyBulkCashInResponse is the response body for API Notify Bulk
// Cash In. BulkID's JSON tag is camelCase "bulkId" here — distinct
// from SubmitBulkCashInResponse's lowercase-d "bulkid" (see that
// type's doc comment for the unresolved research contradiction).
type NotifyBulkCashInResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	BulkID          string `json:"bulkId"`
	PartnerBulkID   string `json:"partnerBulkId"`
}

// NotifyBulkCashIn calls the SNAP Notify Bulk Cash In endpoint
// (Service Code 41, path .../{version}/notify-bulk-cash-in, HTTP POST
// — no method override). hb must already carry every field
// HeaderBuilder needs except Body, which NotifyBulkCashIn sets itself
// so the exact marshaled bytes are used for both signing and the wire
// body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func NotifyBulkCashIn(ctx context.Context, t *Transport, hb HeaderBuilder, req NotifyBulkCashInRequest) (NotifyBulkCashInResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return NotifyBulkCashInResponse{}, fmt.Errorf("snap: notify bulk cash in: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return NotifyBulkCashInResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return NotifyBulkCashInResponse{}, fmt.Errorf("snap: notify bulk cash in: %w", err)
	}

	var resp NotifyBulkCashInResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return NotifyBulkCashInResponse{}, fmt.Errorf("snap: notify bulk cash in: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return NotifyBulkCashInResponse{}, errors.New("snap: notify bulk cash in: response has no responseCode")
	}
	return resp, nil
}
