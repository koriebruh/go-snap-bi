package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// VAInquiryRequest is the request body for API VA - VA Inquiry (Service
// Code 24). PartnerServiceID, CustomerNo, VirtualAccountNo, and
// InquiryRequestID are mandatory per the Guides tab. CustomerNo is
// json.RawMessage rather than string: it is documented String(20) but
// appears as a bare JSON number in this endpoint's sibling VA Inquiry
// Status (26) worked example (research §5.3) — see the Phase 14 design
// doc's ambiguous-type table. A caller setting CustomerNo or
// ChannelCode must supply a complete JSON value (e.g.
// json.RawMessage(`"98765"`) or json.RawMessage(`98765`)), not a bare
// Go string.
type VAInquiryRequest struct {
	PartnerServiceID      string          `json:"partnerServiceId"`
	CustomerNo            json.RawMessage `json:"customerNo"`
	VirtualAccountNo      string          `json:"virtualAccountNo"`
	TrxDateInit           string          `json:"trxDateInit,omitempty"`
	ChannelCode           json.RawMessage `json:"channelCode,omitempty"`
	Language              string          `json:"language,omitempty"`
	HashedSourceAccountNo string          `json:"hashedSourceAccountNo,omitempty"`
	SourceBankCode        string          `json:"sourceBankCode,omitempty"`
	PassApp               string          `json:"passApp,omitempty"`
	InquiryRequestID      string          `json:"inquiryRequestId"`
	AdditionalInfo        json.RawMessage `json:"additionalInfo,omitempty"`
}

// VAInquiryData is the "virtualAccountData" object in VAInquiryResponse.
// Fields beyond the identity triple carry no M/O letter in the Guides
// tab, so all are Optional (omitempty); InquiryReason, TotalAmount, and
// FeeAmount are pointers since omitempty has no effect on a non-pointer
// struct value.
type VAInquiryData struct {
	PartnerServiceID      string          `json:"partnerServiceId,omitempty"`
	CustomerNo            json.RawMessage `json:"customerNo,omitempty"`
	VirtualAccountNo      string          `json:"virtualAccountNo,omitempty"`
	InquiryStatus         string          `json:"inquiryStatus,omitempty"`
	InquiryReason         *LocalizedText  `json:"inquiryReason,omitempty"`
	VirtualAccountName    string          `json:"virtualAccountName,omitempty"`
	VirtualAccountEmail   string          `json:"virtualAccountEmail,omitempty"`
	VirtualAccountPhone   string          `json:"virtualAccountPhone,omitempty"`
	InquiryRequestID      string          `json:"inquiryRequestId,omitempty"`
	TotalAmount           *snap.Money     `json:"totalAmount,omitempty"`
	SubCompany            string          `json:"subCompany,omitempty"`
	BillDetails           []BillDetail    `json:"billDetails,omitempty"`
	FreeTexts             []LocalizedText `json:"freeTexts,omitempty"`
	VirtualAccountTrxType string          `json:"virtualAccountTrxType,omitempty"`
	FeeAmount             *snap.Money     `json:"feeAmount,omitempty"`
	AdditionalInfo        json.RawMessage `json:"additionalInfo,omitempty"`
}

// VAInquiryResponse is the response body for API VA - VA Inquiry.
type VAInquiryResponse struct {
	ResponseCode       string         `json:"responseCode"`
	ResponseMessage    string         `json:"responseMessage"`
	VirtualAccountData *VAInquiryData `json:"virtualAccountData,omitempty"`
}

// VAInquiry calls the SNAP VA - VA Inquiry endpoint (Service Code 24,
// path .../{version}/transfer-va/inquiry, HTTP POST — no method
// override, unlike Phase 13's PUT/DELETE endpoints). hb must already
// carry every field snap.HeaderBuilder needs except Body, which VAInquiry
// sets itself so the exact marshaled bytes are used for both signing
// and the wire body.
func VAInquiry(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req VAInquiryRequest) (VAInquiryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return VAInquiryResponse{}, fmt.Errorf("snap: va inquiry: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return VAInquiryResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return VAInquiryResponse{}, fmt.Errorf("snap: va inquiry: %w", err)
	}

	var resp VAInquiryResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return VAInquiryResponse{}, fmt.Errorf("snap: va inquiry: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return VAInquiryResponse{}, errors.New("snap: va inquiry: response has no responseCode")
	}
	return resp, nil
}
