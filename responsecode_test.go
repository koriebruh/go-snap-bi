package snap

import (
	"errors"
	"strings"
	"testing"
)

func TestParseResponseCode(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		wantHTTPStatus int
		wantService    string
		wantCase       string
		wantErr        bool
	}{
		{name: "200 success", code: "2000000", wantHTTPStatus: 200, wantService: "00", wantCase: "00"},
		{name: "400 bad request", code: "4001101", wantHTTPStatus: 400, wantService: "11", wantCase: "01"},
		{name: "401 unauthorized", code: "4011300", wantHTTPStatus: 401, wantService: "13", wantCase: "00"},
		{name: "403 forbidden", code: "4033601", wantHTTPStatus: 403, wantService: "36", wantCase: "01"},
		{name: "404 not found", code: "4045300", wantHTTPStatus: 404, wantService: "53", wantCase: "00"},
		{name: "500 internal server error", code: "5007400", wantHTTPStatus: 500, wantService: "74", wantCase: "00"},
		{name: "503 service unavailable", code: "5037300", wantHTTPStatus: 503, wantService: "73", wantCase: "00"},
		{name: "504 timeout", code: "5041100", wantHTTPStatus: 504, wantService: "11", wantCase: "00"},
		{name: "too short", code: "40011", wantErr: true},
		{name: "too long", code: "400110100", wantErr: true},
		{name: "non-digit characters", code: "40A1101", wantErr: true},
		{name: "empty string", code: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpStatus, serviceCode, caseCode, err := ParseResponseCode(tt.code)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseResponseCode(%q) error = nil, want non-nil", tt.code)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseResponseCode(%q) unexpected error: %v", tt.code, err)
			}
			if httpStatus != tt.wantHTTPStatus {
				t.Errorf("httpStatus = %d, want %d", httpStatus, tt.wantHTTPStatus)
			}
			if serviceCode != tt.wantService {
				t.Errorf("serviceCode = %q, want %q", serviceCode, tt.wantService)
			}
			if caseCode != tt.wantCase {
				t.Errorf("caseCode = %q, want %q", caseCode, tt.wantCase)
			}
		})
	}
}

func TestResponseCodeError(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		wantErrIs error
	}{
		{name: "400 bad request", code: "4001101", wantErrIs: ErrBadRequest},
		{name: "401 unauthorized", code: "4011300", wantErrIs: ErrUnauthorized},
		{name: "403 forbidden", code: "4033601", wantErrIs: ErrForbidden},
		{name: "404 not found", code: "4045300", wantErrIs: ErrNotFound},
		{name: "500 internal server error", code: "5007400", wantErrIs: ErrInternalServerError},
		{name: "503 service unavailable", code: "5037300", wantErrIs: ErrServiceUnavailable},
		{name: "504 timeout", code: "5041100", wantErrIs: ErrTimeout},
		{name: "unmapped class", code: "2000000", wantErrIs: ErrUnmappedResponseCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ResponseCodeError(tt.code)
			if err == nil {
				t.Fatalf("ResponseCodeError(%q) = nil, want non-nil", tt.code)
			}
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("ResponseCodeError(%q) = %v, want errors.Is match for %v", tt.code, err, tt.wantErrIs)
			}
			if !strings.Contains(err.Error(), tt.code) {
				t.Errorf("ResponseCodeError(%q) message %q does not contain the raw code", tt.code, err.Error())
			}
		})
	}

	t.Run("malformed code returns parse error, not a sentinel", func(t *testing.T) {
		err := ResponseCodeError("bad")
		if err == nil {
			t.Fatal("ResponseCodeError(\"bad\") = nil, want non-nil")
		}
		if errors.Is(err, ErrBadRequest) {
			t.Error("malformed code should not match ErrBadRequest sentinel")
		}
	})
}

// TestCheckResponseStatus directly exercises every branch of
// checkResponseStatus, including the 2xx-HTTP-status paths that no
// httptest-based binding test reaches (every existing non-2xx-responseCode
// test also happens to send a matching non-2xx HTTP status, and the only
// no-responseCode fixture is served with a non-2xx status too — so the
// "HTTP 200 but the body itself signals an error" and "HTTP 200 with no
// responseCode at all" branches were previously dead code as far as the
// test suite could tell, a santa-loop finding: statement coverage on the
// final `return envelopeError(...)` line looked like 100% because it always
// executes and returns nil on every existing test, never proving it can
// also return non-nil).
func TestCheckResponseStatus(t *testing.T) {
	tests := []struct {
		name         string
		responseCode string
		httpStatus   int
		wantErr      bool
		wantErrIs    error // nil means don't check errors.Is, just non-nil/nil
	}{
		{"2xx status, 2xx-class code: success", "2001200", 200, false, nil},
		{"2xx status, empty code: success (mandatory-field check is the caller's job)", "", 200, false, nil},
		{"2xx status, malformed code: error", "abc", 200, true, nil},
		{"2xx status, 4xx-class code in body: error, not silently accepted", "4001200", 200, true, ErrBadRequest},
		{"non-2xx status, matching non-2xx code: error via envelopeError", "4001200", 400, true, ErrBadRequest},
		{"non-2xx status, empty code: error via status fallback", "", 500, true, ErrInternalServerError},
		{"non-2xx status, 2xx-class code in body: transport status wins, error", "2001200", 500, true, ErrInternalServerError},
		{"non-2xx status, malformed code: error", "abc", 500, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkResponseStatus(tt.responseCode, tt.httpStatus)
			if tt.wantErr && err == nil {
				t.Fatalf("checkResponseStatus(%q, %d) = nil, want non-nil", tt.responseCode, tt.httpStatus)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("checkResponseStatus(%q, %d) = %v, want nil", tt.responseCode, tt.httpStatus, err)
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("checkResponseStatus(%q, %d) = %v, want errors.Is match for %v", tt.responseCode, tt.httpStatus, err, tt.wantErrIs)
			}
		})
	}
}
