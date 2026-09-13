package transferdebit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// AuthPaymentRequest is the request body for API Auth Payment (Service
// Code 63, path .../{version}/auth/payment). This places a hold on
// funds; Capture (65) later charges some or all of the held amount,
// Void (67) releases held-but-uncaptured funds.
//
// Items is documented as a list of purchased goods with no full
// item-level schema given anywhere beyond the worked example's
// goodsId/price/category/unit/quantity — modeled as json.RawMessage,
// matching the package's established treatment of genuinely untyped
// fields (the same rationale as AdditionalInfo; also already used for
// AccountBindingRequest.AdditionalData, VerifyOTPResponse.QParams, and
// CPMPaymentRequest.Items, the last being the exact same field name
// and shape precedent).
type AuthPaymentRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo"`
	MerchantID         string          `json:"merchantId"`
	SubMerchantID      string          `json:"subMerchantId,omitempty"`
	Amount             *snap.Money     `json:"amount,omitempty"`
	FeeType            string          `json:"feeType,omitempty"`
	MCC                string          `json:"mcc,omitempty"`
	ProductCode        string          `json:"productCode,omitempty"`
	Title              string          `json:"title"`
	Items              json.RawMessage `json:"items,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthPaymentResponse is the response body for API Auth Payment.
// ReferenceNo is Conditional (success only). Research leaves the
// Amount container unmarked here, same as the request side, but marks
// both its value/currency members Mandatory; Amount is modeled as a
// plain (non-pointer) snap.Money — unlike the request side's *snap.Money — since
// this is the success response for a hold that was just placed, and an
// amount is expected to always be present on it. This is a narrower
// judgment call than the Optional-container-marked cases elsewhere in
// the package (compare AuthPaymentQueryResponse.Amount, which research
// explicitly marks amount O and which is modeled as *snap.Money).
type AuthPaymentResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	ReferenceNo        string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	Amount             snap.Money      `json:"amount"`
	PaidTime           string          `json:"paidTime"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// AuthPayment calls the SNAP Auth Payment endpoint (Service Code 63,
// HTTP POST — no method override; research's Overview tab marks this
// GET, but every worked example across all 7 Auth Payment endpoints is
// POST-shaped with a mandatory JSON body, and this package's signing
// path has no representation for a body-carrying GET). hb must already
// carry every field snap.HeaderBuilder needs except Body, which AuthPayment
// sets itself so the exact marshaled bytes are used for both signing
// and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func AuthPayment(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req AuthPaymentRequest) (AuthPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AuthPaymentResponse{}, fmt.Errorf("snap: auth payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AuthPaymentResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AuthPaymentResponse{}, fmt.Errorf("snap: auth payment: %w", err)
	}

	var resp AuthPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AuthPaymentResponse{}, fmt.Errorf("snap: auth payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AuthPaymentResponse{}, errors.New("snap: auth payment: response has no responseCode")
	}
	return resp, nil
}
