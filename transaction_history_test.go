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

// standardWorkedExampleTransactionHistoryListResponse is the standard's own
// worked example response body (Code Snippets tab), hardcoded independently
// of this package's own types.
const standardWorkedExampleTransactionHistoryListResponse = `{
   "responseCode":"2001200",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "detailData":[
      {
         "dateTime":"2019-07-03T12:08:56+07:00",
         "amount":{"value":"12345678.00","currency":"IDR"},
         "remark":"Payment to Warung Ikan Bakar",
         "sourceOfFunds":[
            {"source":"BALANCE","amount":{"value":"10000.00","currency":"IDR"}}
         ],
         "status":"SUCCESS",
         "type":"PAYMENT"
      }
   ]
}`

func TestTransactionHistoryList_ParsesWorkedExampleResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(standardWorkedExampleTransactionHistoryListResponse))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-list"
	tr := &Transport{}
	resp, err := TransactionHistoryList(context.Background(), tr, hb, TransactionHistoryListRequest{
		PartnerReferenceNo: "2020102900000000000001",
	})
	if err != nil {
		t.Fatalf("TransactionHistoryList() error = %v", err)
	}

	if resp.ResponseCode != "2001200" {
		t.Errorf("ResponseCode = %q, want %q", resp.ResponseCode, "2001200")
	}
	if resp.ReferenceNo != "2020102977770000000009" {
		t.Errorf("ReferenceNo = %q, want %q", resp.ReferenceNo, "2020102977770000000009")
	}
	if resp.PartnerReferenceNo != "2020102900000000000001" {
		t.Errorf("PartnerReferenceNo = %q, want %q", resp.PartnerReferenceNo, "2020102900000000000001")
	}
	if len(resp.DetailData) != 1 {
		t.Fatalf("len(DetailData) = %d, want 1", len(resp.DetailData))
	}
	want := TransactionDetail{
		DateTime: "2019-07-03T12:08:56+07:00",
		Amount:   Money{Value: "12345678.00", Currency: "IDR"},
		Remark:   "Payment to Warung Ikan Bakar",
		SourceOfFunds: []SourceOfFund{
			{Source: "BALANCE", Amount: Money{Value: "10000.00", Currency: "IDR"}},
		},
		Status: "SUCCESS",
		Type:   "PAYMENT",
	}
	got := resp.DetailData[0]
	got.AdditionalInfo = nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DetailData[0] = %+v, want %+v", got, want)
	}
}

func TestTransactionHistoryList_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-list"
	tr := &Transport{}
	req := TransactionHistoryListRequest{
		PartnerReferenceNo: "ref-1",
		FromDateTime:       "2019-07-03T12:08:56+07:00",
		ToDateTime:         "2019-07-04T12:08:56+07:00",
		PageSize:           "10",
		PageNumber:         "2",
	}
	if _, err := TransactionHistoryList(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("TransactionHistoryList() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerReferenceNo"] != "ref-1" {
		t.Errorf(`wire body["partnerReferenceNo"] = %v, want "ref-1"`, got["partnerReferenceNo"])
	}
	if got["pageSize"] != "10" {
		t.Errorf(`wire body["pageSize"] = %v (type %T), want string "10"`, got["pageSize"], got["pageSize"])
	}
	if got["pageNumber"] != "2" {
		t.Errorf(`wire body["pageNumber"] = %v, want "2"`, got["pageNumber"])
	}
}

func TestTransactionHistoryList_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001200","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-list"
	tr := &Transport{}
	_, err := TransactionHistoryList(context.Background(), tr, hb, TransactionHistoryListRequest{})
	if err == nil {
		t.Fatal("TransactionHistoryList() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("TransactionHistoryList() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

// TestTransactionHistoryList_NonTwoXXStatusWithTwoXXBodyIsError is the
// binding-specific regression test for the Phase 2 santa-loop round-3 HIGH
// finding: an HTTP 500 whose body still carries a 2xx-class responseCode
// must not be treated as success. Unlike the plain non-2xx-responseCode
// test above (which would pass even under the old, pre-fix pattern, since
// envelopeError alone already classifies a genuinely non-2xx code
// correctly), this specifically exercises checkResponseStatus's
// transport-status-is-authoritative behavior at this binding's own call
// site.
func TestTransactionHistoryList_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(standardWorkedExampleTransactionHistoryListResponse)) // responseCode "2001200"
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-list"
	tr := &Transport{}
	resp, err := TransactionHistoryList(context.Background(), tr, hb, TransactionHistoryListRequest{})
	if err == nil {
		t.Fatalf("TransactionHistoryList() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("TransactionHistoryList() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}
