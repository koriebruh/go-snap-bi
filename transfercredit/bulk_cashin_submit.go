package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// BulkCashInItem is one entry in Submit Bulk Cash In's request
// "bulkObject[]" array. AccountNumber and PartnerReferenceNo are
// mandatory per the Guides tab.
type BulkCashInItem struct {
	AccountNumber      string          `json:"accountNumber"`
	AccountName        string          `json:"accountName,omitempty"`
	Amount             *snap.Money     `json:"amount,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// SubmitBulkCashInRequest is the request body for API Submit Bulk Cash
// In (Service Code 40). TransactionDate is the only top-level
// mandatory field per the Guides tab.
type SubmitBulkCashInRequest struct {
	PartnerBulkID   string           `json:"partnerBulkId,omitempty"`
	TransactionDate string           `json:"transactionDate"`
	Currency        string           `json:"currency,omitempty"`
	BulkObject      []BulkCashInItem `json:"bulkObject,omitempty"`
	FeeType         string           `json:"feeType,omitempty"`
	AdditionalInfo  json.RawMessage  `json:"additionalInfo,omitempty"`
}

// SubmitBulkCashInResponse is the response body for API Submit Bulk
// Cash In. BulkID's JSON tag is "bulkid" (lowercase d), matching this
// endpoint's own field table (research §5.6 line 172) — the research
// doc itself flags this as "likely a typo for bulkId, unresolved," but
// no worked example exists to break the tie, so this package models
// the literal, only-available documented casing rather than guessing
// at the correction. Endpoint 41 (NotifyBulkCashInResponse) uses
// camelCase "bulkId" per its own, separate field description.
type SubmitBulkCashInResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	BulkID          string `json:"bulkid"`
	PartnerBulkID   string `json:"partnerBulkId,omitempty"`
}

// SubmitBulkCashIn calls the SNAP Submit Bulk Cash In endpoint
// (Service Code 40, path .../{version}/submit-bulk-cash-in, HTTP POST
// — no method override). hb must already carry every field
// snap.HeaderBuilder needs except Body, which SubmitBulkCashIn sets itself
// so the exact marshaled bytes are used for both signing and the wire
// body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func SubmitBulkCashIn(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req SubmitBulkCashInRequest) (SubmitBulkCashInResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return SubmitBulkCashInResponse{}, fmt.Errorf("snap: submit bulk cash in: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return SubmitBulkCashInResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return SubmitBulkCashInResponse{}, fmt.Errorf("snap: submit bulk cash in: %w", err)
	}

	var resp SubmitBulkCashInResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return SubmitBulkCashInResponse{}, fmt.Errorf("snap: submit bulk cash in: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return SubmitBulkCashInResponse{}, errors.New("snap: submit bulk cash in: response has no responseCode")
	}
	return resp, nil
}
