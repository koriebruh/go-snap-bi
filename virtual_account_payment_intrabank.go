package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// VAPaymentIntrabankRequest is the request body for API VA - Payment
// Intrabank (Service Code 33). PartnerServiceID, CustomerNo,
// VirtualAccountNo, PartnerReferenceNo, and PaidAmount are mandatory
// per the Guides tab. CustomerNo is json.RawMessage (research §5.3
// line 135, bare-number worked example). ReferenceNo is
// json.RawMessage too: this endpoint's own worked request shows it as
// a bare number (§5.3 line 137), even though its own response shows
// it quoted — see the Phase 15 design doc's ambiguous-type table.
// PaymentStatus is a free-text status string, explicitly distinct from
// the package's transactionStatus 2-digit enum (research §5.3 line
// 153).
type VAPaymentIntrabankRequest struct {
	PartnerServiceID        string          `json:"partnerServiceId"`
	CustomerNo              json.RawMessage `json:"customerNo"`
	VirtualAccountNo        string          `json:"virtualAccountNo"`
	SourceAccountNo         string          `json:"sourceAccountNo,omitempty"`
	SourceAccountType       string          `json:"sourceAccountType,omitempty"`
	InquiryRequestID        string          `json:"inquiryRequestId,omitempty"`
	PartnerReferenceNo      string          `json:"partnerReferenceNo"`
	PaidAmount              Money           `json:"paidAmount"`
	CumulativePaymentAmount *Money          `json:"cumulativePaymentAmount,omitempty"`
	PaidBills               string          `json:"paidBills,omitempty"`
	PaymentStatus           string          `json:"paymentStatus,omitempty"`
	ReferenceNo             json.RawMessage `json:"referenceNo,omitempty"`
	AdditionalInfo          json.RawMessage `json:"additionalInfo,omitempty"`
}

// VAPaymentIntrabankData is the "virtualAccountdata" object (lowercase
// d, research §5.3 line 132) in VAPaymentIntrabankResponse — mirrors
// VAPaymentIntrabankRequest's fields, per the same
// request-mirrors-into-response convention used for VAPaymentData in
// Phase 14.
type VAPaymentIntrabankData struct {
	PartnerServiceID        string          `json:"partnerServiceId,omitempty"`
	CustomerNo              json.RawMessage `json:"customerNo,omitempty"`
	VirtualAccountNo        string          `json:"virtualAccountNo,omitempty"`
	SourceAccountNo         string          `json:"sourceAccountNo,omitempty"`
	SourceAccountType       string          `json:"sourceAccountType,omitempty"`
	InquiryRequestID        string          `json:"inquiryRequestId,omitempty"`
	PartnerReferenceNo      string          `json:"partnerReferenceNo,omitempty"`
	PaidAmount              *Money          `json:"paidAmount,omitempty"`
	CumulativePaymentAmount *Money          `json:"cumulativePaymentAmount,omitempty"`
	PaidBills               string          `json:"paidBills,omitempty"`
	PaymentStatus           string          `json:"paymentStatus,omitempty"`
	ReferenceNo             json.RawMessage `json:"referenceNo,omitempty"`
	AdditionalInfo          json.RawMessage `json:"additionalInfo,omitempty"`
}

// VAPaymentIntrabankResponse is the response body for API VA - Payment
// Intrabank.
type VAPaymentIntrabankResponse struct {
	ResponseCode       string                  `json:"responseCode"`
	ResponseMessage    string                  `json:"responseMessage"`
	VirtualAccountData *VAPaymentIntrabankData `json:"virtualAccountdata,omitempty"`
}

// VAPaymentIntrabank calls the SNAP VA - Payment Intrabank endpoint
// (Service Code 33, path .../{version}/transfer-va/payment-intrabank,
// HTTP POST — no method override). hb must already carry every field
// HeaderBuilder needs except Body, which VAPaymentIntrabank sets
// itself so the exact marshaled bytes are used for both signing and
// the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func VAPaymentIntrabank(ctx context.Context, t *Transport, hb HeaderBuilder, req VAPaymentIntrabankRequest) (VAPaymentIntrabankResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return VAPaymentIntrabankResponse{}, fmt.Errorf("snap: va payment intrabank: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return VAPaymentIntrabankResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return VAPaymentIntrabankResponse{}, fmt.Errorf("snap: va payment intrabank: %w", err)
	}

	var resp VAPaymentIntrabankResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return VAPaymentIntrabankResponse{}, fmt.Errorf("snap: va payment intrabank: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return VAPaymentIntrabankResponse{}, errors.New("snap: va payment intrabank: response has no responseCode")
	}
	return resp, nil
}
