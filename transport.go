package snap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	HTTPClient *http.Client // optional; nil means a client with defaultHTTPTimeout
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
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
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
			return Envelope{}, fmt.Errorf("snap: transport: %w: http status %d: decode response body: %v", sentinelForHTTPStatus(resp.StatusCode), resp.StatusCode, err)
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
