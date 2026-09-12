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

// TestQRMPMRefundPaymentTypes_FieldCounts guards against a field
// silently added to either type without updating the wire-assertion
// tests above — those tests catch renamed/omitempty-flipped fields but
// not an addition, since a new field defaults to unset and unasserted.
func TestQRMPMRefundPaymentTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(QRMPMRefundPaymentRequest{}).NumField(); n != 9 {
		t.Errorf("QRMPMRefundPaymentRequest has %d fields, want 9", n)
	}
	if n := reflect.TypeOf(QRMPMRefundPaymentResponse{}).NumField(); n != 6 {
		t.Errorf("QRMPMRefundPaymentResponse has %d fields, want 6", n)
	}
}

func TestQRMPMRefundPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2007800",
   "responseMessage":"Request has been processed successfully",
   "refundNo":"REFUND000001",
   "partnerRefundNo":"partner-refund-1",
   "refundAmount":{"value":"10000.00","currency":"IDR"},
   "refundTime":"2020-12-20T10:00:00+07:00"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-refund"
	tr := &Transport{}
	resp, err := QRMPMRefundPayment(context.Background(), tr, hb, QRMPMRefundPaymentRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerRefundNo:            "partner-refund-1",
	})
	if err != nil {
		t.Fatalf("QRMPMRefundPayment() error = %v", err)
	}

	want := QRMPMRefundPaymentResponse{
		ResponseCode:    "2007800",
		ResponseMessage: "Request has been processed successfully",
		RefundNo:        "REFUND000001",
		PartnerRefundNo: "partner-refund-1",
		RefundAmount:    &Money{Value: "10000.00", Currency: "IDR"},
		RefundTime:      "2020-12-20T10:00:00+07:00",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("QRMPMRefundPayment() = %+v, want %+v", resp, want)
	}
}

func TestQRMPMRefundPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2007800","responseMessage":"ok","refundNo":"REFUND000001","refundTime":"2020-12-20T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-refund"
	tr := &Transport{}
	req := QRMPMRefundPaymentRequest{
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		ExternalStoreID:            "STORE01",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		PartnerRefundNo:            "partner-refund-1",
		RefundAmount:               &Money{Value: "10000.00", Currency: "IDR"},
		Reason:                     "customer requested refund",
	}
	if _, err := QRMPMRefundPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("QRMPMRefundPayment() error = %v", err)
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
		"originalPartnerReferenceNo": "partner-ref-1",
		"originalReferenceNo":        "ref-1",
		"originalExternalId":         "ext-1",
		"partnerRefundNo":            "partner-refund-1",
		"refundAmount":               map[string]any{"value": "10000.00", "currency": "IDR"},
		"reason":                     "customer requested refund",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestQRMPMRefundPayment_MandatoryFieldsAlwaysSerialized pins that
// OriginalPartnerReferenceNo and PartnerRefundNo — the request fields
// without omitempty — are always present on the wire, even as "", and
// every other field (all Optional) is omitted when unset.
func TestQRMPMRefundPayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2007800","responseMessage":"ok","refundNo":"REFUND000001","refundTime":"2020-12-20T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-refund"
	tr := &Transport{}
	if _, err := QRMPMRefundPayment(context.Background(), tr, hb, QRMPMRefundPaymentRequest{}); err != nil {
		t.Fatalf("QRMPMRefundPayment() error = %v", err)
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

// TestQRMPMRefundPaymentResponse_MandatoryFieldsAlwaysSerialized pins
// that RefundNo and RefundTime — the response fields without
// omitempty — are always present on a zero-value marshal, and every
// other field is omitted.
func TestQRMPMRefundPaymentResponse_MandatoryFieldsAlwaysSerialized(t *testing.T) {
	b, err := json.Marshal(QRMPMRefundPaymentResponse{})
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

func TestQRMPMRefundPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4007800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-refund"
	tr := &Transport{}
	_, err := QRMPMRefundPayment(context.Background(), tr, hb, QRMPMRefundPaymentRequest{})
	if err == nil {
		t.Fatal("QRMPMRefundPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("QRMPMRefundPayment() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestQRMPMRefundPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2007800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-refund"
	tr := &Transport{}
	resp, err := QRMPMRefundPayment(context.Background(), tr, hb, QRMPMRefundPaymentRequest{})
	if err == nil {
		t.Fatalf("QRMPMRefundPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("QRMPMRefundPayment() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestQRMPMRefundPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refundNo":"REFUND000001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-refund"
	tr := &Transport{}
	resp, err := QRMPMRefundPayment(context.Background(), tr, hb, QRMPMRefundPaymentRequest{})
	if err == nil {
		t.Fatalf("QRMPMRefundPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
