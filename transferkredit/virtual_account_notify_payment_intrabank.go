package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// VANotifyPaymentIntrabankRequest is the request body for API VA -
// Notify Payment Intrabank (Service Code 34). Per research §5.3 line
// 154, this is "a callback the PJP sends outward" — the PJP initiates
// this call, unlike the RTGS/SKNBI/Interbank-Bulk notification
// endpoints (Phase 12), which the PJP only receives. It is therefore a
// normal calling function, not a struct-only inbound type.
// PartnerServiceID is documented "StringNumber" (a literal typo in the
// source table, §5.3 line 140) but shown quoted in this endpoint's own
// worked example, so it stays string — see the Phase 15 design doc's
// ambiguous-type table. CustomerNo is json.RawMessage (§5.3 line 135,
// bare-number worked example).
type VANotifyPaymentIntrabankRequest struct {
	PartnerServiceID   string          `json:"partnerServiceId"`
	CustomerNo         json.RawMessage `json:"customerNo"`
	VirtualAccountNo   string          `json:"virtualAccountNo"`
	InquiryRequestID   string          `json:"inquiryRequestId,omitempty"`
	PaymentRequestID   string          `json:"paymentRequestId,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	TrxDateTime        string          `json:"trxDateTime,omitempty"`
	PaymentStatus      string          `json:"paymentStatus,omitempty"`
	PaymentFlagReason  *LocalizedText  `json:"paymentFlagReason,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// VANotifyPaymentIntrabankData is the "virtualAccountdata" object
// (lowercase d, research §5.3 line 132) in
// VANotifyPaymentIntrabankResponse — mirrors
// VANotifyPaymentIntrabankRequest's fields, per §5.3 line 154
// ("response mirrors most request fields plus envelope").
type VANotifyPaymentIntrabankData struct {
	PartnerServiceID   string          `json:"partnerServiceId,omitempty"`
	CustomerNo         json.RawMessage `json:"customerNo,omitempty"`
	VirtualAccountNo   string          `json:"virtualAccountNo,omitempty"`
	InquiryRequestID   string          `json:"inquiryRequestId,omitempty"`
	PaymentRequestID   string          `json:"paymentRequestId,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	TrxDateTime        string          `json:"trxDateTime,omitempty"`
	PaymentStatus      string          `json:"paymentStatus,omitempty"`
	PaymentFlagReason  *LocalizedText  `json:"paymentFlagReason,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// VANotifyPaymentIntrabankResponse is the response body for API VA -
// Notify Payment Intrabank.
type VANotifyPaymentIntrabankResponse struct {
	ResponseCode       string                        `json:"responseCode"`
	ResponseMessage    string                        `json:"responseMessage"`
	VirtualAccountData *VANotifyPaymentIntrabankData `json:"virtualAccountdata,omitempty"`
}

// VANotifyPaymentIntrabank calls the SNAP VA - Notify Payment
// Intrabank endpoint (Service Code 34, path
// .../{version}/transfer-va/notify-payment-intrabank, HTTP POST — no
// method override). hb must already carry every field snap.HeaderBuilder
// needs except Body, which VANotifyPaymentIntrabank sets itself so the
// exact marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func VANotifyPaymentIntrabank(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req VANotifyPaymentIntrabankRequest) (VANotifyPaymentIntrabankResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return VANotifyPaymentIntrabankResponse{}, fmt.Errorf("snap: va notify payment intrabank: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return VANotifyPaymentIntrabankResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return VANotifyPaymentIntrabankResponse{}, fmt.Errorf("snap: va notify payment intrabank: %w", err)
	}

	var resp VANotifyPaymentIntrabankResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return VANotifyPaymentIntrabankResponse{}, fmt.Errorf("snap: va notify payment intrabank: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return VANotifyPaymentIntrabankResponse{}, errors.New("snap: va notify payment intrabank: response has no responseCode")
	}
	return resp, nil
}
