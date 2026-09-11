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
	"sync/atomic"
	"testing"
)

func TestCustomerTopUp_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003800",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"ref-1",
   "partnerReferenceNo":"partner-ref-1",
   "sessionId":"sess-1",
   "customerNumber":"98765",
   "amount":{"value":"100000.00","currency":"IDR"},
   "referenceNumber":"REF993883"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up"
	tr := &Transport{}
	resp, err := CustomerTopUp(context.Background(), tr, hb, CustomerTopUpRequest{
		PartnerReferenceNo: "partner-ref-1",
	})
	if err != nil {
		t.Fatalf("CustomerTopUp() error = %v", err)
	}

	want := CustomerTopUpResponse{
		ResponseCode:       "2003800",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "ref-1",
		PartnerReferenceNo: "partner-ref-1",
		SessionID:          "sess-1",
		CustomerNumber:     "98765",
		Amount:             &Money{Value: "100000.00", Currency: "IDR"},
		ReferenceNumber:    "REF993883",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CustomerTopUp() = %+v, want %+v", resp, want)
	}
}

func TestCustomerTopUp_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up"
	tr := &Transport{}
	req := CustomerTopUpRequest{
		PartnerReferenceNo: "partner-ref-1",
		CategoryID:         json.RawMessage(`5`),
		Notes:              "top up",
	}
	if _, err := CustomerTopUp(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CustomerTopUp() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if string(got["categoryId"]) != "5" {
		t.Errorf(`wire body["categoryId"] = %s, want bare number 5`, got["categoryId"])
	}
	var notes string
	if err := json.Unmarshal(got["notes"], &notes); err != nil || notes != "top up" {
		t.Errorf(`wire body["notes"] = %s, want "top up"`, got["notes"])
	}
}

// TestCustomerTopUp_MandatoryFieldAlwaysSerialized pins that
// PartnerReferenceNo — the only request field without omitempty — is
// always present on the wire, even as "".
func TestCustomerTopUp_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up"
	tr := &Transport{}
	if _, err := CustomerTopUp(context.Background(), tr, hb, CustomerTopUpRequest{}); err != nil {
		t.Fatalf("CustomerTopUp() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	v, ok := got["partnerReferenceNo"]
	if !ok {
		t.Fatal(`wire body missing "partnerReferenceNo" key; want it always present, even as ""`)
	}
	if v != "" {
		t.Errorf(`wire body["partnerReferenceNo"] = %v, want ""`, v)
	}
	if _, ok := got["categoryId"]; ok {
		t.Error(`wire body has "categoryId" key, want it omitted (Optional here)`)
	}
}

func TestCustomerTopUp_MalformedCategoryIDIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2003800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up"
	tr := &Transport{}
	_, err := CustomerTopUp(context.Background(), tr, hb, CustomerTopUpRequest{
		CategoryID: json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("CustomerTopUp() error = nil, want non-nil for malformed CategoryID JSON")
	}
	if requested.Load() {
		t.Error("CustomerTopUp() sent an HTTP request despite a request-encoding failure")
	}
}

func TestCustomerTopUp_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up"
	tr := &Transport{}
	_, err := CustomerTopUp(context.Background(), tr, hb, CustomerTopUpRequest{})
	if err == nil {
		t.Fatal("CustomerTopUp() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("CustomerTopUp() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestCustomerTopUp_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up"
	tr := &Transport{}
	resp, err := CustomerTopUp(context.Background(), tr, hb, CustomerTopUpRequest{})
	if err == nil {
		t.Fatalf("CustomerTopUp() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("CustomerTopUp() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestCustomerTopUp_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNumber":"REF993883"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up"
	tr := &Transport{}
	resp, err := CustomerTopUp(context.Background(), tr, hb, CustomerTopUpRequest{})
	if err == nil {
		t.Fatalf("CustomerTopUp() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
