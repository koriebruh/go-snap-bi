package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// AccountInquiryInternalRequest is the request body for API Internal
// Account Inquiry (Service Code 15). BeneficiaryAccountNo is the only
// mandatory field per the Guides tab.
type AccountInquiryInternalRequest struct {
	PartnerReferenceNo   string          `json:"partnerReferenceNo,omitempty"`
	BeneficiaryAccountNo string          `json:"beneficiaryAccountNo"`
	AdditionalInfo       json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountInquiryInternalResponse is the response body for API Internal
// Account Inquiry.
type AccountInquiryInternalResponse struct {
	ResponseCode             string          `json:"responseCode"`
	ResponseMessage          string          `json:"responseMessage"`
	ReferenceNo              string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo       string          `json:"partnerReferenceNo,omitempty"`
	BeneficiaryAccountName   string          `json:"beneficiaryAccountName"`
	BeneficiaryAccountNo     string          `json:"beneficiaryAccountNo"`
	BeneficiaryAccountStatus string          `json:"beneficiaryAccountStatus,omitempty"`
	BeneficiaryAccountType   string          `json:"beneficiaryAccountType,omitempty"`
	Currency                 string          `json:"currency,omitempty"`
	AdditionalInfo           json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountInquiryInternal calls the SNAP Internal Account Inquiry endpoint
// (Service Code 15, path .../{version}/account-inquiry-internal). hb must
// already carry every field HeaderBuilder needs except Body, which
// AccountInquiryInternal sets itself so the exact marshaled bytes are used
// for both signing and the wire body.
func AccountInquiryInternal(ctx context.Context, t *Transport, hb HeaderBuilder, req AccountInquiryInternalRequest) (AccountInquiryInternalResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AccountInquiryInternalResponse{}, fmt.Errorf("snap: internal account inquiry: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AccountInquiryInternalResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AccountInquiryInternalResponse{}, fmt.Errorf("snap: internal account inquiry: %w", err)
	}

	var resp AccountInquiryInternalResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AccountInquiryInternalResponse{}, fmt.Errorf("snap: internal account inquiry: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AccountInquiryInternalResponse{}, errors.New("snap: internal account inquiry: response has no responseCode")
	}
	return resp, nil
}
