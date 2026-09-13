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

func TestAuthCaptureQueryTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(AuthCaptureQueryRequest{}).NumField(); n != 7 {
		t.Errorf("AuthCaptureQueryRequest has %d fields, want 7", n)
	}
	if n := reflect.TypeOf(AuthCaptureQueryResponse{}).NumField(); n != 10 {
		t.Errorf("AuthCaptureQueryResponse has %d fields, want 10", n)
	}
}

func TestAuthCaptureQueryRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "merchantId":"MERCHANT001",
   "subMerchantId":"SUBMERCHANT001",
   "captureNo":"CAP001",
   "partnerCaptureNo":"PCAP001",
   "additionalInfo":{"note":"req-value"}
}`
	var got AuthCaptureQueryRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthCaptureQueryRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		SubMerchantID:              "SUBMERCHANT001",
		CaptureNo:                  "CAP001",
		PartnerCaptureNo:           "PCAP001",
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
		"captureNo":                  "CAP001",
		"partnerCaptureNo":           "PCAP001",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestAuthCaptureQueryRequest_MandatoryFieldsHaveNoOmitempty pins that
// OriginalReferenceNo, MerchantID, and PartnerCaptureNo — the fields
// without omitempty — always serialize, even from a zero-value
// request.
func TestAuthCaptureQueryRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthCaptureQueryRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{
		"originalReferenceNo": "",
		"merchantId":          "",
		"partnerCaptureNo":    "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional and unset)", got, want)
	}
}

func TestAuthCaptureQueryResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2006600",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "captureNo":"CAP001",
   "captureAmount":{"value":"25000.00","currency":"IDR"},
   "captureTime":"2026-09-13T10:00:00+07:00",
   "latestCaptureStatus":"SUCCESS",
   "partnerCaptureNo":"PCAP001",
   "additionalInfo":{"note":"resp-value"}
}`
	var got AuthCaptureQueryResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthCaptureQueryResponse{
		ResponseCode:               "2006600",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		CaptureNo:                  "CAP001",
		CaptureAmount:              Money{Value: "25000.00", Currency: "IDR"},
		CaptureTime:                "2026-09-13T10:00:00+07:00",
		LatestCaptureStatus:        "SUCCESS",
		PartnerCaptureNo:           "PCAP001",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestAuthCaptureQueryResponse_MandatoryFieldsHaveNoOmitempty pins
// ResponseCode, ResponseMessage, CaptureAmount (a plain struct, always
// present), and PartnerCaptureNo as always-serializing.
func TestAuthCaptureQueryResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthCaptureQueryResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{
		"responseCode":     "",
		"responseMessage":  "",
		"captureAmount":    map[string]any{"value": "", "currency": ""},
		"partnerCaptureNo": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestAuthCaptureQuery_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := AuthCaptureQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerCaptureNo:    "PCAP001",
		AdditionalInfo:      json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}

func TestAuthCaptureQuery_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2006600",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "captureNo":"CAP001",
   "captureAmount":{"value":"25000.00","currency":"IDR"},
   "captureTime":"2026-09-13T10:00:00+07:00",
   "latestCaptureStatus":"SUCCESS",
   "partnerCaptureNo":"PCAP001",
   "additionalInfo":{"note":"resp-value"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture-query"
	tr := &Transport{}
	resp, err := AuthCaptureQuery(context.Background(), tr, hb, AuthCaptureQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerCaptureNo:    "PCAP001",
	})
	if err != nil {
		t.Fatalf("AuthCaptureQuery() error = %v", err)
	}

	want := AuthCaptureQueryResponse{
		ResponseCode:               "2006600",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		CaptureNo:                  "CAP001",
		CaptureAmount:              Money{Value: "25000.00", Currency: "IDR"},
		CaptureTime:                "2026-09-13T10:00:00+07:00",
		LatestCaptureStatus:        "SUCCESS",
		PartnerCaptureNo:           "PCAP001",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AuthCaptureQuery() = %+v, want %+v", resp, want)
	}
}

func TestAuthCaptureQuery_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006600","responseMessage":"ok","captureAmount":{"value":"","currency":""},"partnerCaptureNo":"PCAP001"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture-query"
	tr := &Transport{}
	req := AuthCaptureQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerCaptureNo:    "PCAP001",
	}
	if _, err := AuthCaptureQuery(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AuthCaptureQuery() error = %v", err)
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
}

func TestAuthCaptureQuery_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4006600","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture-query"
	tr := &Transport{}
	_, err := AuthCaptureQuery(context.Background(), tr, hb, AuthCaptureQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerCaptureNo:    "PCAP001",
	})
	if err == nil {
		t.Fatal("AuthCaptureQuery() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("AuthCaptureQuery() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestAuthCaptureQuery_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2006600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture-query"
	tr := &Transport{}
	resp, err := AuthCaptureQuery(context.Background(), tr, hb, AuthCaptureQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerCaptureNo:    "PCAP001",
	})
	if err == nil {
		t.Fatalf("AuthCaptureQuery() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("AuthCaptureQuery() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestAuthCaptureQuery_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"captureNo":"CAP001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/capture-query"
	tr := &Transport{}
	resp, err := AuthCaptureQuery(context.Background(), tr, hb, AuthCaptureQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerCaptureNo:    "PCAP001",
	})
	if err == nil {
		t.Fatalf("AuthCaptureQuery() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
