package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// DeleteVARequest is the request body for API VA - Delete VA (Service
// Code 31). PartnerServiceID, CustomerNo, and VirtualAccountNo are
// mandatory per the Guides tab; TrxID is Optional.
type DeleteVARequest struct {
	PartnerServiceID string          `json:"partnerServiceId"`
	CustomerNo       string          `json:"customerNo"`
	VirtualAccountNo string          `json:"virtualAccountNo"`
	TrxID            string          `json:"trxId,omitempty"`
	AdditionalInfo   json.RawMessage `json:"additionalInfo,omitempty"`
}

// DeleteVAData is the "virtualAccountData" object in DeleteVAResponse —
// a smaller shape than the other VA management endpoints' data object,
// per the research doc.
type DeleteVAData struct {
	PartnerServiceID string          `json:"partnerServiceId,omitempty"`
	CustomerNo       string          `json:"customerNo,omitempty"`
	VirtualAccountNo string          `json:"virtualAccountNo,omitempty"`
	TrxID            string          `json:"trxId,omitempty"`
	AdditionalInfo   json.RawMessage `json:"additionalInfo,omitempty"`
}

// DeleteVAResponse is the response body for API VA - Delete VA.
type DeleteVAResponse struct {
	ResponseCode       string        `json:"responseCode"`
	ResponseMessage    string        `json:"responseMessage"`
	VirtualAccountData *DeleteVAData `json:"virtualAccountData,omitempty"`
}

// DeleteVA calls the SNAP VA - Delete VA endpoint (Service Code 31,
// path .../{version}/transfer-va/delete-va, HTTP DELETE). hb must
// already carry every field HeaderBuilder needs except Method and Body,
// which DeleteVA sets itself: Method to DELETE (the method is fixed by
// this endpoint, not caller-configurable) and Body, since this DELETE
// ships a JSON request body rather than using path parameters — the
// exact marshaled bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func DeleteVA(ctx context.Context, t *Transport, hb HeaderBuilder, req DeleteVARequest) (DeleteVAResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return DeleteVAResponse{}, fmt.Errorf("snap: delete va: encode request: %w", err)
	}
	hb.Method = http.MethodDelete
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return DeleteVAResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return DeleteVAResponse{}, fmt.Errorf("snap: delete va: %w", err)
	}

	var resp DeleteVAResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return DeleteVAResponse{}, fmt.Errorf("snap: delete va: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return DeleteVAResponse{}, errors.New("snap: delete va: response has no responseCode")
	}
	return resp, nil
}
