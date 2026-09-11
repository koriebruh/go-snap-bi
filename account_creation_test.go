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
		APIKey:             "998877",
		AccountID:          "account-456",
		State:              "csrf-state-789",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AccountCreation() = %+v, want %+v", resp, want)
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
