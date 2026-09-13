package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	snap "github.com/koriebruh/go-snap-bi"
)

// UpdateVARequest is the request body for API VA - Update VA (Service
// Code 28). PartnerServiceID, CustomerNo, VirtualAccountNo,
// VirtualAccountName, and TrxID are mandatory per the Guides tab —
// unlike Create VA, the identity triple is Mandatory here.
type UpdateVARequest struct {
	PartnerServiceID      string          `json:"partnerServiceId"`
	CustomerNo            string          `json:"customerNo"`
	VirtualAccountNo      string          `json:"virtualAccountNo"`
	VirtualAccountName    string          `json:"virtualAccountName"`
	TrxID                 string          `json:"trxId"`
	TotalAmount           *snap.Money     `json:"totalAmount,omitempty"`
	BillDetails           []BillDetail    `json:"billDetails,omitempty"`
	FreeTexts             []LocalizedText `json:"freeTexts,omitempty"`
	VirtualAccountTrxType string          `json:"virtualAccountTrxType,omitempty"`
	FeeAmount             *snap.Money     `json:"feeAmount,omitempty"`
	ExpiredDate           string          `json:"expiredDate,omitempty"`
	AdditionalInfo        json.RawMessage `json:"additionalInfo,omitempty"`
}

// UpdateVAData is the "virtualAccountData" object in UpdateVAResponse —
// Create VA's data fields plus LastUpdateDate and PaymentDate.
type UpdateVAData struct {
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

// UpdateVAResponse is the response body for API VA - Update VA.
type UpdateVAResponse struct {
	ResponseCode       string        `json:"responseCode"`
	ResponseMessage    string        `json:"responseMessage"`
	VirtualAccountData *UpdateVAData `json:"virtualAccountData,omitempty"`
}

// UpdateVA calls the SNAP VA - Update VA endpoint (Service Code 28,
// path .../{version}/transfer-va/update-va, HTTP PUT). hb must already
// carry every field snap.HeaderBuilder needs except Method and Body, which
// UpdateVA sets itself: Method to PUT (the method is fixed by this
// endpoint, not caller-configurable) and Body so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func UpdateVA(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req UpdateVARequest) (UpdateVAResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return UpdateVAResponse{}, fmt.Errorf("snap: update va: encode request: %w", err)
	}
	hb.Method = http.MethodPut
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return UpdateVAResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return UpdateVAResponse{}, fmt.Errorf("snap: update va: %w", err)
	}

	var resp UpdateVAResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return UpdateVAResponse{}, fmt.Errorf("snap: update va: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return UpdateVAResponse{}, errors.New("snap: update va: response has no responseCode")
	}
	return resp, nil
}
