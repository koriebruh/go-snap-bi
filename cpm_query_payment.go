package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CPMQueryPaymentRequest is the request body for API Query Payment
// (Service Code 61). No field is Mandatory per research §5.2.
type CPMQueryPaymentRequest struct {
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	MerchantID                 string          `json:"merchantId,omitempty"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMQueryPaymentResponse is the response body for API Query Payment.
// LatestTransactionStatus and PaidTime are Mandatory per research
// §5.2. OriginalReferenceNo is Conditional (success only).
type CPMQueryPaymentResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	Title                      string          `json:"title,omitempty"`
	LatestTransactionStatus    string          `json:"latestTransactionStatus"`
	TransactionStatusDesc      string          `json:"transactionStatusDesc,omitempty"`
	PaidTime                   string          `json:"paidTime"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMQueryPayment calls the SNAP Query Payment endpoint (Service Code
// 61, path .../{version}/qr/qr-cpm-query, HTTP POST — no method
// override). hb must already carry every field HeaderBuilder needs
// except Body, which CPMQueryPayment sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
//
// This is a read-only status query and carries no non-idempotency
// note, matching TransactionStatusInquiryBank's precedent.
func CPMQueryPayment(ctx context.Context, t *Transport, hb HeaderBuilder, req CPMQueryPaymentRequest) (CPMQueryPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CPMQueryPaymentResponse{}, fmt.Errorf("snap: cpm query payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CPMQueryPaymentResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CPMQueryPaymentResponse{}, fmt.Errorf("snap: cpm query payment: %w", err)
	}

	var resp CPMQueryPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CPMQueryPaymentResponse{}, fmt.Errorf("snap: cpm query payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CPMQueryPaymentResponse{}, errors.New("snap: cpm query payment: response has no responseCode")
	}
	return resp, nil
}
