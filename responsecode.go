package snap

import (
	"errors"
	"fmt"
)

// Sentinel errors for SNAP response-code HTTP-status classes, one per class
// documented in the standard's response-code table. Match via errors.Is
// regardless of the specific case code.
var (
	ErrBadRequest          = errors.New("snap: bad request")
	ErrUnauthorized        = errors.New("snap: unauthorized")
	ErrForbidden           = errors.New("snap: forbidden")
	ErrNotFound            = errors.New("snap: not found")
	ErrInternalServerError = errors.New("snap: internal server error")
	ErrServiceUnavailable  = errors.New("snap: service unavailable")
	ErrTimeout             = errors.New("snap: timeout")

	// ErrUnmappedResponseCode is returned by ResponseCodeError (and the
	// status-derived fallbacks in transport.go/token.go) for an HTTP-status
	// class with no dedicated sentinel above — including statuses SNAP's own
	// table doesn't document (e.g. 429, 502, 520-527) that a proxy, WAF, or
	// gateway in front of the real server commonly returns. Exported so
	// callers can errors.Is against this bucket instead of losing all
	// machine-readable signal for exactly the class of error most likely to
	// come from infrastructure rather than the SNAP server itself.
	ErrUnmappedResponseCode = errors.New("snap: unmapped response code class")
)

// ParseResponseCode splits a 7-character SNAP responseCode into
// HTTPStatus(3) + ServiceCode(2) + CaseCode(2). It returns a non-nil error
// for any code not exactly 7 digits or containing non-digit characters.
func ParseResponseCode(code string) (httpStatus int, serviceCode, caseCode string, err error) {
	if len(code) != 7 {
		return 0, "", "", fmt.Errorf("snap: parse response code %q: want 7 digits, got %d characters", truncateForError(code), len(code))
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return 0, "", "", fmt.Errorf("snap: parse response code %q: contains non-digit characters", truncateForError(code))
		}
	}
	httpStatus = int(code[0]-'0')*100 + int(code[1]-'0')*10 + int(code[2]-'0')
	return httpStatus, code[3:5], code[5:7], nil
}

// truncateForError caps an untrusted string before it's interpolated into an
// error message, so a pathologically large input can't produce a
// proportionally large (and typically logged) error string.
func truncateForError(s string) string {
	const max = 16
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ResponseCodeError parses code and returns the sentinel error matching its
// HTTP-status class, wrapped with the raw code via %w so callers can recover
// it while still using errors.Is against the class sentinel. For a class
// with no matching sentinel, it returns ErrUnmappedResponseCode wrapped the
// same way, never nil.
func ResponseCodeError(code string) error {
	httpStatus, _, _, err := ParseResponseCode(code)
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: response code %s", sentinelForHTTPStatus(httpStatus), code)
}

// checkResponseStatus is the authoritative success/failure check for a
// decoded SNAP response: the transport-level HTTP status always wins over
// what responseCode claims. A non-2xx HTTP status is never treated as
// success, even if responseCode's own embedded class says otherwise (e.g. a
// stale cached body from a misbehaving intermediary, or an out-of-band 500
// with a templated "success" body) — envelopeError alone cannot catch this,
// since it only looks at the code's own embedded status, never the
// transport's actual one.
func checkResponseStatus(responseCode string, httpStatus int) error {
	if httpStatus < 200 || httpStatus >= 300 {
		if responseCode != "" {
			if err := envelopeError(responseCode); err != nil {
				// Join the transport-status sentinel on top of whatever
				// envelopeError produced, so a non-2xx failure is always
				// errors.Is-matchable via the actual HTTP status — including
				// when responseCode itself is malformed, where envelopeError
				// alone returns a plain parse error with no sentinel at all.
				return fmt.Errorf("%w: %w", sentinelForHTTPStatus(httpStatus), err)
			}
			// responseCode claims success but the transport status
			// disagrees — the transport status is authoritative; fall
			// through to the status-derived sentinel below rather than
			// trusting the body over the transport.
		}
		return fmt.Errorf("%w: http status %d", sentinelForHTTPStatus(httpStatus), httpStatus)
	}
	return envelopeError(responseCode)
}

// sentinelForHTTPStatus maps an HTTP status to the sentinel error for its
// class, shared between ResponseCodeError (which has a full 7-digit SNAP
// responseCode) and callers that only have a raw HTTP status to go on (e.g.
// a non-2xx response whose body doesn't carry a responseCode field at all).
func sentinelForHTTPStatus(httpStatus int) error {
	switch httpStatus {
	case 400:
		return ErrBadRequest
	case 401:
		return ErrUnauthorized
	case 403:
		return ErrForbidden
	case 404:
		return ErrNotFound
	case 500:
		return ErrInternalServerError
	case 503:
		return ErrServiceUnavailable
	case 504:
		return ErrTimeout
	default:
		return ErrUnmappedResponseCode
	}
}
