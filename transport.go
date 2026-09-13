package snap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// defaultHTTPTimeout is used when a caller does not supply an *http.Client.
const defaultHTTPTimeout = 30 * time.Second

// maxResponseBytes caps how much of a response body is read, so a
// misbehaving or malicious server can't force unbounded memory use.
const maxResponseBytes = 10 << 20 // 10 MiB

// Envelope is the generic decoded response body for a SNAP request, plus
// the transport-level HTTP status. Per-service bindings (in this same
// package) unmarshal Raw into their own typed response.
//
// StatusCode matters because ResponseCode is only reliable when the server
// actually returned SNAP's own error shape — a body from a proxy, WAF, or
// gateway in front of it may carry no ResponseCode at all. A per-service
// binding with no ResponseCode to fall back on should use
// sentinelForHTTPStatus(env.StatusCode) in that case, the same pattern
// TokenManager already uses internally.
type Envelope struct {
	StatusCode      int
	ResponseCode    string
	ResponseMessage string
	Raw             json.RawMessage
}

// Transport sends a signed request built by a HeaderBuilder and decodes the
// response into an Envelope. It does not retry, rate-limit, or interpret a
// non-2xx responseCode as a Go error — callers do that via ResponseCodeError.
type Transport struct {
	// HTTPClient is optional; nil means a client with defaultHTTPTimeout and
	// a CheckRedirect policy that never follows a redirect (see Do). A
	// caller-supplied HTTPClient bypasses that policy entirely — no SNAP
	// endpoint has a legitimate reason to redirect, and a 307/308 preserves
	// method and body, so a caller-supplied client MUST set an equivalent
	// CheckRedirect (returning http.ErrUseLastResponse, or otherwise
	// refusing every redirect) or risk leaking the signed X-SIGNATURE,
	// X-CLIENT-KEY, X-PARTNER-ID headers and the full request body —
	// account numbers, amounts, customer PII — to whatever host a
	// compromised or misconfigured server redirects to.
	HTTPClient *http.Client
}

// Do signs hb, sends it, and decodes the response body into an Envelope.
// Only transport-level failures (header build error, network error, non-JSON
// body, context cancellation) are returned as error.
func (t *Transport) Do(ctx context.Context, hb HeaderBuilder) (Envelope, error) {
	headers, err := hb.Build()
	if err != nil {
		return Envelope{}, err
	}

	req, err := http.NewRequestWithContext(ctx, hb.Method, hb.EndpointURL, bytes.NewReader(hb.Body))
	if err != nil {
		return Envelope{}, fmt.Errorf("snap: transport: build request: %w", err)
	}
	for k, v := range headers {
		req.Header[k] = v
	}

	client := t.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout, CheckRedirect: neverFollowRedirect}
	}
	resp, err := client.Do(req)
	if err != nil {
		// Unwrap *url.Error before wrapping: its Error() string reprints
		// the request URL verbatim, including any query string — for
		// GetOAuthURL and CardRegistrationInquiry that can carry caller
		// PII (mobile numbers, seamless-sign values) or a signature, and
		// this error is exactly the kind that ends up in a log on a
		// transient network failure. See CardRegistrationInquiry's own
		// doc comment for the same reasoning applied to url.Parse errors.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return Envelope{}, fmt.Errorf("snap: transport: %w", urlErr.Err)
		}
		return Envelope{}, fmt.Errorf("snap: transport: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return Envelope{}, fmt.Errorf("snap: transport: read response body: %w", err)
	}

	var parsed struct {
		ResponseCode    string `json:"responseCode"`
		ResponseMessage string `json:"responseMessage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// A non-JSON body (HTML WAF page, empty body from a load
			// balancer, etc.) on a non-2xx response must still be
			// errors.Is-matchable via the HTTP status, not just an opaque
			// decode error — this is the same proxy/WAF/gateway case
			// Envelope.StatusCode exists for, just reached from the parse
			// failure path instead of a successfully-parsed empty
			// responseCode.
			return Envelope{}, fmt.Errorf("snap: transport: %w: http status %d: decode response body: %w", sentinelForHTTPStatus(resp.StatusCode), resp.StatusCode, err)
		}
		return Envelope{}, fmt.Errorf("snap: transport: decode response body: %w", err)
	}

	return Envelope{
		StatusCode:      resp.StatusCode,
		ResponseCode:    parsed.ResponseCode,
		ResponseMessage: parsed.ResponseMessage,
		Raw:             json.RawMessage(body),
	}, nil
}

// neverFollowRedirect is the CheckRedirect policy for every default
// *http.Client this package constructs. No SNAP endpoint has a
// legitimate reason to redirect, and net/http's default policy (follow
// up to 10 redirects) copies every non-Authorization header — including
// this package's signed X-SIGNATURE, X-CLIENT-KEY, and X-PARTNER-ID —
// and, for a 307/308, the full request body, to whatever host the
// server names. Returning http.ErrUseLastResponse makes the client
// return the first response as-is rather than following it, so a
// caller sees the redirect status code instead of it being silently
// chased.
func neverFollowRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
