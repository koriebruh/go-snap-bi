package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	snap "github.com/koriebruh/go-snap-bi"
)

// UpdateStatusVARequest is the request body for API VA - Update Status
// VA (Service Code 29). PartnerServiceID, CustomerNo, VirtualAccountNo,
// TrxID, and PaidStatus are mandatory per the Guides tab. PaidStatus
// values are "Y"/"N" per the research doc.
type UpdateStatusVARequest struct {
	PartnerServiceID string          `json:"partnerServiceId"`
	CustomerNo       string          `json:"customerNo"`
	VirtualAccountNo string          `json:"virtualAccountNo"`
	TrxID            string          `json:"trxId"`
	PaidStatus       string          `json:"paidStatus"`
	AdditionalInfo   json.RawMessage `json:"additionalInfo,omitempty"`
}

// UpdateStatusVAData is the "virtualAccountData" object in
// UpdateStatusVAResponse — per the research doc, "the full VA data
// object", the same field set as UpdateVAData, under its own type name.
type UpdateStatusVAData struct {
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

// UpdateStatusVAResponse is the response body for API VA - Update
// Status VA.
type UpdateStatusVAResponse struct {
	ResponseCode       string              `json:"responseCode"`
	ResponseMessage    string              `json:"responseMessage"`
	VirtualAccountData *UpdateStatusVAData `json:"virtualAccountData,omitempty"`
}

// UpdateStatusVA calls the SNAP VA - Update Status VA endpoint (Service
// Code 29, path .../{version}/transfer-va/update-status, HTTP PUT). hb
// must already carry every field snap.HeaderBuilder needs except Method and
// Body, which UpdateStatusVA sets itself: Method to PUT (the method is
// fixed by this endpoint, not caller-configurable) and Body so the
// exact marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func UpdateStatusVA(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req UpdateStatusVARequest) (UpdateStatusVAResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return UpdateStatusVAResponse{}, fmt.Errorf("snap: update status va: encode request: %w", err)
	}
	hb.Method = http.MethodPut
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return UpdateStatusVAResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return UpdateStatusVAResponse{}, fmt.Errorf("snap: update status va: %w", err)
	}

	var resp UpdateStatusVAResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return UpdateStatusVAResponse{}, fmt.Errorf("snap: update status va: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return UpdateStatusVAResponse{}, errors.New("snap: update status va: response has no responseCode")
	}
	return resp, nil
}
