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

// fixture is hand-built, not a captured worked example: the design doc
// records that none was available in the portal's Code Snippets tab for
// this endpoint at research time. The apiKey field's quoting here is an
// assumption, not observed wire data — see TestAccountCreation_APIKeyAcceptsEitherWireShape,
// which is why APIKey is typed json.RawMessage rather than string.
func TestAccountCreation_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000600",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "authCode":"authcode-123",
   "apiKey":"998877",
   "accountId":"account-456",
   "state":"csrf-state-789",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-creation"
	tr := &Transport{}
	resp, err := AccountCreation(context.Background(), tr, hb, AccountCreationRequest{
		PartnerReferenceNo: "2020102900000000000001",
	})
	if err != nil {
		t.Fatalf("AccountCreation() error = %v", err)
	}

	want := AccountCreationResponse{
		ResponseCode:       "2000600",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		AuthCode:           "authcode-123",
		APIKey:             json.RawMessage(`"998877"`),
		AccountID:          "account-456",
		State:              "csrf-state-789",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AccountCreation() = %+v, want %+v", resp, want)
	}
}

// TestAccountCreation_APIKeyAcceptsEitherWireShape is the regression test
// for a santa-loop finding: the standard's Guides table labels apiKey
// "Numeric" with no worked example to confirm whether the server quotes it
// as a JSON string. A plain string field would hard-fail the entire decode
// (discarding ReferenceNo/AccountID/AuthCode too) for an unquoted numeric
// value, on a non-idempotent operation where the account may already exist
// server-side. json.RawMessage tolerates either shape.
func TestAccountCreation_APIKeyAcceptsEitherWireShape(t *testing.T) {
	tests := []struct {
		name       string
		apiKeyJSON string
	}{
		{"quoted string", `"998877"`},
		{"unquoted number", `998877`},
		// json null decodes to a non-nil 4-byte RawMessage("null"), distinct
		// from an absent key (which decodes to nil) — a third caller-visible
		// shape found in santa-loop review, worth pinning explicitly rather
		// than leaving it as an unstated side effect of the type choice.
		{"json null", `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"responseCode":"2000600","responseMessage":"ok","accountId":"account-456","apiKey":`+tt.apiKeyJSON+`}`)
			}))
			defer server.Close()

			hb := testHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/registration-account-creation"
			tr := &Transport{}
			resp, err := AccountCreation(context.Background(), tr, hb, AccountCreationRequest{})
			if err != nil {
				t.Fatalf("AccountCreation() error = %v, want nil — apiKey shape must not break the whole decode", err)
			}
			if resp.AccountID != "account-456" {
				t.Errorf("AccountID = %q, want %q (lost due to a decode failure elsewhere in the struct)", resp.AccountID, "account-456")
			}
			if string(resp.APIKey) != tt.apiKeyJSON {
				t.Errorf("APIKey = %s, want %s", resp.APIKey, tt.apiKeyJSON)
			}
		})
	}
}

func TestAccountCreation_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-creation"
	tr := &Transport{}
	req := AccountCreationRequest{
		PartnerReferenceNo: "ref-1",
		Email:              "user@example.com",
		PhoneNo:            "6281234567890",
		DeviceInfo: &DeviceInfo{
			OS:           "Android",
			OSVersion:    "14",
			Model:        "Pixel 8",
			Manufacturer: "Google",
		},
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := AccountCreation(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AccountCreation() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerReferenceNo"] != "ref-1" {
		t.Errorf(`wire body["partnerReferenceNo"] = %v, want "ref-1"`, got["partnerReferenceNo"])
	}
	if got["email"] != "user@example.com" {
		t.Errorf(`wire body["email"] = %v, want "user@example.com"`, got["email"])
	}
	if got["phoneNo"] != "6281234567890" {
		t.Errorf(`wire body["phoneNo"] = %v, want "6281234567890"`, got["phoneNo"])
	}
	deviceInfo, ok := got["deviceInfo"].(map[string]any)
	if !ok || deviceInfo["os"] != "Android" || deviceInfo["osVersion"] != "14" || deviceInfo["model"] != "Pixel 8" || deviceInfo["manufacturer"] != "Google" {
		t.Errorf(`wire body["deviceInfo"] = %v, want {os:Android osVersion:14 model:"Pixel 8" manufacturer:Google}`, got["deviceInfo"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

func TestAccountCreation_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000600","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-creation"
	tr := &Transport{}
	_, err := AccountCreation(context.Background(), tr, hb, AccountCreationRequest{})
	if err == nil {
		t.Fatal("AccountCreation() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("AccountCreation() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

// TestAccountCreation_NonTwoXXStatusWithTwoXXBodyIsError mirrors Phase 2's
// santa-loop round-3 HIGH-finding regression test: an HTTP 500 whose body
// still carries a 2xx-class responseCode must not be treated as success.
func TestAccountCreation_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-creation"
	tr := &Transport{}
	resp, err := AccountCreation(context.Background(), tr, hb, AccountCreationRequest{})
	if err == nil {
		t.Fatalf("AccountCreation() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("AccountCreation() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

// TestAccountCreation_TwoXXStatusWithNoResponseCodeIsError mirrors Phase 3's
// santa-loop round-1 finding: the "HTTP 200, but the JSON body carries no
// responseCode" path must be exercised at HTTP 200, not only paired with a
// non-2xx status.
func TestAccountCreation_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accountId":"account-456"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-creation"
	tr := &Transport{}
	resp, err := AccountCreation(context.Background(), tr, hb, AccountCreationRequest{})
	if err == nil {
		t.Fatalf("AccountCreation() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
