package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// VAInquiryStatusRequest is the request body for API VA - VA Inquiry
// Status (Service Code 26). PartnerServiceID, CustomerNo, and
// VirtualAccountNo (the identity triple) are mandatory per the Guides
// tab; InquiryRequestID and PaymentRequestID are both Optional/
// Conditional (omitempty). Research §5.3 notes an unresolved
// contradiction: the portal text says "if InquiryRequestID is not
// sent, returns array based on virtualAccountNo," but the response is
// documented elsewhere as a single Object — this package models the
// single-Object reading (see the Phase 14 design doc) and does not
// attempt to guess at the array reading.
type VAInquiryStatusRequest struct {
	PartnerServiceID string          `json:"partnerServiceId"`
	CustomerNo       json.RawMessage `json:"customerNo"`
	VirtualAccountNo string          `json:"virtualAccountNo"`
	InquiryRequestID string          `json:"inquiryRequestId,omitempty"`
	PaymentRequestID string          `json:"paymentRequestId,omitempty"`
}

// VAInquiryStatusData is the "virtualAccountData" object in
// VAInquiryStatusResponse — per research §5.3, "mirrors VA Payment's
// response shape plus transactionDate." It is a distinct type from
// VAPaymentData (not a reuse), per the package's "distinct types per
// service code even for identical/near-identical shapes" convention;
// transfer_shared_types_test.go guards the two against field/tag drift.
type VAInquiryStatusData struct {
	PartnerServiceID        string          `json:"partnerServiceId,omitempty"`
	CustomerNo              json.RawMessage `json:"customerNo,omitempty"`
	VirtualAccountNo        string          `json:"virtualAccountNo,omitempty"`
	TrxID                   string          `json:"trxId,omitempty"`
	PaymentRequestID        string          `json:"paymentRequestId,omitempty"`
	ChannelCode             json.RawMessage `json:"channelCode,omitempty"`
	HashedSourceAccountNo   string          `json:"hashedSourceAccountNo,omitempty"`
	SourceBankCode          string          `json:"sourceBankCode,omitempty"`
	PaidAmount              *Money          `json:"paidAmount,omitempty"`
	CumulativePaymentAmount *Money          `json:"cumulativePaymentAmount,omitempty"`
	PaidBills               string          `json:"paidBills,omitempty"`
	TotalAmount             *Money          `json:"totalAmount,omitempty"`
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
	TransactionDate         string          `json:"transactionDate,omitempty"`
}

// VAInquiryStatusResponse is the response body for API VA - VA Inquiry
// Status.
type VAInquiryStatusResponse struct {
	ResponseCode       string               `json:"responseCode"`
	ResponseMessage    string               `json:"responseMessage"`
	VirtualAccountData *VAInquiryStatusData `json:"virtualAccountData,omitempty"`
}

// VAInquiryStatus calls the SNAP VA - VA Inquiry Status endpoint
// (Service Code 26, path .../{version}/transfer-va/inquiry-status,
// HTTP POST — no method override). hb must already carry every field
// HeaderBuilder needs except Body, which VAInquiryStatus sets itself so
// the exact marshaled bytes are used for both signing and the wire
// body.
func VAInquiryStatus(ctx context.Context, t *Transport, hb HeaderBuilder, req VAInquiryStatusRequest) (VAInquiryStatusResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return VAInquiryStatusResponse{}, fmt.Errorf("snap: va inquiry status: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return VAInquiryStatusResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return VAInquiryStatusResponse{}, fmt.Errorf("snap: va inquiry status: %w", err)
	}

	var resp VAInquiryStatusResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return VAInquiryStatusResponse{}, fmt.Errorf("snap: va inquiry status: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return VAInquiryStatusResponse{}, errors.New("snap: va inquiry status: response has no responseCode")
	}
	return resp, nil
}
