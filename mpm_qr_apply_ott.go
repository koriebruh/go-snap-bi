package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ApplyOTTRequest is the request body for API Payment Redirect - Apply
// OTT (Service Code 49).
//
// Contradiction, unresolved: the source research documents a
// userResources field (Array of String(64), Mandatory), but its own
// worked example shows the request body as the bare JSON array
// ["OTT"], not an object wrapping a userResources key. This is the
// only endpoint in the package whose worked example is not an object
// body. Modeled here as a defined slice type, marshaled directly as a
// top-level JSON array — the literal reading of the worked example.
// The alternative reading ({"userResources":[...]}) is recorded here
// but not implemented; needs sandbox/Postman verification.
//
// A nil ApplyOTTRequest marshals to the JSON literal null, not []; the
// package does no client-side validation anywhere (matching its
// general practice), so callers must pass a non-nil, non-empty slice
// themselves.
type ApplyOTTRequest []string

// ApplyOTTUserResource is one entry in an ApplyOTTResponse's
// userResources array. Despite sharing a field name with the request,
// this is a distinct shape (an object, not a bare string).
type ApplyOTTUserResource struct {
	ResourceType string `json:"resourceType"`
	Value        string `json:"value"`
}

// ApplyOTTResponse is the response body for API Payment Redirect -
// Apply OTT.
type ApplyOTTResponse struct {
	ResponseCode    string                 `json:"responseCode"`
	ResponseMessage string                 `json:"responseMessage"`
	UserResources   []ApplyOTTUserResource `json:"userResources"`
}

// ApplyOTT calls the SNAP Payment Redirect - Apply OTT endpoint
// (Service Code 49, path .../{version}/qr/apply-ott, HTTP POST — no
// method override). hb must already carry every field HeaderBuilder
// needs except Body, which ApplyOTT sets itself so the exact marshaled
// bytes are used for both signing and the wire body.
//
// This operation is not idempotent and this package does not retry.
// Callers that retry a failed or timed-out call should reuse the same
// X-EXTERNAL-ID, since the server's own duplicate-detection keys on it.
func ApplyOTT(ctx context.Context, t *Transport, hb HeaderBuilder, req ApplyOTTRequest) (ApplyOTTResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return ApplyOTTResponse{}, fmt.Errorf("snap: apply ott: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return ApplyOTTResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return ApplyOTTResponse{}, fmt.Errorf("snap: apply ott: %w", err)
	}

	var resp ApplyOTTResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return ApplyOTTResponse{}, fmt.Errorf("snap: apply ott: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return ApplyOTTResponse{}, errors.New("snap: apply ott: response has no responseCode")
	}
	return resp, nil
}
