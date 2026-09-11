package snap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

func TestCardRegistrationSetLimit_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000200",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration/card-bind-limit"
	tr := &Transport{}
	resp, err := CardRegistrationSetLimit(context.Background(), tr, hb, CardRegistrationSetLimitRequest{BankCardToken: "6d7963617264746f6b656e"})
	if err != nil {
		t.Fatalf("CardRegistrationSetLimit() error = %v", err)
	}

	want := CardRegistrationSetLimitResponse{
		ResponseCode:       "2000200",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CardRegistrationSetLimit() = %+v, want %+v", resp, want)
	}
}

func TestCardRegistrationSetLimit_RequestBodyRoundTrips(t *testing.T) {
	var mu sync.Mutex
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		mu.Lock()
		gotBody = b
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2000200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration/card-bind-limit"
	tr := &Transport{}
	req := CardRegistrationSetLimitRequest{
		PartnerReferenceNo: "ref-1",
		BankCardToken:      "6d7963617264746f6b656e",
		Limit:              json.RawMessage(`"1000000"`),
		OTP:                "12345678",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := CardRegistrationSetLimit(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CardRegistrationSetLimit() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["bankCardToken"] != "6d7963617264746f6b656e" {
		t.Errorf(`wire body["bankCardToken"] = %v, want "6d7963617264746f6b656e"`, got["bankCardToken"])
	}
	if got["limit"] != "1000000" {
		t.Errorf(`wire body["limit"] = %v, want "1000000"`, got["limit"])
	}
	if got["otp"] != "12345678" {
		t.Errorf(`wire body["otp"] = %v, want "12345678"`, got["otp"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestCardRegistrationSetLimit_LimitAcceptsEitherWireShape mirrors
// TestCardRegistration_LimitAndCardDataAcceptEitherWireShape: the Guides
// tab labels Limit "decimal", which permits a non-string JSON
// representation even though the portal's worked example quotes it. Limit
// is request-only, so this asserts the captured wire body preserves the
// input shape (whitespace compacted, as encoding/json always does for
// RawMessage), not only that the call succeeded.
func TestCardRegistrationSetLimit_LimitAcceptsEitherWireShape(t *testing.T) {
	tests := []struct {
		name  string
		limit string
	}{
		{"quoted string", `"1000000"`},
		{"unquoted number", `1000000`},
		{"json null", `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var gotBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				mu.Lock()
				gotBody = b
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"responseCode":"2000200","responseMessage":"ok"}`)
			}))
			defer server.Close()

			hb := testHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/registration/card-bind-limit"
			tr := &Transport{}
			req := CardRegistrationSetLimitRequest{
				BankCardToken: "6d7963617264746f6b656e",
				Limit:         json.RawMessage(tt.limit),
			}
			if _, err := CardRegistrationSetLimit(context.Background(), tr, hb, req); err != nil {
				t.Fatalf("CardRegistrationSetLimit() error = %v, want nil — limit shape must not break the call", err)
			}

			mu.Lock()
			defer mu.Unlock()
			var got map[string]json.RawMessage
			if err := json.Unmarshal(gotBody, &got); err != nil {
				t.Fatalf("decode request body the server received: %v", err)
			}
			if string(got["limit"]) != tt.limit {
				t.Errorf(`wire body["limit"] = %s, want %s (shape preserved)`, got["limit"], tt.limit)
			}
		})
	}
}

// TestCardRegistrationSetLimit_NonNumericUnquotedLimitFailsEncode mirrors
// TestCardRegistration_NonNumericUnquotedLimitFailsEncode: a formatted
// decimal is not valid JSON on its own, so json.Marshal fails and the
// request is never sent.
func TestCardRegistrationSetLimit_NonNumericUnquotedLimitFailsEncode(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"responseCode":"2000200","responseMessage":"ok"}`)
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration/card-bind-limit"
	tr := &Transport{}
	req := CardRegistrationSetLimitRequest{
		BankCardToken: "6d7963617264746f6b656e",
		Limit:         json.RawMessage(`1,000,000`),
	}
	_, err := CardRegistrationSetLimit(context.Background(), tr, hb, req)
	if err == nil {
		t.Fatal("CardRegistrationSetLimit() error = nil, want non-nil for a non-numeric unquoted limit")
	}
	if called {
		t.Error("CardRegistrationSetLimit() reached the server despite a marshal failure; want the request never sent")
	}
}

// TestCardRegistrationSetLimit_BankCardTokenAlwaysSerialized pins that
// BankCardToken — the one request field without omitempty — is always
// present on the wire, even as "", mirroring
// TestAccountBinding_MerchantIDAlwaysSerialized/
// TestAccountUnbinding_MerchantIDAlwaysSerialized.
func TestCardRegistrationSetLimit_BankCardTokenAlwaysSerialized(t *testing.T) {
	var mu sync.Mutex
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		mu.Lock()
		gotBody = b
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"responseCode":"2000200","responseMessage":"ok"}`)
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration/card-bind-limit"
	tr := &Transport{}
	if _, err := CardRegistrationSetLimit(context.Background(), tr, hb, CardRegistrationSetLimitRequest{}); err != nil {
		t.Fatalf("CardRegistrationSetLimit() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	bankCardToken, ok := got["bankCardToken"]
	if !ok {
		t.Fatal(`wire body missing "bankCardToken" key; want it always present, even as ""`)
	}
	if bankCardToken != "" {
		t.Errorf(`wire body["bankCardToken"] = %v, want ""`, bankCardToken)
	}
}

func TestCardRegistrationSetLimit_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000200","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration/card-bind-limit"
	tr := &Transport{}
	_, err := CardRegistrationSetLimit(context.Background(), tr, hb, CardRegistrationSetLimitRequest{BankCardToken: "6d7963617264746f6b656e"})
	if err == nil {
		t.Fatal("CardRegistrationSetLimit() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("CardRegistrationSetLimit() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestCardRegistrationSetLimit_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration/card-bind-limit"
	tr := &Transport{}
	resp, err := CardRegistrationSetLimit(context.Background(), tr, hb, CardRegistrationSetLimitRequest{BankCardToken: "6d7963617264746f6b656e"})
	if err == nil {
		t.Fatalf("CardRegistrationSetLimit() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("CardRegistrationSetLimit() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestCardRegistrationSetLimit_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"2020102977770000000009"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration/card-bind-limit"
	tr := &Transport{}
	resp, err := CardRegistrationSetLimit(context.Background(), tr, hb, CardRegistrationSetLimitRequest{BankCardToken: "6d7963617264746f6b656e"})
	if err == nil {
		t.Fatalf("CardRegistrationSetLimit() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
