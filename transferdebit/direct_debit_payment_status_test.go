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

// TestDirectDebitPaymentStatusTypes_FieldCounts guards against a field
// silently added to any of these types without updating the
// wire-assertion tests below.
func TestDirectDebitPaymentStatusTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(DirectDebitRefundHistoryItem{}).NumField(); n != 6 {
		t.Errorf("DirectDebitRefundHistoryItem has %d fields, want 6", n)
	}
	if n := reflect.TypeOf(DirectDebitPaymentStatusRequest{}).NumField(); n != 9 {
		t.Errorf("DirectDebitPaymentStatusRequest has %d fields, want 9", n)
	}
	if n := reflect.TypeOf(DirectDebitPaymentStatusResponse{}).NumField(); n != 19 {
		t.Errorf("DirectDebitPaymentStatusResponse has %d fields, want 19", n)
	}
}

func TestDirectDebitPaymentStatus_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2005500",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"PARTNER001",
   "originalReferenceNo":"REF001",
   "originalExternalId":"EXT001",
   "serviceCode":"55",
   "transactionDate":"2026-09-12T09:00:00+07:00",
   "amount":{"value":"50000.00","currency":"IDR"},
   "approvalCode":"APPROVAL001",
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "originalResponseCode":"2005400",
   "originalResponseMessage":"Success",
   "sessionId":"SESSION001",
   "requestId":"REQUEST001",
   "refundHistory":[
      {"refundNo":"REFUND001","partnerRefundNo":"PARTNERREFUND001","refundAmount":{"value":"1000.00","currency":"IDR"},"refundStatus":"00","refundDate":"2026-09-12T10:00:00+07:00","reason":"customer request"}
   ],
   "transAmount":{"value":"50000.00","currency":"IDR"},
   "feeAmount":{"value":"1000.00","currency":"IDR"},
   "paidTime":"2026-09-12T09:05:00+07:00"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/status"
	tr := &snap.Transport{}
	resp, err := DirectDebitPaymentStatus(context.Background(), tr, hb, DirectDebitPaymentStatusRequest{
		ServiceCode: "55",
	})
	if err != nil {
		t.Fatalf("DirectDebitPaymentStatus() error = %v", err)
	}

	want := DirectDebitPaymentStatusResponse{
		ResponseCode:               "2005500",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		OriginalExternalID:         "EXT001",
		ServiceCode:                "55",
		TransactionDate:            "2026-09-12T09:00:00+07:00",
		Amount:                     &snap.Money{Value: "50000.00", Currency: "IDR"},
		ApprovalCode:               "APPROVAL001",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		OriginalResponseCode:       "2005400",
		OriginalResponseMessage:    "Success",
		SessionID:                  "SESSION001",
		RequestID:                  "REQUEST001",
		RefundHistory: []DirectDebitRefundHistoryItem{
			{
				RefundNo:        "REFUND001",
				PartnerRefundNo: "PARTNERREFUND001",
				RefundAmount:    &snap.Money{Value: "1000.00", Currency: "IDR"},
				RefundStatus:    "00",
				RefundDate:      "2026-09-12T10:00:00+07:00",
				Reason:          "customer request",
			},
		},
		TransAmount: &snap.Money{Value: "50000.00", Currency: "IDR"},
		FeeAmount:   &snap.Money{Value: "1000.00", Currency: "IDR"},
		PaidTime:    "2026-09-12T09:05:00+07:00",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("DirectDebitPaymentStatus() = %+v, want %+v", resp, want)
	}
}

func TestDirectDebitPaymentStatus_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005500","responseMessage":"ok","latestTransactionStatus":"00"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/status"
	tr := &snap.Transport{}
	req := DirectDebitPaymentStatusRequest{
		OriginalPartnerReferenceNo: "PARTNER001",
		OriginalReferenceNo:        "REF001",
		OriginalExternalID:         "EXT001",
		ServiceCode:                "55",
		TransactionDate:            "2026-09-12T09:00:00+07:00",
		Amount:                     &snap.Money{Value: "50000.00", Currency: "IDR"},
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		ExternalStoreID:            "STORE01",
	}
	if _, err := DirectDebitPaymentStatus(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("DirectDebitPaymentStatus() error = %v", err)
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
		"originalExternalId":         "EXT001",
		"serviceCode":                "55",
		"transactionDate":            "2026-09-12T09:00:00+07:00",
		"amount":                     map[string]any{"value": "50000.00", "currency": "IDR"},
		"merchantId":                 "MERCH01",
		"subMerchantId":              "SUBMERCH01",
		"externalStoreId":            "STORE01",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestDirectDebitPaymentStatus_MandatoryFieldAlwaysSerialized pins that
// ServiceCode — the only request field without omitempty — is always
// present on the wire, even as "", and every other field is omitted
// when unset.
func TestDirectDebitPaymentStatus_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005500","responseMessage":"ok","latestTransactionStatus":"00"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/status"
	tr := &snap.Transport{}
	if _, err := DirectDebitPaymentStatus(context.Background(), tr, hb, DirectDebitPaymentStatusRequest{}); err != nil {
		t.Fatalf("DirectDebitPaymentStatus() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"serviceCode": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestDirectDebitRefundHistoryItem_MandatoryFieldsHaveNoOmitempty pins
// that PartnerRefundNo and RefundStatus — Mandatory per research §5.1
// line 136-138 — always serialize, even from a zero-value item. The
// parse test above always populates every item field, so this is the
// only guard against a tag silently gaining omitempty.
func TestDirectDebitRefundHistoryItem_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(DirectDebitRefundHistoryItem{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value DirectDebitRefundHistoryItem: %v", err)
	}
	for _, key := range []string{"partnerRefundNo", "refundStatus"} {
		if _, ok := got[key]; !ok {
			t.Errorf(`marshaled zero-value DirectDebitRefundHistoryItem missing %q key; want it always present`, key)
		}
	}
}

// TestDirectDebitPaymentStatusResponse_ZeroValueOmitsOptionalFields
// pins that a zero-value response marshals to just the envelope plus
// LatestTransactionStatus (the response's only non-omitempty field
// beyond the envelope).
func TestDirectDebitPaymentStatusResponse_ZeroValueOmitsOptionalFields(t *testing.T) {
	b, err := json.Marshal(DirectDebitPaymentStatusResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{"responseCode": "", "responseMessage": "", "latestTransactionStatus": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestDirectDebitPaymentStatus_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4005500","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/status"
	tr := &snap.Transport{}
	_, err := DirectDebitPaymentStatus(context.Background(), tr, hb, DirectDebitPaymentStatusRequest{})
	if err == nil {
		t.Fatal("DirectDebitPaymentStatus() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("DirectDebitPaymentStatus() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestDirectDebitPaymentStatus_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2005500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/status"
	tr := &snap.Transport{}
	resp, err := DirectDebitPaymentStatus(context.Background(), tr, hb, DirectDebitPaymentStatusRequest{})
	if err == nil {
		t.Fatalf("DirectDebitPaymentStatus() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("DirectDebitPaymentStatus() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestDirectDebitPaymentStatus_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"latestTransactionStatus":"00"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/status"
	tr := &snap.Transport{}
	resp, err := DirectDebitPaymentStatus(context.Background(), tr, hb, DirectDebitPaymentStatusRequest{})
	if err == nil {
		t.Fatalf("DirectDebitPaymentStatus() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
