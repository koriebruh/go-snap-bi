package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// AccountInquiryExternalRequest is the request body for API External
// Account Inquiry (Service Code 16). BeneficiaryAccountNo and
// BeneficiaryBankCode are mandatory per the Guides tab.
type AccountInquiryExternalRequest struct {
	PartnerReferenceNo   string          `json:"partnerReferenceNo,omitempty"`
	BeneficiaryAccountNo string          `json:"beneficiaryAccountNo"`
	BeneficiaryBankCode  string          `json:"beneficiaryBankCode"`
	AdditionalInfo       json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountInquiryExternalResponse is the response body for API External
// Account Inquiry.
type AccountInquiryExternalResponse struct {
	ResponseCode           string          `json:"responseCode"`
	ResponseMessage        string          `json:"responseMessage"`
	ReferenceNo            string          `json:"referenceNo,omitempty"`
	PartnerReferenceNo     string          `json:"partnerReferenceNo,omitempty"`
	BeneficiaryAccountName string          `json:"beneficiaryAccountName"`
	BeneficiaryAccountNo   string          `json:"beneficiaryAccountNo"`
	BeneficiaryBankName    string          `json:"beneficiaryBankName,omitempty"`
	Currency               string          `json:"currency,omitempty"`
	AdditionalInfo         json.RawMessage `json:"additionalInfo,omitempty"`
}

// AccountInquiryExternal calls the SNAP External Account Inquiry endpoint
// (Service Code 16, path .../{version}/account-inquiry-external). hb must
// already carry every field snap.HeaderBuilder needs except Body, which
// AccountInquiryExternal sets itself so the exact marshaled bytes are used
// for both signing and the wire body.
func AccountInquiryExternal(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req AccountInquiryExternalRequest) (AccountInquiryExternalResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AccountInquiryExternalResponse{}, fmt.Errorf("snap: external account inquiry: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return AccountInquiryExternalResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return AccountInquiryExternalResponse{}, fmt.Errorf("snap: external account inquiry: %w", err)
	}

	var resp AccountInquiryExternalResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return AccountInquiryExternalResponse{}, fmt.Errorf("snap: external account inquiry: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return AccountInquiryExternalResponse{}, errors.New("snap: external account inquiry: response has no responseCode")
	}
	return resp, nil
}
