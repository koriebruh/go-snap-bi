package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	snap "github.com/koriebruh/go-snap-bi"
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

// errInvalidEndpointURL is CardRegistrationInquiry's sentinel for a
// hb.EndpointURL that isn't a bare http(s) URL with a host — a query
// string, fragment, userinfo, non-http(s) scheme, or missing host.
// Exposed via errors.Is so a test (or a caller) can distinguish this
// rejection from a downstream transport failure that happens to also
// return a non-nil error for the same malformed input.
var errInvalidEndpointURL = errors.New("snap: card registration inquiry: EndpointURL must be a bare http(s) URL with a host, no userinfo, and no query string or fragment")

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
// field snap.HeaderBuilder needs except Method and Body, which
// CardRegistrationInquiry sets itself: Method to GET and Body to nil,
// since this is the package's only read-only GET endpoint and a caller
// should not need to configure protocol details specific to it.
// hb.EndpointURL must be an http(s) URL with a host, no userinfo, and no
// query string or fragment — CardRegistrationInquiry rejects one that
// isn't, rather than silently mishandling it (e.g. a fragment is never
// transmitted by net/http at all, so appending after one would silently
// drop custIDMerchant from the actual request with no error).
//
// custIDMerchant is validated against custIDMerchantPattern (see its doc
// comment), then joined onto hb.EndpointURL's path via url.URL.JoinPath
// — this is the package's first caller input that reaches a URL path
// rather than a JSON body, and the first place this package parses a URL
// rather than building one by concatenation. JoinPath does NOT "leave
// RawPath alone": it rebuilds the joined path from u.EscapedPath() and
// then re-derives both Path and RawPath from that already-escaped
// string (net/url's setPath). Encoding already present in EndpointURL
// survives because it's carried through EscapedPath(), not because
// RawPath is untouched — RawPath is in fact always overwritten. Getting
// this distinction wrong matters here specifically: an earlier version
// of this function set u.Path directly (the decoded form) and cleared
// u.RawPath to force re-derivation FROM THE DECODED PATH, which silently
// turned an encoded "%2F"/"%2E%2E" already in the caller's own
// EndpointURL into a real path boundary before the request was ever
// sent. JoinPath avoids that because it starts from EscapedPath(), not
// Path.
func CardRegistrationInquiry(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, custIDMerchant string) (CardRegistrationInquiryResponse, error) {
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
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return CardRegistrationInquiryResponse{}, errInvalidEndpointURL
	}
	u = u.JoinPath("custIdMerchant", custIDMerchant)

	hb.Method = http.MethodGet
	hb.Body = nil
	hb.EndpointURL = u.String()

	env, err := t.Do(ctx, hb)
	if err != nil {
		return CardRegistrationInquiryResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
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
