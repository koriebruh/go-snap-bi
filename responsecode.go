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

	// errUnmappedResponseCode is returned by ResponseCodeError for an
	// HTTP-status class with no dedicated sentinel above (e.g. 2xx success
	// codes, or classes not documented in the standard's table).
	errUnmappedResponseCode = errors.New("snap: unmapped response code class")
)

// ParseResponseCode splits a 7-character SNAP responseCode into
// HTTPStatus(3) + ServiceCode(2) + CaseCode(2). It returns a non-nil error
// for any code not exactly 7 digits or containing non-digit characters.
func ParseResponseCode(code string) (httpStatus int, serviceCode, caseCode string, err error) {
	if len(code) != 7 {
		return 0, "", "", fmt.Errorf("snap: parse response code %q: want 7 digits, got %d characters", code, len(code))
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return 0, "", "", fmt.Errorf("snap: parse response code %q: contains non-digit characters", code)
		}
	}
	httpStatus = int(code[0]-'0')*100 + int(code[1]-'0')*10 + int(code[2]-'0')
	return httpStatus, code[3:5], code[5:7], nil
}

// ResponseCodeError parses code and returns the sentinel error matching its
// HTTP-status class, wrapped with the raw code via %w so callers can recover
// it while still using errors.Is against the class sentinel. For a class
// with no matching sentinel, it returns errUnmappedResponseCode wrapped the
// same way, never nil.
func ResponseCodeError(code string) error {
	httpStatus, _, _, err := ParseResponseCode(code)
	if err != nil {
		return err
	}

	var sentinel error
	switch httpStatus {
	case 400:
		sentinel = ErrBadRequest
	case 401:
		sentinel = ErrUnauthorized
	case 403:
		sentinel = ErrForbidden
	case 404:
		sentinel = ErrNotFound
	case 500:
		sentinel = ErrInternalServerError
	case 503:
		sentinel = ErrServiceUnavailable
	case 504:
		sentinel = ErrTimeout
	default:
		sentinel = errUnmappedResponseCode
	}
	return fmt.Errorf("%w: response code %s", sentinel, code)
}
