package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CustomerTopUpInquiryStatusRequest is the request body for API
// Customer Top Up Inquiry Status (Service Code 39). Per research
// §5.5, this endpoint has "same originalX/serviceCode/status shape as
// Transaction Status Inquiry Bank" (Phase 16) — this type is
// field-identical to TransactionStatusInquiryBankRequest under its own
// name, per the package's "distinct types per service code even for
// identical shapes" convention. ServiceCode is the only mandatory
// field per the Guides tab.
type CustomerTopUpInquiryStatusRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	ServiceCode                string          `json:"serviceCode"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	Amount                     *Money          `json:"amount,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// CustomerTopUpInquiryStatusResponse is the response body for API
// Customer Top Up Inquiry Status — field-identical to
// TransactionStatusInquiryBankResponse under its own name (see the
// request type's comment); customer_topup_inquiry_status_test.go
// guards the two against drift with a full-equality reflection test.
type CustomerTopUpInquiryStatusResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	ServiceCode                string          `json:"serviceCode,omitempty"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	Amount                     *Money          `json:"amount,omitempty"`
	BeneficiaryAccountNo       string          `json:"beneficiaryAccountNo"`
	BeneficiaryBankCode        string          `json:"beneficiaryBankCode,omitempty"`
	PreviousResponseCode       string          `json:"previousResponseCode,omitempty"`
	ReferenceNumber            string          `json:"referenceNumber"`
	SourceAccountNo            string          `json:"sourceAccountNo"`
	TransactionID              string          `json:"transactionId,omitempty"`
	LatestTransactionStatus    string          `json:"latestTransactionStatus"`
	TransactionStatusDesc      string          `json:"transactionStatusDesc,omitempty"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// CustomerTopUpInquiryStatus calls the SNAP Customer Top Up Inquiry
// Status endpoint (Service Code 39, path
// .../{version}/customer-top-up-inquiry-status, HTTP POST — no method
// override). hb must already carry every field HeaderBuilder needs
// except Body, which CustomerTopUpInquiryStatus sets itself so the
// exact marshaled bytes are used for both signing and the wire body.
func CustomerTopUpInquiryStatus(ctx context.Context, t *Transport, hb HeaderBuilder, req CustomerTopUpInquiryStatusRequest) (CustomerTopUpInquiryStatusResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CustomerTopUpInquiryStatusResponse{}, fmt.Errorf("snap: customer top up inquiry status: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CustomerTopUpInquiryStatusResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CustomerTopUpInquiryStatusResponse{}, fmt.Errorf("snap: customer top up inquiry status: %w", err)
	}

	var resp CustomerTopUpInquiryStatusResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CustomerTopUpInquiryStatusResponse{}, fmt.Errorf("snap: customer top up inquiry status: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CustomerTopUpInquiryStatusResponse{}, errors.New("snap: customer top up inquiry status: response has no responseCode")
	}
	return resp, nil
}
