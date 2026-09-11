package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// AccountBindingInquiryRequest is the request body for API Account Binding
// Inquiry (Service Code 08, path .../{version}/registration-account-inquiry
// — note the URL segment drops the word "binding", per the portal).
type AccountBindingInquiryRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountBindingInquiryResponse is the response body for API Account
// Binding Inquiry. Unlike AccountBindingResponse (Service Code 07), this
// response is flat — no accessTokenInfo/userInfo nesting.
//
// AccountTransactionLimit is typed string, not a numeric type: the Guides
// tab labels it "Numeric", but the portal's own worked example renders it
// as a quoted wire value ("accountTransactionLimit":"1000000") — the wire
// shape is confirmed, not guessed, so string matches what the server
// actually sends.
type AccountBindingInquiryResponse struct {
	ResponseCode            string          `json:"responseCode"`
	ResponseMessage         string          `json:"responseMessage"`
	ReferenceNo             string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo      string          `json:"partnerReferenceNo,omitempty"`
	AccountCurrency         string          `json:"accountCurrency,omitempty"`
	AccountName             string          `json:"accountName,omitempty"`
	AccountNo               string          `json:"accountNo,omitempty"`
	AccountTransactionLimit string          `json:"accountTransactionLimit,omitempty"`
	EndDatePeriod           string          `json:"endDatePeriod,omitempty"`
	StartDatePeriod         string          `json:"startDatePeriod,omitempty"`
	AdditionalInfo          json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountBindingInquiry calls the SNAP Account Binding Inquiry endpoint
// (Service Code 08). hb must already carry every field HeaderBuilder needs
// except Body, which AccountBindingInquiry sets itself so the exact
// marshaled bytes are used for both signing and the wire body.
func AccountBindingInquiry(ctx context.Context, t *Transport, hb HeaderBuilder, req AccountBindingInquiryRequest) (AccountBindingInquiryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AccountBindingInquiryResponse{}, fmt.Errorf("snap: account binding inquiry: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AccountBindingInquiryResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AccountBindingInquiryResponse{}, fmt.Errorf("snap: account binding inquiry: %w", err)
	}

	var resp AccountBindingInquiryResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AccountBindingInquiryResponse{}, fmt.Errorf("snap: account binding inquiry: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AccountBindingInquiryResponse{}, errors.New("snap: account binding inquiry: response has no responseCode")
	}
	return resp, nil
}
