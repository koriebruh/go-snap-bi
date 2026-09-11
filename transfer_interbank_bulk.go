package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// InterbankBulkTransferRequest is the request body for API Interbank
// Bulk Transfer (Service Code 20). CustomerReference, SourceAccountNo,
// TransactionDate, and BulkObject are mandatory per the Guides tab.
type InterbankBulkTransferRequest struct {
	PartnerBulkID     string                      `json:"partnerBulkId,omitempty"`
	Currency          string                      `json:"currency,omitempty"`
	CustomerReference string                      `json:"customerReference"`
	FeeType           string                      `json:"feeType,omitempty"`
	Remark            string                      `json:"remark,omitempty"`
	SourceAccountNo   string                      `json:"sourceAccountNo"`
	TransactionDate   string                      `json:"transactionDate"`
	BulkObject        []InterbankBulkTransferItem `json:"bulkObject"`
	AdditionalInfo    json.RawMessage             `json:"additionalInfo,omitempty"`
}

// InterbankBulkTransferResponse is the response body for API Interbank
// Bulk Transfer.
type InterbankBulkTransferResponse struct {
	ResponseCode    string          `json:"responseCode"`
	ResponseMessage string          `json:"responseMessage"`
	BulkID          string          `json:"bulkId,omitempty"`
	PartnerBulkID   string          `json:"partnerBulkId,omitempty"`
	AdditionalInfo  json.RawMessage `json:"additionalInfo,omitempty"`
}

// InterbankBulkTransfer calls the SNAP Interbank Bulk Transfer endpoint
// (Service Code 20, path .../{version}/transfer-interbank-bulk). hb
// must already carry every field HeaderBuilder needs except Body, which
// InterbankBulkTransfer sets itself so the exact marshaled bytes are
// used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it —
// a fresh X-EXTERNAL-ID on retry risks a duplicate bulk transfer.
func InterbankBulkTransfer(ctx context.Context, t *Transport, hb HeaderBuilder, req InterbankBulkTransferRequest) (InterbankBulkTransferResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return InterbankBulkTransferResponse{}, fmt.Errorf("snap: interbank bulk transfer: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return InterbankBulkTransferResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return InterbankBulkTransferResponse{}, fmt.Errorf("snap: interbank bulk transfer: %w", err)
	}

	var resp InterbankBulkTransferResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return InterbankBulkTransferResponse{}, fmt.Errorf("snap: interbank bulk transfer: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return InterbankBulkTransferResponse{}, errors.New("snap: interbank bulk transfer: response has no responseCode")
	}
	return resp, nil
}
