package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// RequestForPaymentRequest is the request body for API Request for
// Payment (Service Code 19). PartnerReferenceNo, BankCode,
// BeneficiaryAccountNo, BeneficiaryAccountName, ExpiredDatetime,
// SourceAccountNo, and SourceAccountName are mandatory per the Guides
// tab.
type RequestForPaymentRequest struct {
	PartnerReferenceNo     string          `json:"partnerReferenceNo"`
	BankCode               string          `json:"bankCode"`
	BeneficiaryAccountNo   string          `json:"beneficiaryAccountNo"`
	BeneficiaryAccountName string          `json:"beneficiaryAccountName"`
	Remark                 string          `json:"remark,omitempty"`
	ExpiredDatetime        string          `json:"expiredDatetime"`
	SourceAccountNo        string          `json:"sourceAccountNo"`
	SourceAccountName      string          `json:"sourceAccountName"`
	Currency               string          `json:"currency,omitempty"`
	Amount                 *snap.Money     `json:"amount,omitempty"`
	FeeType                string          `json:"feeType,omitempty"`
	AdditionalInfo         json.RawMessage `json:"additionalInfo,omitempty"`
}

// RequestForPaymentResponse is the response body for API Request for
// Payment.
type RequestForPaymentResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// RequestForPayment calls the SNAP Request for Payment endpoint
// (Service Code 19, path .../{version}/transfer-request-for-payment).
// hb must already carry every field snap.HeaderBuilder needs except Body,
// which RequestForPayment sets itself so the exact marshaled bytes are
// used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it —
// a fresh X-EXTERNAL-ID on retry risks a duplicate payment request.
func RequestForPayment(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req RequestForPaymentRequest) (RequestForPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return RequestForPaymentResponse{}, fmt.Errorf("snap: request for payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return RequestForPaymentResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return RequestForPaymentResponse{}, fmt.Errorf("snap: request for payment: %w", err)
	}

	var resp RequestForPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return RequestForPaymentResponse{}, fmt.Errorf("snap: request for payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return RequestForPaymentResponse{}, errors.New("snap: request for payment: response has no responseCode")
	}
	return resp, nil
}
