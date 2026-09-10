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
		{name: "unmapped class", code: "2000000", wantErrIs: errUnmappedResponseCode},
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
