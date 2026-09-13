package transferdebit

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

	snap "github.com/koriebruh/go-snap-bi"
	"github.com/koriebruh/go-snap-bi/internal/snaptest"
)

func TestCPMRefundPaymentTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(CPMRefundPaymentRequest{}).NumField(); n != 10 {
		t.Errorf("CPMRefundPaymentRequest has %d fields, want 10", n)
	}
	if n := reflect.TypeOf(CPMRefundPaymentResponse{}).NumField(); n != 10 {
		t.Errorf("CPMRefundPaymentResponse has %d fields, want 10", n)
	}
}

func TestCPMRefundPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2008000",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"PARTNER001",
   "originalReferenceNo":"REF001",
   "originalExternalId":"EXT001",
   "refundNo":"REFUND001",
   "partnerRefundNo":"PARTNERREFUND001",
   "refundAmount":{"value":"1000.00","currency":"IDR"},
   "refundTime":"2026-09-13T10:00:00+07:00",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-refund"
	tr := &snap.Transport{}
	resp, err := CPMRefundPayment(context.Background(), tr, hb, CPMRefundPaymentRequest{
		OriginalPartnerReferenceNo: "PARTNER001",
		PartnerRefundNo:            "PARTNERREFUND001",
	})
	if err != nil {
		t.Fatalf("CPMRefundPayment() error = %v", err)
	}

	want := CPMRefundPaymentResponse{
		ResponseCode:               "2008000",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		OriginalExternalID:         "EXT001",
		RefundNo:                   "REFUND001",
		PartnerRefundNo:            "PARTNERREFUND001",
		RefundAmount:               &snap.Money{Value: "1000.00", Currency: "IDR"},
		RefundTime:                 "2026-09-13T10:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CPMRefundPayment() = %+v, want %+v", resp, want)
	}
}

func TestCPMRefundPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2008000","responseMessage":"ok","refundNo":"REFUND001","refundTime":"2026-09-13T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-refund"
	tr := &snap.Transport{}
	req := CPMRefundPaymentRequest{
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		ExternalStoreID:            "STORE01",
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		OriginalExternalID:         "EXT001",
		PartnerRefundNo:            "PARTNERREFUND001",
		RefundAmount:               &snap.Money{Value: "1000.00", Currency: "IDR"},
		Reason:                     "customer requested refund",
		AdditionalInfo:             json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := CPMRefundPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CPMRefundPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"merchantId":                 "MERCH01",
		"subMerchantId":              "SUBMERCH01",
		"externalStoreId":            "STORE01",
		"originalPartnerReferenceNo": "PARTNER001",
		"originalReferenceNo":        "REF001",
		"originalExternalId":         "EXT001",
		"partnerRefundNo":            "PARTNERREFUND001",
		"refundAmount":               map[string]any{"value": "1000.00", "currency": "IDR"},
		"reason":                     "customer requested refund",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestCPMRefundPayment_MandatoryFieldsAlwaysSerialized pins that
// OriginalPartnerReferenceNo and PartnerRefundNo — the request fields
// without omitempty — always serialize, even as "", and every other
// field is omitted when unset.
func TestCPMRefundPayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2008000","responseMessage":"ok","refundNo":"REFUND001","refundTime":"2026-09-13T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-refund"
	tr := &snap.Transport{}
	if _, err := CPMRefundPayment(context.Background(), tr, hb, CPMRefundPaymentRequest{}); err != nil {
		t.Fatalf("CPMRefundPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"originalPartnerReferenceNo": "", "partnerRefundNo": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestCPMRefundPaymentResponse_MandatoryFieldsAlwaysSerialized pins
// that RefundNo and RefundTime — the response fields without
// omitempty — always serialize from a zero-value response, and every
// other field is omitted when unset.
func TestCPMRefundPaymentResponse_MandatoryFieldsAlwaysSerialized(t *testing.T) {
	b, err := json.Marshal(CPMRefundPaymentResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{"responseCode": "", "responseMessage": "", "refundNo": "", "refundTime": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestCPMRefundPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4008000","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-refund"
	tr := &snap.Transport{}
	_, err := CPMRefundPayment(context.Background(), tr, hb, CPMRefundPaymentRequest{})
	if err == nil {
		t.Fatal("CPMRefundPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("CPMRefundPayment() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestCPMRefundPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2008000","responseMessage":"ok","refundNo":"REFUND001","refundTime":"2026-09-13T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-refund"
	tr := &snap.Transport{}
	resp, err := CPMRefundPayment(context.Background(), tr, hb, CPMRefundPaymentRequest{})
	if err == nil {
		t.Fatalf("CPMRefundPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("CPMRefundPayment() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestCPMRefundPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refundNo":"REFUND001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-refund"
	tr := &snap.Transport{}
	resp, err := CPMRefundPayment(context.Background(), tr, hb, CPMRefundPaymentRequest{})
	if err == nil {
		t.Fatalf("CPMRefundPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
