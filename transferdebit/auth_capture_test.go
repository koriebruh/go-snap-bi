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

func TestAuthCaptureTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(AuthCaptureRequest{}).NumField(); n != 9 {
		t.Errorf("AuthCaptureRequest has %d fields, want 9", n)
	}
	if n := reflect.TypeOf(AuthCaptureResponse{}).NumField(); n != 9 {
		t.Errorf("AuthCaptureResponse has %d fields, want 9", n)
	}
}

func TestAuthCaptureRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "merchantId":"MERCHANT001",
   "subMerchantId":"SUBMERCHANT001",
   "partnerCaptureNo":"PCAP001",
   "captureAmount":{"value":"25000.00","currency":"IDR"},
   "title":"Order #123 partial capture",
   "lastCapture":"TRUE",
   "additionalInfo":{"note":"req-value"}
}`
	var got AuthCaptureRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthCaptureRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		SubMerchantID:              "SUBMERCHANT001",
		PartnerCaptureNo:           "PCAP001",
		CaptureAmount:              &snap.Money{Value: "25000.00", Currency: "IDR"},
		Title:                      "Order #123 partial capture",
		LastCapture:                "TRUE",
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
		"originalReferenceNo":        "ref-1",
		"originalPartnerReferenceNo": "partner-ref-1",
		"merchantId":                 "MERCHANT001",
		"subMerchantId":              "SUBMERCHANT001",
		"partnerCaptureNo":           "PCAP001",
		"captureAmount":              map[string]any{"value": "25000.00", "currency": "IDR"},
		"title":                      "Order #123 partial capture",
		"lastCapture":                "TRUE",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestAuthCaptureRequest_MandatoryFieldsHaveNoOmitempty pins that
// OriginalReferenceNo, OriginalPartnerReferenceNo, MerchantID,
// PartnerCaptureNo, and Title — the fields without omitempty — always
// serialize, even from a zero-value request.
func TestAuthCaptureRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthCaptureRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{
		"originalReferenceNo":        "",
		"originalPartnerReferenceNo": "",
		"merchantId":                 "",
		"partnerCaptureNo":           "",
		"title":                      "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional and unset)", got, want)
	}
}

func TestAuthCaptureResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2006500",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "partnerCaptureNo":"PCAP001",
   "captureNo":"CAP001",
   "captureAmount":{"value":"25000.00","currency":"IDR"},
   "captureTime":"2026-09-13T10:00:00+07:00",
   "additionalInfo":{"note":"resp-value"}
}`
	var got AuthCaptureResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthCaptureResponse{
		ResponseCode:               "2006500",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerCaptureNo:           "PCAP001",
		CaptureNo:                  "CAP001",
		CaptureAmount:              snap.Money{Value: "25000.00", Currency: "IDR"},
		CaptureTime:                "2026-09-13T10:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestAuthCaptureResponse_MandatoryFieldsHaveNoOmitempty pins
// ResponseCode, ResponseMessage, and CaptureAmount (a plain struct,
// always present) as always-serializing.
func TestAuthCaptureResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthCaptureResponse{})
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
		"captureAmount":   map[string]any{"value": "", "currency": ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestAuthCapture_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := AuthCaptureRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerCaptureNo:           "PCAP001",
		Title:                      "Order #123",
		AdditionalInfo:             json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}

func TestAuthCapture_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2006500",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "partnerCaptureNo":"PCAP001",
   "captureNo":"CAP001",
   "captureAmount":{"value":"25000.00","currency":"IDR"},
   "captureTime":"2026-09-13T10:00:00+07:00",
   "additionalInfo":{"note":"resp-value"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture"
	tr := &snap.Transport{}
	resp, err := AuthCapture(context.Background(), tr, hb, AuthCaptureRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerCaptureNo:           "PCAP001",
		Title:                      "Order #123",
	})
	if err != nil {
		t.Fatalf("AuthCapture() error = %v", err)
	}

	want := AuthCaptureResponse{
		ResponseCode:               "2006500",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		PartnerCaptureNo:           "PCAP001",
		CaptureNo:                  "CAP001",
		CaptureAmount:              snap.Money{Value: "25000.00", Currency: "IDR"},
		CaptureTime:                "2026-09-13T10:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AuthCapture() = %+v, want %+v", resp, want)
	}
}

func TestAuthCapture_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006500","responseMessage":"ok","captureAmount":{"value":"","currency":""}}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture"
	tr := &snap.Transport{}
	req := AuthCaptureRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerCaptureNo:           "PCAP001",
		Title:                      "Order #123",
		CaptureAmount:              &snap.Money{Value: "25000.00", Currency: "IDR"},
	}
	if _, err := AuthCapture(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AuthCapture() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerCaptureNo"] != "PCAP001" {
		t.Errorf(`wire body["partnerCaptureNo"] = %v, want "PCAP001"`, got["partnerCaptureNo"])
	}
	amount, ok := got["captureAmount"].(map[string]any)
	if !ok {
		t.Fatalf(`wire body["captureAmount"] = %v, want an object`, got["captureAmount"])
	}
	if amount["value"] != "25000.00" {
		t.Errorf(`wire body["captureAmount"]["value"] = %v, want "25000.00"`, amount["value"])
	}
}

func TestAuthCapture_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4006500","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture"
	tr := &snap.Transport{}
	_, err := AuthCapture(context.Background(), tr, hb, AuthCaptureRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerCaptureNo:           "PCAP001",
		Title:                      "Order #123",
	})
	if err == nil {
		t.Fatal("AuthCapture() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("AuthCapture() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestAuthCapture_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2006500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture"
	tr := &snap.Transport{}
	resp, err := AuthCapture(context.Background(), tr, hb, AuthCaptureRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerCaptureNo:           "PCAP001",
		Title:                      "Order #123",
	})
	if err == nil {
		t.Fatalf("AuthCapture() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("AuthCapture() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestAuthCapture_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"captureNo":"CAP001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture"
	tr := &snap.Transport{}
	resp, err := AuthCapture(context.Background(), tr, hb, AuthCaptureRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerCaptureNo:           "PCAP001",
		Title:                      "Order #123",
	})
	if err == nil {
		t.Fatalf("AuthCapture() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
