package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// CardRegistrationInquiryAccountData is the "accountData" object nested
// inside each API Card Registration Inquiry accountList entry. MaxLimit
// and CredentialNo are typed string: both are masked/formatted display
// values on the wire (e.g. "800000", "************0750"), not the raw
// numeric limit CardRegistrationSetLimitRequest.Limit sets — unambiguous
// per the Guides tab, unlike Phase 7's limit/cardData.
type CardRegistrationInquiryAccountData struct {
	AccountID      string `json:"accountId,omitempty"`
	CreatedDate    string `json:"createdDate,omitempty"`
	CredentialNo   string `json:"credentialNo,omitempty"`
	CredentialType string `json:"credentialType,omitempty"`
	MaxLimit       string `json:"maxLimit,omitempty"`
	Status         string `json:"status,omitempty"`
}

// CardRegistrationInquiryAccount is one entry in
// CardRegistrationInquiryResponse's AccountList. It wraps AccountData
// rather than flattening it, mirroring the wire shape's extra nesting
// level exactly ({"accountList":[{"accountData":{...}}]}).
type CardRegistrationInquiryAccount struct {
	AccountData CardRegistrationInquiryAccountData `json:"accountData"`
}

// CardRegistrationInquiryResponse is the response body for API Card
// Registration Inquiry (Service Code 03).
type CardRegistrationInquiryResponse struct {
	ResponseCode    string                           `json:"responseCode"`
	ResponseMessage string                           `json:"responseMessage"`
	AccountList     []CardRegistrationInquiryAccount `json:"accountList,omitempty"`
	AdditionalInfo  json.RawMessage                  `json:"additionalInfo,omitempty"`
}

// CardRegistrationInquiry calls the SNAP Card Registration Inquiry
// endpoint (Service Code 03, path
// .../{version}/registration-card-inquiry/custIdMerchant/{value} — a URL
// path parameter, not a JSON body field). hb must already carry every
// field HeaderBuilder needs except Method and Body, which
// CardRegistrationInquiry sets itself: Method to GET and Body to nil,
// since this is the package's only read-only GET endpoint and a caller
// should not need to configure protocol details specific to it.
// custIDMerchant is percent-encoded via url.PathEscape before being
// appended to hb.EndpointURL — this is the package's first caller input
// that reaches a URL path rather than a JSON body, so unlike every other
// binding's fields, an unescaped value here could alter the request path
// rather than simply being rejected as an invalid identifier.
func CardRegistrationInquiry(ctx context.Context, t *Transport, hb HeaderBuilder, custIDMerchant string) (CardRegistrationInquiryResponse, error) {
	hb.Method = http.MethodGet
	hb.Body = nil
	hb.EndpointURL = hb.EndpointURL + "/custIdMerchant/" + url.PathEscape(custIDMerchant)

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CardRegistrationInquiryResponse{}, err
	}
	if err := checkResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return CardRegistrationInquiryResponse{}, fmt.Errorf("snap: card registration inquiry: %w", err)
	}

	var resp CardRegistrationInquiryResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return CardRegistrationInquiryResponse{}, fmt.Errorf("snap: card registration inquiry: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return CardRegistrationInquiryResponse{}, errors.New("snap: card registration inquiry: response has no responseCode")
	}
	return resp, nil
}
