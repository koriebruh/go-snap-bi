package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// AuthPaymentQueryRequest is the request body for API Payment Query
// (Service Code 64, path .../{version}/auth/query). No field is
// Mandatory.
type AuthPaymentQueryRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	MerchantID                 string          `json:"merchantId,omitempty"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthPaymentQueryResponse is the response body for API Payment Query.
// PaidTime and LatestTransactionStatus are Mandatory. Amount is
// modeled as *Money since research explicitly marks the container
// Optional (amount O, members M) — the package's established rule for
// an Optional container is *Money regardless of its members' own
// markers (see e.g. TransactionStatusInquiryBankResponse.Amount).
//
// The worked response example in research key-cases
// originalpartnerReferenceNo (lowercase p) against the field table's
// originalPartnerReferenceNo (research §6 item 2). This field is
// request-Optional and only echoed back in the response here, and
// Go's encoding/json unmarshal is case-insensitive, so either wire
// casing populates OriginalPartnerReferenceNo correctly — the field
// table's casing is used for the Go tag, consistent with every other
// occurrence of this field name across the package.
type AuthPaymentQueryResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	Amount                     *Money          `json:"amount,omitempty"`
	PaidTime                   string          `json:"paidTime"`
	LatestTransactionStatus    string          `json:"latestTransactionStatus"`
	TransactionStatusDesc      string          `json:"transactionStatusDesc,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthPaymentQuery calls the SNAP Payment Query endpoint (Service Code
// 64, HTTP POST — no method override, same GET/POST resolution as
// AuthPayment). hb must already carry every field HeaderBuilder needs
// except Body, which AuthPaymentQuery sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
func AuthPaymentQuery(ctx context.Context, t *Transport, hb HeaderBuilder, req AuthPaymentQueryRequest) (AuthPaymentQueryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AuthPaymentQueryResponse{}, fmt.Errorf("snap: auth payment query: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AuthPaymentQueryResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AuthPaymentQueryResponse{}, fmt.Errorf("snap: auth payment query: %w", err)
	}

	var resp AuthPaymentQueryResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AuthPaymentQueryResponse{}, fmt.Errorf("snap: auth payment query: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AuthPaymentQueryResponse{}, errors.New("snap: auth payment query: response has no responseCode")
	}
	return resp, nil
}
