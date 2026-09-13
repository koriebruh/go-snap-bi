package transactionhistory

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

// standardWorkedExampleTransactionHistoryDetailResponse is the standard's
// own worked example response body (Code Snippets tab), hardcoded
// independently of this package's own types.
const standardWorkedExampleTransactionHistoryDetailResponse = `{
   "responseCode":"2001300",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "amount":{"value":"12345678.00","currency":"IDR"},
   "cancelledTime":"2009-07-03T12:08:56+07:00",
   "dateTime":"2009-07-03T12:08:56+07:00",
   "refundAmount":{"value":"12345678.00","currency":"IDR"},
   "remark":"Payment to Warung Ikan Bakar",
   "sourceOfFunds":[{"source":"BALANCE","amount":{"value":"10000.00","currency":"IDR"}}],
   "status":"SUCCESS",
   "type":"PAYMENT",
   "additionalInfo":{"deviceId":"12345679237","channel":"mobilephone"}
}`

func TestTransactionHistoryDetailTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(TransactionHistoryDetailRequest{}).NumField(); n != 2 {
		t.Errorf("TransactionHistoryDetailRequest has %d fields, want 2", n)
	}
	if n := reflect.TypeOf(TransactionHistoryDetailResponse{}).NumField(); n != 13 {
		t.Errorf("TransactionHistoryDetailResponse has %d fields, want 13", n)
	}
}

func TestTransactionHistoryDetailRequest_RoundTrips(t *testing.T) {
	const fixture = `{"originalPartnerReferenceNo":"2020102900000000000001","additionalInfo":{"deviceId":"12345679237","channel":"mobilephone"}}`
	var got TransactionHistoryDetailRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := TransactionHistoryDetailRequest{
		OriginalPartnerReferenceNo: "2020102900000000000001",
		AdditionalInfo:             json.RawMessage(`{"deviceId":"12345679237","channel":"mobilephone"}`),
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
		"originalPartnerReferenceNo": "2020102900000000000001",
		"additionalInfo":             map[string]any{"deviceId": "12345679237", "channel": "mobilephone"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestTransactionHistoryDetailRequest_MandatoryFieldsHaveNoOmitempty pins
// that OriginalPartnerReferenceNo — the only field without omitempty —
// always serializes, even from a zero-value request.
func TestTransactionHistoryDetailRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(TransactionHistoryDetailRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{
		"originalPartnerReferenceNo": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional and unset)", got, want)
	}
}

func TestTransactionHistoryDetailResponse_RoundTrips(t *testing.T) {
	var got TransactionHistoryDetailResponse
	if err := json.Unmarshal([]byte(standardWorkedExampleTransactionHistoryDetailResponse), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := TransactionHistoryDetailResponse{
		ResponseCode:       "2001300",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		Amount:             &snap.Money{Value: "12345678.00", Currency: "IDR"},
		CancelledTime:      "2009-07-03T12:08:56+07:00",
		DateTime:           "2009-07-03T12:08:56+07:00",
		RefundAmount:       &snap.Money{Value: "12345678.00", Currency: "IDR"},
		Remark:             "Payment to Warung Ikan Bakar",
		SourceOfFunds: []SourceOfFund{
			{Source: "BALANCE", Amount: &snap.Money{Value: "10000.00", Currency: "IDR"}},
		},
		Status:         "SUCCESS",
		Type:           "PAYMENT",
		AdditionalInfo: json.RawMessage(`{"deviceId":"12345679237","channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestTransactionHistoryDetailResponse_MandatoryFieldsHaveNoOmitempty pins
// ResponseCode, ResponseMessage, DateTime, Status, and Type — the
// fields without omitempty — as always-serializing. Amount and
// RefundAmount are Optional (*snap.Money, omitempty) and must be
// absent from a zero-value response, not present as empty objects.
func TestTransactionHistoryDetailResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(TransactionHistoryDetailResponse{})
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
		"dateTime":        "",
		"status":          "",
		"type":            "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestTransactionHistoryDetail_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := TransactionHistoryDetailRequest{
		OriginalPartnerReferenceNo: "2020102900000000000001",
		AdditionalInfo:             json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}

func TestTransactionHistoryDetail_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(standardWorkedExampleTransactionHistoryDetailResponse))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-detail"
	tr := &snap.Transport{}
	resp, err := TransactionHistoryDetail(context.Background(), tr, hb, TransactionHistoryDetailRequest{
		OriginalPartnerReferenceNo: "2020102900000000000001",
	})
	if err != nil {
		t.Fatalf("TransactionHistoryDetail() error = %v", err)
	}

	want := TransactionHistoryDetailResponse{
		ResponseCode:       "2001300",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		Amount:             &snap.Money{Value: "12345678.00", Currency: "IDR"},
		CancelledTime:      "2009-07-03T12:08:56+07:00",
		DateTime:           "2009-07-03T12:08:56+07:00",
		RefundAmount:       &snap.Money{Value: "12345678.00", Currency: "IDR"},
		Remark:             "Payment to Warung Ikan Bakar",
		SourceOfFunds: []SourceOfFund{
			{Source: "BALANCE", Amount: &snap.Money{Value: "10000.00", Currency: "IDR"}},
		},
		Status:         "SUCCESS",
		Type:           "PAYMENT",
		AdditionalInfo: json.RawMessage(`{"deviceId":"12345679237","channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("TransactionHistoryDetail() = %+v, want %+v", resp, want)
	}
}

func TestTransactionHistoryDetail_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001300","responseMessage":"ok","amount":{"value":"","currency":""},"refundAmount":{"value":"","currency":""}}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-detail"
	tr := &snap.Transport{}
	req := TransactionHistoryDetailRequest{
		OriginalPartnerReferenceNo: "2020102900000000000001",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := TransactionHistoryDetail(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("TransactionHistoryDetail() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["originalPartnerReferenceNo"] != "2020102900000000000001" {
		t.Errorf(`wire body["originalPartnerReferenceNo"] = %v, want "2020102900000000000001"`, got["originalPartnerReferenceNo"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

func TestTransactionHistoryDetail_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001300","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-detail"
	tr := &snap.Transport{}
	_, err := TransactionHistoryDetail(context.Background(), tr, hb, TransactionHistoryDetailRequest{
		OriginalPartnerReferenceNo: "2020102900000000000001",
	})
	if err == nil {
		t.Fatal("TransactionHistoryDetail() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("TransactionHistoryDetail() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestTransactionHistoryDetail_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(standardWorkedExampleTransactionHistoryDetailResponse)) // responseCode "2001300"
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-detail"
	tr := &snap.Transport{}
	resp, err := TransactionHistoryDetail(context.Background(), tr, hb, TransactionHistoryDetailRequest{
		OriginalPartnerReferenceNo: "2020102900000000000001",
	})
	if err == nil {
		t.Fatalf("TransactionHistoryDetail() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("TransactionHistoryDetail() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestTransactionHistoryDetail_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"2020102977770000000009"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-detail"
	tr := &snap.Transport{}
	resp, err := TransactionHistoryDetail(context.Background(), tr, hb, TransactionHistoryDetailRequest{
		OriginalPartnerReferenceNo: "2020102900000000000001",
	})
	if err == nil {
		t.Fatalf("TransactionHistoryDetail() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
