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

func TestAuthVoidQueryTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(AuthVoidQueryRequest{}).NumField(); n != 7 {
		t.Errorf("AuthVoidQueryRequest has %d fields, want 7", n)
	}
	if n := reflect.TypeOf(AuthVoidQueryResponse{}).NumField(); n != 10 {
		t.Errorf("AuthVoidQueryResponse has %d fields, want 10", n)
	}
}

func TestAuthVoidQueryRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "merchantId":"MERCHANT001",
   "subMerchantId":"SUBMERCHANT001",
   "voidNo":"VOID001",
   "partnerVoidNo":"PVOID001",
   "additionalInfo":{"note":"req-value"}
}`
	var got AuthVoidQueryRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthVoidQueryRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		SubMerchantID:              "SUBMERCHANT001",
		VoidNo:                     "VOID001",
		PartnerVoidNo:              "PVOID001",
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
		"voidNo":                     "VOID001",
		"partnerVoidNo":              "PVOID001",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestAuthVoidQueryRequest_MandatoryFieldsHaveNoOmitempty pins that
// OriginalReferenceNo, MerchantID, and PartnerVoidNo — the fields
// without omitempty — always serialize, even from a zero-value
// request.
func TestAuthVoidQueryRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthVoidQueryRequest{})
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
		"partnerVoidNo":       "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional and unset)", got, want)
	}
}

func TestAuthVoidQueryResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2006800",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "voidNo":"VOID001",
   "voidAmount":{"value":"25000.00","currency":"IDR"},
   "voidTime":"2026-09-13T10:00:00+07:00",
   "latestVoidStatus":"SUCCESS",
   "partnerVoidNo":"PVOID001",
   "additionalInfo":{"note":"resp-value"}
}`
	var got AuthVoidQueryResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthVoidQueryResponse{
		ResponseCode:               "2006800",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		VoidNo:                     "VOID001",
		VoidAmount:                 Money{Value: "25000.00", Currency: "IDR"},
		VoidTime:                   "2026-09-13T10:00:00+07:00",
		LatestVoidStatus:           "SUCCESS",
		PartnerVoidNo:              "PVOID001",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestAuthVoidQueryResponse_MandatoryFieldsHaveNoOmitempty pins
// ResponseCode, ResponseMessage, and VoidAmount (a plain struct,
// always present) as always-serializing; PartnerVoidNo is Optional
// here — unlike AuthVoidResponse's own PartnerVoidNo.
func TestAuthVoidQueryResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthVoidQueryResponse{})
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
		"voidAmount":      map[string]any{"value": "", "currency": ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestAuthVoidQuery_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := AuthVoidQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerVoidNo:       "PVOID001",
		AdditionalInfo:      json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}

func TestAuthVoidQuery_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2006800",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "voidNo":"VOID001",
   "voidAmount":{"value":"25000.00","currency":"IDR"},
   "voidTime":"2026-09-13T10:00:00+07:00",
   "latestVoidStatus":"SUCCESS",
   "partnerVoidNo":"PVOID001",
   "additionalInfo":{"note":"resp-value"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void-query"
	tr := &Transport{}
	resp, err := AuthVoidQuery(context.Background(), tr, hb, AuthVoidQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerVoidNo:       "PVOID001",
	})
	if err != nil {
		t.Fatalf("AuthVoidQuery() error = %v", err)
	}

	want := AuthVoidQueryResponse{
		ResponseCode:               "2006800",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		VoidNo:                     "VOID001",
		VoidAmount:                 Money{Value: "25000.00", Currency: "IDR"},
		VoidTime:                   "2026-09-13T10:00:00+07:00",
		LatestVoidStatus:           "SUCCESS",
		PartnerVoidNo:              "PVOID001",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AuthVoidQuery() = %+v, want %+v", resp, want)
	}
}

func TestAuthVoidQuery_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006800","responseMessage":"ok","voidAmount":{"value":"","currency":""}}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void-query"
	tr := &Transport{}
	req := AuthVoidQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerVoidNo:       "PVOID001",
	}
	if _, err := AuthVoidQuery(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AuthVoidQuery() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerVoidNo"] != "PVOID001" {
		t.Errorf(`wire body["partnerVoidNo"] = %v, want "PVOID001"`, got["partnerVoidNo"])
	}
}

func TestAuthVoidQuery_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4006800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void-query"
	tr := &Transport{}
	_, err := AuthVoidQuery(context.Background(), tr, hb, AuthVoidQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerVoidNo:       "PVOID001",
	})
	if err == nil {
		t.Fatal("AuthVoidQuery() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("AuthVoidQuery() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestAuthVoidQuery_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2006800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void-query"
	tr := &Transport{}
	resp, err := AuthVoidQuery(context.Background(), tr, hb, AuthVoidQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerVoidNo:       "PVOID001",
	})
	if err == nil {
		t.Fatalf("AuthVoidQuery() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("AuthVoidQuery() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestAuthVoidQuery_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"voidNo":"VOID001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void-query"
	tr := &Transport{}
	resp, err := AuthVoidQuery(context.Background(), tr, hb, AuthVoidQueryRequest{
		OriginalReferenceNo: "ref-1",
		MerchantID:          "MERCHANT001",
		PartnerVoidNo:       "PVOID001",
	})
	if err == nil {
		t.Fatalf("AuthVoidQuery() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
