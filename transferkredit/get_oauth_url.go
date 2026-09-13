package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	snap "github.com/koriebruh/go-snap-bi"
)

// errGetOAuthURLInvalidEndpointURL is GetOAuthURL's sentinel for a
// hb.EndpointURL that isn't a bare http(s) URL with a host — a query
// string, fragment, userinfo, non-http(s) scheme, or missing host —
// mirroring CardRegistrationInquiry's same defensive check, since this
// is this package's second endpoint to build a URL rather than a JSON
// body.
var errGetOAuthURLInvalidEndpointURL = errors.New("snap: get oauth url: EndpointURL must be a bare http(s) URL with a host, no userinfo, and no query string or fragment")

// GetOAuthURLRequest holds API Get OAuth URL's parameters (Service
// Code 10, Registrasi group). Unlike every other calling function in
// this package, these are never JSON-marshaled — the Guides tab
// documents them as a query string, not a request body — so this type
// carries no json tags; GetOAuthURL builds a url.Values query string
// from it directly. RedirectURL, Scopes, and State are Mandatory;
// every other field is Optional except SeamlessSign, which is
// Conditional ("must be filled if seamlessData is present") — this
// package validates wire shape only, never business rules, so
// SeamlessSign is simply omitted from the query when empty, the same
// as every other Optional field here.
type GetOAuthURLRequest struct {
	RedirectURL string   // M, String(256)
	Scopes      []string // M, List(256) -- joined with "," per the worked example
	State       string   // M, String(32)

	MerchantID        string // O, String(64)
	SubMerchantID     string // O, String(32)
	Lang              string // O, String(2), ISO 639-1
	AllowRegistration *bool  // O, Boolean -- pointer so "absent" and "explicitly false" stay distinguishable
	SeamlessData      string // O, String(512)
	MobileNumber      string // O, String(18)
	VerifiedTime      string // O, String(25)
	ExternalUID       string // O, String(32)
	DeviceID          string // O, String(32)
	SeamlessSign      string // C, String(512) -- must be filled if SeamlessData is present, per the Guides tab
}

// GetOAuthURLResponse is the response body for API Get OAuth URL.
// Every field is Mandatory.
type GetOAuthURLResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	AuthCode        string `json:"authCode"`
	State           string `json:"state"`
}

// GetOAuthURL calls the SNAP Get OAuth URL endpoint (Service Code 10,
// path .../{version}/get-auth-code). hb must already carry every
// field snap.HeaderBuilder needs except Method and Body, which
// GetOAuthURL sets itself: Method to GET and Body to nil, since this
// is a read-only, query-string-only call. hb.EndpointURL must be an
// http(s) URL with a host, no userinfo, and no query string or
// fragment (same defensive check as CardRegistrationInquiry, this
// package's other URL-building endpoint) — GetOAuthURL rejects one
// that isn't, rather than silently overwriting or merging into a
// caller-supplied query string.
func GetOAuthURL(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req GetOAuthURLRequest) (GetOAuthURLResponse, error) {
	u, perr := url.Parse(hb.EndpointURL)
	if perr != nil {
		if urlErr, ok := perr.(*url.Error); ok {
			return GetOAuthURLResponse{}, fmt.Errorf("snap: get oauth url: invalid EndpointURL: %w", urlErr.Err)
		}
		return GetOAuthURLResponse{}, fmt.Errorf("snap: get oauth url: invalid EndpointURL: %w", perr)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return GetOAuthURLResponse{}, errGetOAuthURLInvalidEndpointURL
	}

	q := url.Values{}
	q.Set("redirectUrl", req.RedirectURL)
	q.Set("scopes", strings.Join(req.Scopes, ","))
	q.Set("state", req.State)
	if req.MerchantID != "" {
		q.Set("merchantId", req.MerchantID)
	}
	if req.SubMerchantID != "" {
		q.Set("subMerchantId", req.SubMerchantID)
	}
	if req.Lang != "" {
		q.Set("lang", req.Lang)
	}
	if req.AllowRegistration != nil {
		q.Set("allowRegistration", strconv.FormatBool(*req.AllowRegistration))
	}
	if req.SeamlessData != "" {
		q.Set("seamlessData", req.SeamlessData)
	}
	if req.MobileNumber != "" {
		q.Set("mobileNumber", req.MobileNumber)
	}
	if req.VerifiedTime != "" {
		q.Set("verifiedTime", req.VerifiedTime)
	}
	if req.ExternalUID != "" {
		q.Set("externalUid", req.ExternalUID)
	}
	if req.DeviceID != "" {
		q.Set("deviceId", req.DeviceID)
	}
	if req.SeamlessSign != "" {
		q.Set("seamlessSign", req.SeamlessSign)
	}
	u.RawQuery = q.Encode()

	hb.Method = http.MethodGet
	hb.Body = nil
	hb.EndpointURL = u.String()

	env, err := t.Do(ctx, hb)
	if err != nil {
		return GetOAuthURLResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return GetOAuthURLResponse{}, fmt.Errorf("snap: get oauth url: %w", err)
	}

	var resp GetOAuthURLResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return GetOAuthURLResponse{}, fmt.Errorf("snap: get oauth url: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return GetOAuthURLResponse{}, errors.New("snap: get oauth url: response has no responseCode")
	}
	return resp, nil
}
