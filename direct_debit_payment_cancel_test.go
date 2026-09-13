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

func TestDirectDebitPaymentCancelTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(DirectDebitPaymentCancelRequest{}).NumField(); n != 10 {
		t.Errorf("DirectDebitPaymentCancelRequest has %d fields, want 10", n)
	}
	if n := reflect.TypeOf(DirectDebitPaymentCancelResponse{}).NumField(); n != 8 {
		t.Errorf("DirectDebitPaymentCancelResponse has %d fields, want 8", n)
	}
}

func TestDirectDebitPaymentCancel_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2005700",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"PARTNER001",
   "originalReferenceNo":"REF001",
   "originalExternalId":"EXT001",
   "cancelTime":"2026-09-12T10:00:00+07:00",
   "transactionDate":"2026-09-12T09:00:00+07:00",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/cancel"
	tr := &Transport{}
	resp, err := DirectDebitPaymentCancel(context.Background(), tr, hb, DirectDebitPaymentCancelRequest{
		OriginalPartnerReferenceNo: "PARTNER001",
	})
	if err != nil {
		t.Fatalf("DirectDebitPaymentCancel() error = %v", err)
	}

	want := DirectDebitPaymentCancelResponse{
		ResponseCode:               "2005700",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		OriginalExternalID:         "EXT001",
		CancelTime:                 "2026-09-12T10:00:00+07:00",
		TransactionDate:            "2026-09-12T09:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("DirectDebitPaymentCancel() = %+v, want %+v", resp, want)
	}
}

func TestDirectDebitPaymentCancel_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/cancel"
	tr := &Transport{}
	req := DirectDebitPaymentCancelRequest{
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		ApprovalCode:               "APPROVAL001",
		OriginalExternalID:         "EXT001",
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		Reason:                     "customer requested cancellation",
		ExternalStoreID:            "STORE01",
		Amount:                     &Money{Value: "50000.00", Currency: "IDR"},
		AdditionalInfo:             json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := DirectDebitPaymentCancel(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("DirectDebitPaymentCancel() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"originalPartnerReferenceNo": "PARTNER001",
		"originalReferenceNo":        "REF001",
		"approvalCode":               "APPROVAL001",
		"originalExternalId":         "EXT001",
		"merchantId":                 "MERCH01",
		"subMerchantId":              "SUBMERCH01",
		"reason":                     "customer requested cancellation",
		"externalStoreId":            "STORE01",
		"amount":                     map[string]any{"value": "50000.00", "currency": "IDR"},
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestDirectDebitPaymentCancel_MandatoryFieldAlwaysSerialized pins that
// OriginalPartnerReferenceNo — the only request field without
// omitempty — is always present on the wire, even as "", and every
// other field is omitted when unset.
func TestDirectDebitPaymentCancel_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/cancel"
	tr := &Transport{}
	if _, err := DirectDebitPaymentCancel(context.Background(), tr, hb, DirectDebitPaymentCancelRequest{}); err != nil {
		t.Fatalf("DirectDebitPaymentCancel() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"originalPartnerReferenceNo": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestDirectDebitPaymentCancelResponse_ZeroValueOmitsOptionalFields
// pins that a zero-value response marshals to just the two envelope
// fields.
func TestDirectDebitPaymentCancelResponse_ZeroValueOmitsOptionalFields(t *testing.T) {
	b, err := json.Marshal(DirectDebitPaymentCancelResponse{})
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

func TestDirectDebitPaymentCancel_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4005700","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/cancel"
	tr := &Transport{}
	_, err := DirectDebitPaymentCancel(context.Background(), tr, hb, DirectDebitPaymentCancelRequest{})
	if err == nil {
		t.Fatal("DirectDebitPaymentCancel() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("DirectDebitPaymentCancel() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestDirectDebitPaymentCancel_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2005700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/cancel"
	tr := &Transport{}
	resp, err := DirectDebitPaymentCancel(context.Background(), tr, hb, DirectDebitPaymentCancelRequest{})
	if err == nil {
		t.Fatalf("DirectDebitPaymentCancel() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("DirectDebitPaymentCancel() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestDirectDebitPaymentCancel_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cancelTime":"2026-09-12T10:00:00+07:00"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/cancel"
	tr := &Transport{}
	resp, err := DirectDebitPaymentCancel(context.Background(), tr, hb, DirectDebitPaymentCancelRequest{})
	if err == nil {
		t.Fatalf("DirectDebitPaymentCancel() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
