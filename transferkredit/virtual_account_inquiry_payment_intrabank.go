package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// VAInquiryPaymentIntrabankRequest is the request body for API VA -
// Inquiry Payment Intrabank (Service Code 32). PartnerServiceID,
// CustomerNo, and VirtualAccountNo (the identity triple) are mandatory
// per the Guides tab. CustomerNo is json.RawMessage: this endpoint's
// own worked example shows it as a bare JSON number (research §5.3
// line 135) — see the Phase 15 design doc's ambiguous-type table. Only
// the fields §5.3's own field-summary row names for this endpoint are
// modeled; billDetails/freeTexts/totalAmount/feeAmount are not
// restated for 32 the way they are for endpoint 24, so they are not
// added here.
type VAInquiryPaymentIntrabankRequest struct {
	PartnerServiceID   string          `json:"partnerServiceId"`
	CustomerNo         json.RawMessage `json:"customerNo"`
	VirtualAccountNo   string          `json:"virtualAccountNo"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	SourceAccountNo    string          `json:"sourceAccountNo,omitempty"`
	SourceAccountType  string          `json:"sourceAccountType,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// VAInquiryPaymentIntrabankData is the "virtualAccountdata" object
// (lowercase d — confirmed by research §5.3 line 132 as not a table
// typo, unlike every VA type from Phases 13-14 which use capital-D
// virtualAccountData) in VAInquiryPaymentIntrabankResponse.
type VAInquiryPaymentIntrabankData struct {
	PartnerServiceID   string          `json:"partnerServiceId,omitempty"`
	CustomerNo         json.RawMessage `json:"customerNo,omitempty"`
	VirtualAccountNo   string          `json:"virtualAccountNo,omitempty"`
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	SourceAccountNo    string          `json:"sourceAccountNo,omitempty"`
	SourceAccountType  string          `json:"sourceAccountType,omitempty"`
	ProductName        string          `json:"productName,omitempty"`
	BillAmountLabel    string          `json:"billAmountLabel,omitempty"`
	BillAmountValue    string          `json:"billAmountValue,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// VAInquiryPaymentIntrabankResponse is the response body for API VA -
// Inquiry Payment Intrabank.
type VAInquiryPaymentIntrabankResponse struct {
	ResponseCode       string                         `json:"responseCode"`
	ResponseMessage    string                         `json:"responseMessage"`
	VirtualAccountData *VAInquiryPaymentIntrabankData `json:"virtualAccountdata,omitempty"`
}

// VAInquiryPaymentIntrabank calls the SNAP VA - Inquiry Payment
// Intrabank endpoint (Service Code 32, path
// .../{version}/transfer-va/inquiry-payment-intrabank, HTTP POST — no
// method override). hb must already carry every field snap.HeaderBuilder
// needs except Body, which VAInquiryPaymentIntrabank sets itself so
// the exact marshaled bytes are used for both signing and the wire
// body.
func VAInquiryPaymentIntrabank(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req VAInquiryPaymentIntrabankRequest) (VAInquiryPaymentIntrabankResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return VAInquiryPaymentIntrabankResponse{}, fmt.Errorf("snap: va inquiry payment intrabank: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return VAInquiryPaymentIntrabankResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return VAInquiryPaymentIntrabankResponse{}, fmt.Errorf("snap: va inquiry payment intrabank: %w", err)
	}

	var resp VAInquiryPaymentIntrabankResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return VAInquiryPaymentIntrabankResponse{}, fmt.Errorf("snap: va inquiry payment intrabank: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return VAInquiryPaymentIntrabankResponse{}, errors.New("snap: va inquiry payment intrabank: response has no responseCode")
	}
	return resp, nil
}
