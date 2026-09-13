package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// TransferToOTCTransferStatusRequest is the request body for API
// Transfer To OTC - Transfer Status (Service Code 45). Per research
// §5.8, this endpoint uses the same originalX/serviceCode/status
// pattern as TransactionStatusInquiryBank (Phase 16) but "adds
// customerNumber M, amount M on request" — CustomerNumber and Amount
// are additions beyond that base pattern, and Amount is Mandatory
// here (a plain snap.Money value, not *snap.Money, per the established rule),
// unlike the base pattern's Optional *snap.Money. AdditionalInfo is kept
// from the base pattern (research §5.8 names only the two additions
// above, not a subtraction, and §5.4's base request documents
// additionalInfo — go-review round 1 caught this field's initial,
// undocumented omission). ServiceCode and CustomerNumber are mandatory
// per the Guides tab.
type TransferToOTCTransferStatusRequest struct {
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	ServiceCode                string          `json:"serviceCode"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	CustomerNumber             string          `json:"customerNumber"`
	Amount                     snap.Money      `json:"amount"`
	AdditionalInfo             json.RawMessage `json:"additionalInfo,omitempty"`
}

// TransferToOTCTransferStatusResponse is the response body for API
// Transfer To OTC - Transfer Status — field-identical to
// TransactionStatusInquiryBankResponse under its own name, per the
// package's "distinct types per service code even for identical
// shapes" convention: research scopes this endpoint's additions to
// "on request" only, so the response mirrors the base pattern exactly.
// transfer_to_otc_status_test.go guards the two against drift with a
// full-equality reflection test.
type TransferToOTCTransferStatusResponse struct {
	ResponseCode               string          `json:"responseCode"`
	ResponseMessage            string          `json:"responseMessage"`
	OriginalPartnerReferenceNo string          `json:"originalPartnerReferenceNo,omitempty"`
	OriginalReferenceNo        string          `json:"originalReferenceNo,omitempty"`
	OriginalExternalID         string          `json:"originalExternalId,omitempty"`
	ServiceCode                string          `json:"serviceCode,omitempty"`
	TransactionDate            string          `json:"transactionDate,omitempty"`
	Amount                     *snap.Money     `json:"amount,omitempty"`
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

// TransferToOTCTransferStatus calls the SNAP Transfer To OTC -
// Transfer Status endpoint (Service Code 45, path
// .../{version}/emoney/otc-status, HTTP POST — no method override).
// hb must already carry every field snap.HeaderBuilder needs except Body,
// which TransferToOTCTransferStatus sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
func TransferToOTCTransferStatus(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req TransferToOTCTransferStatusRequest) (TransferToOTCTransferStatusResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TransferToOTCTransferStatusResponse{}, fmt.Errorf("snap: transfer to otc transfer status: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return TransferToOTCTransferStatusResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return TransferToOTCTransferStatusResponse{}, fmt.Errorf("snap: transfer to otc transfer status: %w", err)
	}

	var resp TransferToOTCTransferStatusResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return TransferToOTCTransferStatusResponse{}, fmt.Errorf("snap: transfer to otc transfer status: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return TransferToOTCTransferStatusResponse{}, errors.New("snap: transfer to otc transfer status: response has no responseCode")
	}
	return resp, nil
}
