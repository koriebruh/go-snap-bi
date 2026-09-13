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

func TestCPMCancelPaymentTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(CPMCancelPaymentRequest{}).NumField(); n != 9 {
		t.Errorf("CPMCancelPaymentRequest has %d fields, want 9", n)
	}
	if n := reflect.TypeOf(CPMCancelPaymentResponse{}).NumField(); n != 8 {
		t.Errorf("CPMCancelPaymentResponse has %d fields, want 8", n)
	}
}

func TestCPMCancelPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2006200",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"PARTNER001",
   "originalReferenceNo":"REF001",
   "originalExternalId":"EXT001",
   "cancelTime":"2026-09-13T10:00:00+07:00",
   "transactionDate":"2026-09-13T09:00:00+07:00",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-cancel"
	tr := &Transport{}
	resp, err := CPMCancelPayment(context.Background(), tr, hb, CPMCancelPaymentRequest{
		OriginalPartnerReferenceNo: "PARTNER001",
	})
	if err != nil {
		t.Fatalf("CPMCancelPayment() error = %v", err)
	}

	want := CPMCancelPaymentResponse{
		ResponseCode:               "2006200",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		OriginalExternalID:         "EXT001",
		CancelTime:                 "2026-09-13T10:00:00+07:00",
		TransactionDate:            "2026-09-13T09:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CPMCancelPayment() = %+v, want %+v", resp, want)
	}
}

func TestCPMCancelPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-cancel"
	tr := &Transport{}
	req := CPMCancelPaymentRequest{
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		OriginalExternalID:         "EXT001",
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		ExternalStoreID:            "STORE01",
		Amount:                     &Money{Value: "50000.00", Currency: "IDR"},
		Reason:                     "customer requested cancellation",
		AdditionalInfo:             json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := CPMCancelPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CPMCancelPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"originalPartnerReferenceNo": "PARTNER001",
		"originalReferenceNo":        "REF001",
		"originalExternalId":         "EXT001",
		"merchantId":                 "MERCH01",
		"subMerchantId":              "SUBMERCH01",
		"externalStoreId":            "STORE01",
		"amount":                     map[string]any{"value": "50000.00", "currency": "IDR"},
		"reason":                     "customer requested cancellation",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestCPMCancelPayment_MandatoryFieldAlwaysSerialized pins that
// OriginalPartnerReferenceNo — the only request field without
// omitempty — is always present on the wire, even as "", and every
// other field is omitted when unset.
func TestCPMCancelPayment_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-cancel"
	tr := &Transport{}
	if _, err := CPMCancelPayment(context.Background(), tr, hb, CPMCancelPaymentRequest{}); err != nil {
		t.Fatalf("CPMCancelPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"originalPartnerReferenceNo": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestCPMCancelPaymentResponse_ZeroValueOmitsOptionalFields pins that
// a zero-value response marshals to just the two envelope fields.
func TestCPMCancelPaymentResponse_ZeroValueOmitsOptionalFields(t *testing.T) {
	b, err := json.Marshal(CPMCancelPaymentResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{"responseCode": "", "responseMessage": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestCPMCancelPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4006200","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-cancel"
	tr := &Transport{}
	_, err := CPMCancelPayment(context.Background(), tr, hb, CPMCancelPaymentRequest{})
	if err == nil {
		t.Fatal("CPMCancelPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("CPMCancelPayment() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestCPMCancelPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2006200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-cancel"
	tr := &Transport{}
	resp, err := CPMCancelPayment(context.Background(), tr, hb, CPMCancelPaymentRequest{})
	if err == nil {
		t.Fatalf("CPMCancelPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("CPMCancelPayment() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestCPMCancelPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cancelTime":"2026-09-13T10:00:00+07:00"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-cancel"
	tr := &Transport{}
	resp, err := CPMCancelPayment(context.Background(), tr, hb, CPMCancelPaymentRequest{})
	if err == nil {
		t.Fatalf("CPMCancelPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
