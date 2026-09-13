package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// InquiryVARequest is the request body for API VA - Inquiry VA (Service
// Code 30). PartnerServiceID, CustomerNo, VirtualAccountNo, and TrxID
// are mandatory per the Guides tab.
type InquiryVARequest struct {
	PartnerServiceID string          `json:"partnerServiceId"`
	CustomerNo       string          `json:"customerNo"`
	VirtualAccountNo string          `json:"virtualAccountNo"`
	TrxID            string          `json:"trxId"`
	AdditionalInfo   json.RawMessage `json:"additionalInfo,omitempty"`
}

// InquiryVAData is the "virtualAccountData" object in InquiryVAResponse
// — per the research doc, "full VA data object (same shape as Update VA
// response)", the same field set as UpdateVAData, under its own type
// name.
type InquiryVAData struct {
	PartnerServiceID      string          `json:"partnerServiceId,omitempty"`
	CustomerNo            string          `json:"customerNo,omitempty"`
	VirtualAccountNo      string          `json:"virtualAccountNo,omitempty"`
	VirtualAccountName    string          `json:"virtualAccountName,omitempty"`
	TrxID                 string          `json:"trxId,omitempty"`
	TotalAmount           *snap.Money     `json:"totalAmount,omitempty"`
	BillDetails           []BillDetail    `json:"billDetails,omitempty"`
	FreeTexts             []LocalizedText `json:"freeTexts,omitempty"`
	VirtualAccountTrxType string          `json:"virtualAccountTrxType,omitempty"`
	FeeAmount             *snap.Money     `json:"feeAmount,omitempty"`
	ExpiredDate           string          `json:"expiredDate,omitempty"`
	LastUpdateDate        string          `json:"lastUpdateDate,omitempty"`
	PaymentDate           string          `json:"paymentDate,omitempty"`
	AdditionalInfo        json.RawMessage `json:"additionalInfo,omitempty"`
}

// InquiryVAResponse is the response body for API VA - Inquiry VA.
type InquiryVAResponse struct {
	ResponseCode       string         `json:"responseCode"`
	ResponseMessage    string         `json:"responseMessage"`
	VirtualAccountData *InquiryVAData `json:"virtualAccountData,omitempty"`
}

// InquiryVA calls the SNAP VA - Inquiry VA endpoint (Service Code 30,
// path .../{version}/transfer-va/inquiry-va). hb must already carry
// every field snap.HeaderBuilder needs except Body, which InquiryVA sets
// itself so the exact marshaled bytes are used for both signing and the
// wire body.
func InquiryVA(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req InquiryVARequest) (InquiryVAResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return InquiryVAResponse{}, fmt.Errorf("snap: inquiry va: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return InquiryVAResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return InquiryVAResponse{}, fmt.Errorf("snap: inquiry va: %w", err)
	}

	var resp InquiryVAResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return InquiryVAResponse{}, fmt.Errorf("snap: inquiry va: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return InquiryVAResponse{}, errors.New("snap: inquiry va: response has no responseCode")
	}
	return resp, nil
}
