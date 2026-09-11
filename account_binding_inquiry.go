package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// AccountBindingInquiryRequest is the request body for API Account Binding
// Inquiry (Service Code 08, path .../{version}/registration-account-inquiry
// — note the URL segment drops the word "binding", per the portal). Neither
// the Guides tab's field table nor this type defines an account identifier
// field, and the portal's worked example carries only a plain B2B
// Authorization header — how a given PJP identifies which account to
// report is not specified here; a caller whose PJP needs a consumer
// identifier for this call places it in AdditionalInfo.
type AccountBindingInquiryRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountBindingInquiryResponse is the response body for API Account
// Binding Inquiry. Unlike AccountBindingResponse (Service Code 07), this
// response is flat — no accessTokenInfo/userInfo nesting.
//
// AccountTransactionLimit is typed json.RawMessage, not string or a numeric
// type, for the same reason as AccountCreationResponse.APIKey: the Guides
// tab labels it "Numeric", and while this portal's one worked example
// renders it as a quoted wire value ("accountTransactionLimit":"1000000"),
// SNAP is a multi-PJP standard and one example from one PJP is thin
// evidence about what every issuer sends. A plain string field would fail
// the ENTIRE decode (discarding ResponseCode, AccountNo, AccountName, etc.
// too) if some other issuer sends it unquoted, for a field this package
// has no way to retry around. json.RawMessage accepts either wire shape
// without loss; a caller strips surrounding quotes themselves if the value
// is quoted. A JSON null decodes to a non-nil RawMessage("null"), distinct
// from an absent key (nil) — a third shape a caller checking for presence
// should account for.
type AccountBindingInquiryResponse struct {
	ResponseCode            string          `json:"responseCode"`
	ResponseMessage         string          `json:"responseMessage"`
	ReferenceNo             string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo      string          `json:"partnerReferenceNo,omitempty"`
	AccountCurrency         string          `json:"accountCurrency,omitempty"`
	AccountName             string          `json:"accountName,omitempty"`
	AccountNo               string          `json:"accountNo,omitempty"`
	AccountTransactionLimit json.RawMessage `json:"accountTransactionLimit,omitempty"`
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
