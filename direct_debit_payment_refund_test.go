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

func TestDirectDebitPaymentRefundTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(DirectDebitPaymentRefundRequest{}).NumField(); n != 10 {
		t.Errorf("DirectDebitPaymentRefundRequest has %d fields, want 10", n)
	}
	if n := reflect.TypeOf(DirectDebitPaymentRefundResponse{}).NumField(); n != 11 {
		t.Errorf("DirectDebitPaymentRefundResponse has %d fields, want 11", n)
	}
}

func TestDirectDebitPaymentRefund_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2005800",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"PARTNER001",
   "originalReferenceNo":"REF001",
   "originalExternalId":"EXT001",
   "partnerTrxId":"TRX001",
   "refundNo":"REFUND001",
   "partnerRefundNo":"PARTNERREFUND001",
   "refundAmount":{"value":"1000.00","currency":"IDR"},
   "refundTime":"2026-09-12T10:00:00+07:00",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/refund"
	tr := &Transport{}
	resp, err := DirectDebitPaymentRefund(context.Background(), tr, hb, DirectDebitPaymentRefundRequest{
		OriginalPartnerReferenceNo: "PARTNER001",
		PartnerRefundNo:            "PARTNERREFUND001",
	})
	if err != nil {
		t.Fatalf("DirectDebitPaymentRefund() error = %v", err)
	}

	want := DirectDebitPaymentRefundResponse{
		ResponseCode:               "2005800",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		OriginalExternalID:         "EXT001",
		PartnerTrxID:               "TRX001",
		RefundNo:                   "REFUND001",
		PartnerRefundNo:            "PARTNERREFUND001",
		RefundAmount:               &Money{Value: "1000.00", Currency: "IDR"},
		RefundTime:                 "2026-09-12T10:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("DirectDebitPaymentRefund() = %+v, want %+v", resp, want)
	}
}

func TestDirectDebitPaymentRefund_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005800","responseMessage":"ok","refundNo":"REFUND001","partnerRefundNo":"PARTNERREFUND001","refundTime":"2026-09-12T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/refund"
	tr := &Transport{}
	req := DirectDebitPaymentRefundRequest{
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		OriginalExternalID:         "EXT001",
		PartnerRefundNo:            "PARTNERREFUND001",
		RefundAmount:               &Money{Value: "1000.00", Currency: "IDR"},
		ExternalStoreID:            "STORE01",
		Reason:                     "customer requested refund",
		AdditionalInfo:             json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := DirectDebitPaymentRefund(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("DirectDebitPaymentRefund() error = %v", err)
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
		"originalPartnerReferenceNo": "PARTNER001",
		"originalReferenceNo":        "REF001",
		"originalExternalId":         "EXT001",
		"partnerRefundNo":            "PARTNERREFUND001",
		"refundAmount":               map[string]any{"value": "1000.00", "currency": "IDR"},
		"externalStoreId":            "STORE01",
		"reason":                     "customer requested refund",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestDirectDebitPaymentRefund_MandatoryFieldsAlwaysSerialized pins
// that OriginalPartnerReferenceNo and PartnerRefundNo — the request
// fields without omitempty — always serialize, even as "", and every
// other field is omitted when unset.
func TestDirectDebitPaymentRefund_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005800","responseMessage":"ok","refundNo":"REFUND001","partnerRefundNo":"PARTNERREFUND001","refundTime":"2026-09-12T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/refund"
	tr := &Transport{}
	if _, err := DirectDebitPaymentRefund(context.Background(), tr, hb, DirectDebitPaymentRefundRequest{}); err != nil {
		t.Fatalf("DirectDebitPaymentRefund() error = %v", err)
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

// TestDirectDebitPaymentRefundResponse_MandatoryFieldsAlwaysSerialized
// pins that RefundNo, PartnerRefundNo, and RefundTime — the response
// fields without omitempty — always serialize from a zero-value
// response, and every other field is omitted when unset.
func TestDirectDebitPaymentRefundResponse_MandatoryFieldsAlwaysSerialized(t *testing.T) {
	b, err := json.Marshal(DirectDebitPaymentRefundResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{"responseCode": "", "responseMessage": "", "refundNo": "", "partnerRefundNo": "", "refundTime": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestDirectDebitPaymentRefund_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4005800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/refund"
	tr := &Transport{}
	_, err := DirectDebitPaymentRefund(context.Background(), tr, hb, DirectDebitPaymentRefundRequest{})
	if err == nil {
		t.Fatal("DirectDebitPaymentRefund() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("DirectDebitPaymentRefund() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestDirectDebitPaymentRefund_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2005800","responseMessage":"ok","refundNo":"REFUND001","partnerRefundNo":"PARTNERREFUND001","refundTime":"2026-09-12T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/refund"
	tr := &Transport{}
	resp, err := DirectDebitPaymentRefund(context.Background(), tr, hb, DirectDebitPaymentRefundRequest{})
	if err == nil {
		t.Fatalf("DirectDebitPaymentRefund() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("DirectDebitPaymentRefund() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestDirectDebitPaymentRefund_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refundNo":"REFUND001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/refund"
	tr := &Transport{}
	resp, err := DirectDebitPaymentRefund(context.Background(), tr, hb, DirectDebitPaymentRefundRequest{})
	if err == nil {
		t.Fatalf("DirectDebitPaymentRefund() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
