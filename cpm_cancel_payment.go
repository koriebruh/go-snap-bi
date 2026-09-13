package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CPMCancelPaymentRequest is the request body for API Cancel Payment
// (Service Code 62). OriginalPartnerReferenceNo is the only Mandatory
// field per research §5.2's own field table.
//
// Research's summary prose claims this endpoint is "structurally
// identical to QRMPMCancelPaymentRequest/Response" (Phase 24) — that
// claim does not hold: QRMPMCancelPaymentRequest's Mandatory fields
// are MerchantID/Reason (all three originalX Optional there too), a
// disjoint set from this endpoint's own Mandatory
// OriginalPartnerReferenceNo. The per-field table is authoritative
// here, not the summary sentence — see the design doc for the full
// comparison. Modeled as an independent type, with no
// reflection-drift-guard against QRMPMCancelPaymentRequest since they
// are not the same shape.
type CPMCancelPaymentRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	MerchantID                 string          `json:"merchantId,omitempty"`
	SubMerchantID              string          `json:"subMerchantId,omitempty"`
	ExternalStoreID            string          `json:"externalStoreId,omitempty"`
	Amount                     *Money          `json:"amount,omitempty"`
	Reason                     string          `json:"reason,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMCancelPaymentResponse is the response body for API Cancel
// Payment. OriginalReferenceNo is Conditional (success only).
// CancelTime is Conditional ("filled if successful"), matching
// QRMPMCancelPaymentResponse.CancelTime's identical treatment — but
// this response also carries originalX fields that
// QRMPMCancelPaymentResponse does not have at all, so it is not a
// shared/identical shape with that sibling either.
type CPMCancelPaymentResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	CancelTime                 string          `json:"cancelTime,omitempty"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// CPMCancelPayment calls the SNAP Cancel Payment endpoint (Service
// Code 62, path .../{version}/qr/qr-cpm-cancel, HTTP POST — no method
// override). hb must already carry every field HeaderBuilder needs
// except Body, which CPMCancelPayment sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func CPMCancelPayment(ctx context.Context, t *Transport, hb HeaderBuilder, req CPMCancelPaymentRequest) (CPMCancelPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CPMCancelPaymentResponse{}, fmt.Errorf("snap: cpm cancel payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CPMCancelPaymentResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CPMCancelPaymentResponse{}, fmt.Errorf("snap: cpm cancel payment: %w", err)
	}

	var resp CPMCancelPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CPMCancelPaymentResponse{}, fmt.Errorf("snap: cpm cancel payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CPMCancelPaymentResponse{}, errors.New("snap: cpm cancel payment: response has no responseCode")
	}
	return resp, nil
}
