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

// TestQRMPMCancelPaymentTypes_FieldCounts guards against a field
// silently added to either type without updating the wire-assertion
// tests below — those tests catch renamed/omitempty-flipped fields but
// not an addition, since a new field defaults to unset and unasserted.
func TestQRMPMCancelPaymentTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(QRMPMCancelPaymentRequest{}).NumField(); n != 8 {
		t.Errorf("QRMPMCancelPaymentRequest has %d fields, want 8", n)
	}
	if n := reflect.TypeOf(QRMPMCancelPaymentResponse{}).NumField(); n != 4 {
		t.Errorf("QRMPMCancelPaymentResponse has %d fields, want 4", n)
	}
}

func TestQRMPMCancelPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2007700",
   "responseMessage":"Request has been processed successfully",
   "cancelTime":"2020-12-20T10:00:00+07:00",
   "transactionDate":"2020-12-20T09:00:00+07:00"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-cancel"
	tr := &Transport{}
	resp, err := QRMPMCancelPayment(context.Background(), tr, hb, QRMPMCancelPaymentRequest{
		MerchantID: "MERCH01",
		Reason:     "customer requested cancellation",
	})
	if err != nil {
		t.Fatalf("QRMPMCancelPayment() error = %v", err)
	}

	want := QRMPMCancelPaymentResponse{
		ResponseCode:    "2007700",
		ResponseMessage: "Request has been processed successfully",
		CancelTime:      "2020-12-20T10:00:00+07:00",
		TransactionDate: "2020-12-20T09:00:00+07:00",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("QRMPMCancelPayment() = %+v, want %+v", resp, want)
	}
}

func TestQRMPMCancelPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2007700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-cancel"
	tr := &Transport{}
	req := QRMPMCancelPaymentRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		ExternalStoreID:            "STORE01",
		Reason:                     "customer requested cancellation",
		Amount:                     &Money{Value: "25000.00", Currency: "IDR"},
	}
	if _, err := QRMPMCancelPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("QRMPMCancelPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"originalPartnerReferenceNo": "partner-ref-1",
		"originalReferenceNo":        "ref-1",
		"originalExternalId":         "ext-1",
		"merchantId":                 "MERCH01",
		"subMerchantId":              "SUBMERCH01",
		"externalStoreId":            "STORE01",
		"reason":                     "customer requested cancellation",
		"amount":                     map[string]any{"value": "25000.00", "currency": "IDR"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestQRMPMCancelPayment_MandatoryFieldsAlwaysSerialized pins that
// MerchantID and Reason — the request fields without omitempty — are
// always present on the wire, even as "", and every other field (all
// Optional) is omitted when unset.
func TestQRMPMCancelPayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2007700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-cancel"
	tr := &Transport{}
	if _, err := QRMPMCancelPayment(context.Background(), tr, hb, QRMPMCancelPaymentRequest{}); err != nil {
		t.Fatalf("QRMPMCancelPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"merchantId": "", "reason": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestQRMPMCancelPaymentResponse_CancelTimeAndTransactionDateHaveOmitempty
// pins that both response fields are Optional/Conditional (omitempty),
// matching research's C/O markers — a zero-value response marshals to
// just the two envelope fields.
func TestQRMPMCancelPaymentResponse_CancelTimeAndTransactionDateHaveOmitempty(t *testing.T) {
	b, err := json.Marshal(QRMPMCancelPaymentResponse{})
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

func TestQRMPMCancelPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4007700","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-cancel"
	tr := &Transport{}
	_, err := QRMPMCancelPayment(context.Background(), tr, hb, QRMPMCancelPaymentRequest{})
	if err == nil {
		t.Fatal("QRMPMCancelPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("QRMPMCancelPayment() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestQRMPMCancelPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2007700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-cancel"
	tr := &Transport{}
	resp, err := QRMPMCancelPayment(context.Background(), tr, hb, QRMPMCancelPaymentRequest{})
	if err == nil {
		t.Fatalf("QRMPMCancelPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("QRMPMCancelPayment() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestQRMPMCancelPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cancelTime":"2020-12-20T10:00:00+07:00"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-cancel"
	tr := &Transport{}
	resp, err := QRMPMCancelPayment(context.Background(), tr, hb, QRMPMCancelPaymentRequest{})
	if err == nil {
		t.Fatalf("QRMPMCancelPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
