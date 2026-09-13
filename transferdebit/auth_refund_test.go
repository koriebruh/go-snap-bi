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

func TestAuthRefundTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(AuthRefundRequest{}).NumField(); n != 10 {
		t.Errorf("AuthRefundRequest has %d fields, want 10", n)
	}
	if n := reflect.TypeOf(AuthRefundResponse{}).NumField(); n != 10 {
		t.Errorf("AuthRefundResponse has %d fields, want 10", n)
	}
}

func TestAuthRefundRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "partnerRefundNo":"PREF001",
   "merchantId":"MERCHANT001",
   "subMerchantId":"SUBMERCHANT001",
   "originalCaptureNo":"CAP001",
   "refundAmount":{"value":"10000.00","currency":"IDR"},
   "externalStoreId":"STORE001",
   "reason":"customer requested refund",
   "additionalInfo":{"note":"req-value"}
}`
	var got AuthRefundRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthRefundRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		PartnerRefundNo:            "PREF001",
		MerchantID:                 "MERCHANT001",
		SubMerchantID:              "SUBMERCHANT001",
		OriginalCaptureNo:          "CAP001",
		RefundAmount:               &snap.Money{Value: "10000.00", Currency: "IDR"},
		ExternalStoreID:            "STORE001",
		Reason:                     "customer requested refund",
		AdditionalInfo:             json.RawMessage(`{"note":"req-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}

	b, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatalf("decode marshaled request: %v", err)
	}
	wantWire := map[string]any{
		"originalPartnerReferenceNo": "partner-ref-1",
		"originalReferenceNo":        "ref-1",
		"partnerRefundNo":            "PREF001",
		"merchantId":                 "MERCHANT001",
		"subMerchantId":              "SUBMERCHANT001",
		"originalCaptureNo":          "CAP001",
		"refundAmount":               map[string]any{"value": "10000.00", "currency": "IDR"},
		"externalStoreId":            "STORE001",
		"reason":                     "customer requested refund",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestAuthRefundRequest_MandatoryFieldsHaveNoOmitempty pins that
// OriginalPartnerReferenceNo and PartnerRefundNo — the fields without
// omitempty — always serialize, even from a zero-value request.
func TestAuthRefundRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthRefundRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{
		"originalPartnerReferenceNo": "",
		"partnerRefundNo":            "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional/Conditional and unset)", got, want)
	}
}

func TestAuthRefundResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2006900",
   "responseMessage":"Request has been processed successfully",
   "originalCaptureNo":"CAP001",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "partnerRefundNo":"PREF001",
   "refundNo":"REF001",
   "refundAmount":{"value":"10000.00","currency":"IDR"},
   "refundTime":"2026-09-13T10:00:00+07:00",
   "additionalInfo":{"note":"resp-value"}
}`
	var got AuthRefundResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthRefundResponse{
		ResponseCode:               "2006900",
		ResponseMessage:            "Request has been processed successfully",
		OriginalCaptureNo:          "CAP001",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerRefundNo:            "PREF001",
		RefundNo:                   "REF001",
		RefundAmount:               &snap.Money{Value: "10000.00", Currency: "IDR"},
		RefundTime:                 "2026-09-13T10:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestAuthRefundResponse_MandatoryFieldsHaveNoOmitempty pins
// ResponseCode, ResponseMessage, RefundNo, and RefundTime as
// always-serializing.
func TestAuthRefundResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthRefundResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{
		"responseCode":    "",
		"responseMessage": "",
		"refundNo":        "",
		"refundTime":      "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestAuthRefund_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := AuthRefundRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerRefundNo:            "PREF001",
		AdditionalInfo:             json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}

func TestAuthRefund_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2006900",
   "responseMessage":"Request has been processed successfully",
   "originalCaptureNo":"CAP001",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "partnerRefundNo":"PREF001",
   "refundNo":"REF001",
   "refundAmount":{"value":"10000.00","currency":"IDR"},
   "refundTime":"2026-09-13T10:00:00+07:00",
   "additionalInfo":{"note":"resp-value"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/refund"
	tr := &snap.Transport{}
	resp, err := AuthRefund(context.Background(), tr, hb, AuthRefundRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerRefundNo:            "PREF001",
	})
	if err != nil {
		t.Fatalf("AuthRefund() error = %v", err)
	}

	want := AuthRefundResponse{
		ResponseCode:               "2006900",
		ResponseMessage:            "Request has been processed successfully",
		OriginalCaptureNo:          "CAP001",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerRefundNo:            "PREF001",
		RefundNo:                   "REF001",
		RefundAmount:               &snap.Money{Value: "10000.00", Currency: "IDR"},
		RefundTime:                 "2026-09-13T10:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AuthRefund() = %+v, want %+v", resp, want)
	}
}

func TestAuthRefund_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006900","responseMessage":"ok","refundNo":"REF001","refundTime":"2026-09-13T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/refund"
	tr := &snap.Transport{}
	req := AuthRefundRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerRefundNo:            "PREF001",
		RefundAmount:               &snap.Money{Value: "10000.00", Currency: "IDR"},
	}
	if _, err := AuthRefund(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AuthRefund() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerRefundNo"] != "PREF001" {
		t.Errorf(`wire body["partnerRefundNo"] = %v, want "PREF001"`, got["partnerRefundNo"])
	}
	amount, ok := got["refundAmount"].(map[string]any)
	if !ok {
		t.Fatalf(`wire body["refundAmount"] = %v, want an object`, got["refundAmount"])
	}
	if amount["value"] != "10000.00" {
		t.Errorf(`wire body["refundAmount"]["value"] = %v, want "10000.00"`, amount["value"])
	}
}

func TestAuthRefund_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4006900","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/refund"
	tr := &snap.Transport{}
	_, err := AuthRefund(context.Background(), tr, hb, AuthRefundRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerRefundNo:            "PREF001",
	})
	if err == nil {
		t.Fatal("AuthRefund() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("AuthRefund() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestAuthRefund_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2006900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/refund"
	tr := &snap.Transport{}
	resp, err := AuthRefund(context.Background(), tr, hb, AuthRefundRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerRefundNo:            "PREF001",
	})
	if err == nil {
		t.Fatalf("AuthRefund() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("AuthRefund() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestAuthRefund_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refundNo":"REF001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/refund"
	tr := &snap.Transport{}
	resp, err := AuthRefund(context.Background(), tr, hb, AuthRefundRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerRefundNo:            "PREF001",
	})
	if err == nil {
		t.Fatalf("AuthRefund() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
