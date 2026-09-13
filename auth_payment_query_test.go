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

func TestAuthPaymentQueryTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(AuthPaymentQueryRequest{}).NumField(); n != 6 {
		t.Errorf("AuthPaymentQueryRequest has %d fields, want 6", n)
	}
	if n := reflect.TypeOf(AuthPaymentQueryResponse{}).NumField(); n != 9 {
		t.Errorf("AuthPaymentQueryResponse has %d fields, want 9", n)
	}
}

func TestAuthPaymentQueryRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "merchantId":"MERCHANT001",
   "subMerchantId":"SUBMERCHANT001",
   "externalStoreId":"STORE001",
   "additionalInfo":{"note":"req-value"}
}`
	var got AuthPaymentQueryRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthPaymentQueryRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		MerchantID:                 "MERCHANT001",
		SubMerchantID:              "SUBMERCHANT001",
		ExternalStoreID:            "STORE001",
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
		"merchantId":                 "MERCHANT001",
		"subMerchantId":              "SUBMERCHANT001",
		"externalStoreId":            "STORE001",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestAuthPaymentQueryRequest_ZeroValueMarshalsEmpty pins that a
// zero-value request has no Mandatory fields — every key is omitted.
func TestAuthPaymentQueryRequest_ZeroValueMarshalsEmpty(t *testing.T) {
	b, err := json.Marshal(AuthPaymentQueryRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("marshaled zero-value request = %v, want empty object (no Mandatory field)", got)
	}
}

func TestAuthPaymentQueryResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2006400",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "amount":{"value":"50000.00","currency":"IDR"},
   "paidTime":"2026-09-13T10:00:00+07:00",
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "additionalInfo":{"note":"resp-value"}
}`
	var got AuthPaymentQueryResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthPaymentQueryResponse{
		ResponseCode:               "2006400",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		Amount:                     &Money{Value: "50000.00", Currency: "IDR"},
		PaidTime:                   "2026-09-13T10:00:00+07:00",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestAuthPaymentQueryResponse_MandatoryFieldsHaveNoOmitempty pins
// ResponseCode, ResponseMessage, PaidTime, and LatestTransactionStatus
// as always-serializing; Amount is Optional (*Money, omitempty) and
// must be absent from a zero-value response.
func TestAuthPaymentQueryResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthPaymentQueryResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{
		"responseCode":            "",
		"responseMessage":         "",
		"paidTime":                "",
		"latestTransactionStatus": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

// TestAuthPaymentQueryResponse_LowercasePCasingUnmarshalsCorrectly
// pins the resolution of research §6 item 2: the worked example's
// originalpartnerReferenceNo (lowercase p) casing still populates
// OriginalPartnerReferenceNo via Go's case-insensitive JSON decode.
func TestAuthPaymentQueryResponse_LowercasePCasingUnmarshalsCorrectly(t *testing.T) {
	const fixture = `{
   "responseCode":"2006400",
   "responseMessage":"Request has been processed successfully",
   "originalpartnerReferenceNo":"partner-ref-1",
   "amount":{"value":"50000.00","currency":"IDR"},
   "paidTime":"2026-09-13T10:00:00+07:00",
   "latestTransactionStatus":"00"
}`
	var got AuthPaymentQueryResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.OriginalPartnerReferenceNo != "partner-ref-1" {
		t.Errorf("OriginalPartnerReferenceNo = %q, want %q (lowercase-p wire casing must still decode)", got.OriginalPartnerReferenceNo, "partner-ref-1")
	}
}

func TestAuthPaymentQuery_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := AuthPaymentQueryRequest{
		AdditionalInfo: json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}

func TestAuthPaymentQuery_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2006400",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "amount":{"value":"50000.00","currency":"IDR"},
   "paidTime":"2026-09-13T10:00:00+07:00",
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "additionalInfo":{"note":"resp-value"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/query"
	tr := &Transport{}
	resp, err := AuthPaymentQuery(context.Background(), tr, hb, AuthPaymentQueryRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
	})
	if err != nil {
		t.Fatalf("AuthPaymentQuery() error = %v", err)
	}

	want := AuthPaymentQueryResponse{
		ResponseCode:               "2006400",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		Amount:                     &Money{Value: "50000.00", Currency: "IDR"},
		PaidTime:                   "2026-09-13T10:00:00+07:00",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AuthPaymentQuery() = %+v, want %+v", resp, want)
	}
}

func TestAuthPaymentQuery_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006400","responseMessage":"ok","amount":{"value":"","currency":""},"paidTime":"","latestTransactionStatus":""}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/query"
	tr := &Transport{}
	req := AuthPaymentQueryRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
	}
	if _, err := AuthPaymentQuery(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AuthPaymentQuery() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["originalPartnerReferenceNo"] != "partner-ref-1" {
		t.Errorf(`wire body["originalPartnerReferenceNo"] = %v, want "partner-ref-1"`, got["originalPartnerReferenceNo"])
	}
	if got["merchantId"] != "MERCHANT001" {
		t.Errorf(`wire body["merchantId"] = %v, want "MERCHANT001"`, got["merchantId"])
	}
}

func TestAuthPaymentQuery_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4006400","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/query"
	tr := &Transport{}
	_, err := AuthPaymentQuery(context.Background(), tr, hb, AuthPaymentQueryRequest{})
	if err == nil {
		t.Fatal("AuthPaymentQuery() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("AuthPaymentQuery() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestAuthPaymentQuery_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2006400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/query"
	tr := &Transport{}
	resp, err := AuthPaymentQuery(context.Background(), tr, hb, AuthPaymentQueryRequest{})
	if err == nil {
		t.Fatalf("AuthPaymentQuery() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("AuthPaymentQuery() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestAuthPaymentQuery_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"originalReferenceNo":"ref-1"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/query"
	tr := &Transport{}
	resp, err := AuthPaymentQuery(context.Background(), tr, hb, AuthPaymentQueryRequest{})
	if err == nil {
		t.Fatalf("AuthPaymentQuery() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
