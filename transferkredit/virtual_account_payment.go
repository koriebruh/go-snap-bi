package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// VAPaymentRequest is the request body for API VA - VA Payment (Service
// Code 25). PartnerServiceID, CustomerNo, VirtualAccountNo,
// PaymentRequestID, and PaidAmount are mandatory per the Guides tab.
// TrxID is Conditional — "mandatory if from Create VA" per research
// §5.3, a data-dependent condition the type system can't express, so
// it's Optional (omitempty) here. CustomerNo, ChannelCode, and
// PaymentType are json.RawMessage: see the Phase 14 design doc's
// ambiguous-type table (PaymentType is documented String(1) but every
// worked example shows a bare number).
type VAPaymentRequest struct {
	PartnerServiceID        string          `json:"partnerServiceId"`
	CustomerNo              json.RawMessage `json:"customerNo"`
	VirtualAccountNo        string          `json:"virtualAccountNo"`
	TrxID                   string          `json:"trxId,omitempty"`
	PaymentRequestID        string          `json:"paymentRequestId"`
	ChannelCode             json.RawMessage `json:"channelCode,omitempty"`
	HashedSourceAccountNo   string          `json:"hashedSourceAccountNo,omitempty"`
	SourceBankCode          string          `json:"sourceBankCode,omitempty"`
	PaidAmount              snap.Money      `json:"paidAmount"`
	CumulativePaymentAmount *snap.Money     `json:"cumulativePaymentAmount,omitempty"`
	PaidBills               string          `json:"paidBills,omitempty"`
	TotalAmount             *snap.Money     `json:"totalAmount,omitempty"`
	TrxDateTime             string          `json:"trxDateTime,omitempty"`
	ReferenceNo             string          `json:"referenceNo,omitempty"`
	JournalNum              string          `json:"journalNum,omitempty"`
	PaymentType             json.RawMessage `json:"paymentType,omitempty"`
	FlagAdvise              string          `json:"flagAdvise,omitempty"`
	SubCompany              string          `json:"subCompany,omitempty"`
	BillDetails             []BillDetail    `json:"billDetails,omitempty"`
	FreeTexts               []LocalizedText `json:"freeTexts,omitempty"`
	AdditionalInfo          json.RawMessage `json:"additionalInfo,omitempty"`
}

// VAPaymentData is the "virtualAccountData" object in VAPaymentResponse
// — per research §5.3, the response mirrors the full request field set
// plus PaymentFlagReason and PaymentFlagStatus. Per-bill status/reason
// need no new field: BillDetail already carries Status and Reason
// (Phase 13).
type VAPaymentData struct {
	PartnerServiceID        string          `json:"partnerServiceId,omitempty"`
	CustomerNo              json.RawMessage `json:"customerNo,omitempty"`
	VirtualAccountNo        string          `json:"virtualAccountNo,omitempty"`
	TrxID                   string          `json:"trxId,omitempty"`
	PaymentRequestID        string          `json:"paymentRequestId,omitempty"`
	ChannelCode             json.RawMessage `json:"channelCode,omitempty"`
	HashedSourceAccountNo   string          `json:"hashedSourceAccountNo,omitempty"`
	SourceBankCode          string          `json:"sourceBankCode,omitempty"`
	PaidAmount              *snap.Money     `json:"paidAmount,omitempty"`
	CumulativePaymentAmount *snap.Money     `json:"cumulativePaymentAmount,omitempty"`
	PaidBills               string          `json:"paidBills,omitempty"`
	TotalAmount             *snap.Money     `json:"totalAmount,omitempty"`
	TrxDateTime             string          `json:"trxDateTime,omitempty"`
	ReferenceNo             string          `json:"referenceNo,omitempty"`
	JournalNum              string          `json:"journalNum,omitempty"`
	PaymentType             json.RawMessage `json:"paymentType,omitempty"`
	FlagAdvise              string          `json:"flagAdvise,omitempty"`
	SubCompany              string          `json:"subCompany,omitempty"`
	BillDetails             []BillDetail    `json:"billDetails,omitempty"`
	FreeTexts               []LocalizedText `json:"freeTexts,omitempty"`
	PaymentFlagReason       *LocalizedText  `json:"paymentFlagReason,omitempty"`
	PaymentFlagStatus       string          `json:"paymentFlagStatus,omitempty"`
	AdditionalInfo          json.RawMessage `json:"additionalInfo,omitempty"`
}

// VAPaymentResponse is the response body for API VA - VA Payment.
type VAPaymentResponse struct {
	ResponseCode       string         `json:"responseCode"`
	ResponseMessage    string         `json:"responseMessage"`
	VirtualAccountData *VAPaymentData `json:"virtualAccountData,omitempty"`
}

// VAPayment calls the SNAP VA - VA Payment endpoint (Service Code 25,
// path .../{version}/transfer-va/payment, HTTP POST — no method
// override). hb must already carry every field snap.HeaderBuilder needs
// except Body, which VAPayment sets itself so the exact marshaled bytes
// are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func VAPayment(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req VAPaymentRequest) (VAPaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return VAPaymentResponse{}, fmt.Errorf("snap: va payment: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return VAPaymentResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return VAPaymentResponse{}, fmt.Errorf("snap: va payment: %w", err)
	}

	var resp VAPaymentResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return VAPaymentResponse{}, fmt.Errorf("snap: va payment: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return VAPaymentResponse{}, errors.New("snap: va payment: response has no responseCode")
	}
	return resp, nil
}
