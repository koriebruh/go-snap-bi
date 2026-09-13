package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// TransactionHistoryDetailRequest is the request body for API Transaction
// History Detail (Service Code 13, path
// .../{version}/transaction-history-detail).
type TransactionHistoryDetailRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// TransactionHistoryDetailResponse is the response body for API
// Transaction History Detail. Amount and RefundAmount are Mandatory
// containers with Mandatory members, so both are modeled as plain
// (non-pointer) snap.Money, per this package's convention (see
// transaction_history.go's TransactionDetail.Amount).
type TransactionHistoryDetailResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	Amount             snap.Money      `json:"amount"`
	CancelledTime      string          `json:"cancelledTime,omitempty"`
	DateTime           string          `json:"dateTime"`
	RefundAmount       snap.Money      `json:"refundAmount"`
	Remark             string          `json:"remark,omitempty"`
	SourceOfFunds      []SourceOfFund  `json:"sourceOfFunds,omitempty"`
	Status             string          `json:"status"`
	Type               string          `json:"type"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// TransactionHistoryDetail calls the SNAP Transaction History Detail
// endpoint (Service Code 13, path
// .../{version}/transaction-history-detail). hb must already carry every
// field snap.HeaderBuilder needs except Body, which
// TransactionHistoryDetail sets itself so the exact marshaled bytes are
// used for both signing and the wire body.
func TransactionHistoryDetail(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req TransactionHistoryDetailRequest) (TransactionHistoryDetailResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TransactionHistoryDetailResponse{}, fmt.Errorf("snap: transaction history detail: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return TransactionHistoryDetailResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return TransactionHistoryDetailResponse{}, fmt.Errorf("snap: transaction history detail: %w", err)
	}

	var resp TransactionHistoryDetailResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return TransactionHistoryDetailResponse{}, fmt.Errorf("snap: transaction history detail: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return TransactionHistoryDetailResponse{}, errors.New("snap: transaction history detail: response has no responseCode")
	}
	return resp, nil
}
