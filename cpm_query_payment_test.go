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

func TestCPMQueryPaymentTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(CPMQueryPaymentRequest{}).NumField(); n != 7 {
		t.Errorf("CPMQueryPaymentRequest has %d fields, want 7", n)
	}
	if n := reflect.TypeOf(CPMQueryPaymentResponse{}).NumField(); n != 10 {
		t.Errorf("CPMQueryPaymentResponse has %d fields, want 10", n)
	}
}

func TestCPMQueryPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2006100",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"REF001",
   "originalPartnerReferenceNo":"PARTNER001",
   "originalExternalId":"EXT001",
   "title":"Coffee order",
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "paidTime":"2026-09-13T09:05:00+07:00",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-query"
	tr := &Transport{}
	resp, err := CPMQueryPayment(context.Background(), tr, hb, CPMQueryPaymentRequest{})
	if err != nil {
		t.Fatalf("CPMQueryPayment() error = %v", err)
	}

	want := CPMQueryPaymentResponse{
		ResponseCode:               "2006100",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "REF001",
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalExternalID:         "EXT001",
		Title:                      "Coffee order",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		PaidTime:                   "2026-09-13T09:05:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CPMQueryPayment() = %+v, want %+v", resp, want)
	}
}

func TestCPMQueryPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006100","responseMessage":"ok","latestTransactionStatus":"00","paidTime":"2026-09-13T09:05:00+07:00"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-query"
	tr := &Transport{}
	req := CPMQueryPaymentRequest{
		OriginalReferenceNo:        "REF001",
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalExternalID:         "EXT001",
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		ExternalStoreID:            "STORE01",
		AdditionalInfo:             json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := CPMQueryPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CPMQueryPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"originalReferenceNo":        "REF001",
		"originalPartnerReferenceNo": "PARTNER001",
		"originalExternalId":         "EXT001",
		"merchantId":                 "MERCH01",
		"subMerchantId":              "SUBMERCH01",
		"externalStoreId":            "STORE01",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestCPMQueryPaymentRequest_ZeroValueOmitsAllFields pins that every
// request field is Optional, so a zero-value request marshals to an
// empty object.
func TestCPMQueryPaymentRequest_ZeroValueOmitsAllFields(t *testing.T) {
	b, err := json.Marshal(CPMQueryPaymentRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("marshaled zero-value request = %v, want empty object (all fields Optional)", got)
	}
}

// TestCPMQueryPaymentResponse_MandatoryFieldsAlwaysSerialized pins
// that LatestTransactionStatus and PaidTime — the response fields
// without omitempty — always serialize from a zero-value response,
// and every other field is omitted when unset.
func TestCPMQueryPaymentResponse_MandatoryFieldsAlwaysSerialized(t *testing.T) {
	b, err := json.Marshal(CPMQueryPaymentResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{"responseCode": "", "responseMessage": "", "latestTransactionStatus": "", "paidTime": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestCPMQueryPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4006100","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-query"
	tr := &Transport{}
	_, err := CPMQueryPayment(context.Background(), tr, hb, CPMQueryPaymentRequest{})
	if err == nil {
		t.Fatal("CPMQueryPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("CPMQueryPayment() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestCPMQueryPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2006100","responseMessage":"ok","latestTransactionStatus":"00","paidTime":"2026-09-13T09:05:00+07:00"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-query"
	tr := &Transport{}
	resp, err := CPMQueryPayment(context.Background(), tr, hb, CPMQueryPaymentRequest{})
	if err == nil {
		t.Fatalf("CPMQueryPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("CPMQueryPayment() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestCPMQueryPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"title":"Coffee order"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-query"
	tr := &Transport{}
	resp, err := CPMQueryPayment(context.Background(), tr, hb, CPMQueryPaymentRequest{})
	if err == nil {
		t.Fatalf("CPMQueryPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
