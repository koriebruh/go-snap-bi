package snap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
)

// custIDMerchantPattern is an allowlist, not a denylist, for
// CardRegistrationInquiry's path parameter: the Guides tab specs
// custIdMerchant as a String of length 18, so anything outside
// [A-Za-z0-9_-] (up to a generous 64 chars, in case some issuer's ID
// scheme differs from the one worked example) is rejected outright. An
// allowlist closes the whole class of path-segment hazards in one guard
// — not just "." and ".." literally, but any encoding trick (invalid
// UTF-8, Unicode dot lookalikes, a %2F-smuggled extra segment) a
// decode-then-normalize intermediary might unfold back into a real path
// boundary, none of which a three-value denylist can enumerate.
var custIDMerchantPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

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
// hb.EndpointURL must be an absolute URL with a host and no query string
// or fragment — CardRegistrationInquiry rejects one that isn't, rather
// than silently mishandling it (e.g. a fragment is never transmitted by
// net/http at all, so appending after one would silently drop
// custIDMerchant from the actual request with no error).
//
// custIDMerchant is validated against custIDMerchantPattern (see its doc
// comment), then joined onto hb.EndpointURL's path via url.URL.JoinPath
// — this is the package's first caller input that reaches a URL path
// rather than a JSON body, and the first place this package parses a URL
// rather than building one by concatenation. JoinPath (not manual
// string-building on u.Path, and NOT a separate url.PathEscape call,
// which would double-escape) both collapses any number of slashes
// between EndpointURL and "custIdMerchant" to exactly one, and preserves
// any percent-encoding already present in EndpointURL's path — an
// earlier version of this function cleared u.RawPath to force
// re-derivation from the decoded u.Path, which silently turned an
// encoded "%2F"/"%2E%2E" already in the caller's own EndpointURL into a
// real path boundary before the request was ever sent.
func CardRegistrationInquiry(ctx context.Context, t *Transport, hb HeaderBuilder, custIDMerchant string) (CardRegistrationInquiryResponse, error) {
	if !custIDMerchantPattern.MatchString(custIDMerchant) {
		return CardRegistrationInquiryResponse{}, fmt.Errorf("snap: card registration inquiry: invalid custIdMerchant %q", custIDMerchant)
	}

	u, perr := url.Parse(hb.EndpointURL)
	if perr != nil {
		// Deliberately don't wrap the raw *url.Error: its Error() string
		// reprints the input URL verbatim, which could carry userinfo
		// credentials if a caller ever put one in EndpointURL.
		if urlErr, ok := perr.(*url.Error); ok {
			return CardRegistrationInquiryResponse{}, fmt.Errorf("snap: card registration inquiry: invalid EndpointURL: %w", urlErr.Err)
		}
		return CardRegistrationInquiryResponse{}, fmt.Errorf("snap: card registration inquiry: invalid EndpointURL: %w", perr)
	}
	if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.Host == "" {
		return CardRegistrationInquiryResponse{}, errors.New("snap: card registration inquiry: EndpointURL must be an absolute URL with a host and no query string or fragment")
	}
	u = u.JoinPath("custIdMerchant", custIDMerchant)

	hb.Method = http.MethodGet
	hb.Body = nil
	hb.EndpointURL = u.String()

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
