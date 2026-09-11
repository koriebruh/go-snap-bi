package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CreateVARequest is the request body for API VA - Create VA (Service
// Code 27). VirtualAccountName and TrxID are mandatory per the Guides
// tab; unlike every other Virtual Account endpoint, the identity triple
// (PartnerServiceID, CustomerNo, VirtualAccountNo) is Optional here.
type CreateVARequest struct {
	PartnerServiceID      string          `json:"partnerServiceId,omitempty"`
	CustomerNo            string          `json:"customerNo,omitempty"`
	VirtualAccountNo      string          `json:"virtualAccountNo,omitempty"`
	VirtualAccountName    string          `json:"virtualAccountName"`
	TrxID                 string          `json:"trxId"`
	TotalAmount           *Money          `json:"totalAmount,omitempty"`
	BillDetails           []BillDetail    `json:"billDetails,omitempty"`
	FreeTexts             []LocalizedText `json:"freeTexts,omitempty"`
	VirtualAccountTrxType string          `json:"virtualAccountTrxType,omitempty"`
	FeeAmount             *Money          `json:"feeAmount,omitempty"`
	ExpiredDate           string          `json:"expiredDate,omitempty"`
	AdditionalInfo        json.RawMessage `json:"additionalInfo,omitempty"`
}

// CreateVAData is the "virtualAccountData" object in CreateVAResponse.
type CreateVAData struct {
	PartnerServiceID      string          `json:"partnerServiceId,omitempty"`
	CustomerNo            string          `json:"customerNo,omitempty"`
	VirtualAccountNo      string          `json:"virtualAccountNo,omitempty"`
	VirtualAccountName    string          `json:"virtualAccountName,omitempty"`
	TrxID                 string          `json:"trxId,omitempty"`
	TotalAmount           *Money          `json:"totalAmount,omitempty"`
	BillDetails           []BillDetail    `json:"billDetails,omitempty"`
	FreeTexts             []LocalizedText `json:"freeTexts,omitempty"`
	VirtualAccountTrxType string          `json:"virtualAccountTrxType,omitempty"`
	FeeAmount             *Money          `json:"feeAmount,omitempty"`
	ExpiredDate           string          `json:"expiredDate,omitempty"`
	AdditionalInfo        json.RawMessage `json:"additionalInfo,omitempty"`
}

// CreateVAResponse is the response body for API VA - Create VA.
type CreateVAResponse struct {
	ResponseCode       string        `json:"responseCode"`
	ResponseMessage    string        `json:"responseMessage"`
	VirtualAccountData *CreateVAData `json:"virtualAccountData,omitempty"`
}

// CreateVA calls the SNAP VA - Create VA endpoint (Service Code 27,
// path .../{version}/transfer-va/create-va). hb must already carry
// every field HeaderBuilder needs except Body, which CreateVA sets
// itself so the exact marshaled bytes are used for both signing and the
// wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it —
// a fresh X-EXTERNAL-ID on retry risks creating a duplicate VA.
func CreateVA(ctx context.Context, t *Transport, hb HeaderBuilder, req CreateVARequest) (CreateVAResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CreateVAResponse{}, fmt.Errorf("snap: create va: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CreateVAResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CreateVAResponse{}, fmt.Errorf("snap: create va: %w", err)
	}

	var resp CreateVAResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CreateVAResponse{}, fmt.Errorf("snap: create va: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CreateVAResponse{}, errors.New("snap: create va: response has no responseCode")
	}
	return resp, nil
}
