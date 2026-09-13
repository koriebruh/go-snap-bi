package transferkredit

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
         "type":"PAYMENT",
         "additionalInfo":{"note":"per-transaction"}
      }
   ],
   "additionalInfo":{"deviceId":"12345679237","channel":"mobilephone"}
}`

func TestTransactionHistoryList_ParsesWorkedExampleResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(standardWorkedExampleTransactionHistoryListResponse))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-list"
	tr := &snap.Transport{}
	resp, err := TransactionHistoryList(context.Background(), tr, hb, TransactionHistoryListRequest{
		PartnerReferenceNo: "2020102900000000000001",
	})
	if err != nil {
		t.Fatalf("TransactionHistoryList() error = %v", err)
	}

	// Compare the entire decoded response against a literal expected value —
	// not a hand-picked subset of fields — so a typo'd json tag on any
	// field, including responseMessage, can't pass silently. reflect.DeepEqual
	// (not !=) because AdditionalInfo is a json.RawMessage ([]byte).
	want := TransactionHistoryListResponse{
		ResponseCode:       "2001200",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		DetailData: []TransactionDetail{
			{
				DateTime: "2019-07-03T12:08:56+07:00",
				Amount:   snap.Money{Value: "12345678.00", Currency: "IDR"},
				Remark:   "Payment to Warung Ikan Bakar",
				SourceOfFunds: []SourceOfFund{
					{Source: "BALANCE", Amount: snap.Money{Value: "10000.00", Currency: "IDR"}},
				},
				Status:         "SUCCESS",
				Type:           "PAYMENT",
				AdditionalInfo: json.RawMessage(`{"note":"per-transaction"}`),
			},
		},
		AdditionalInfo: json.RawMessage(`{"deviceId":"12345679237","channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("TransactionHistoryList() = %+v, want %+v", resp, want)
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

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-list"
	tr := &snap.Transport{}
	req := TransactionHistoryListRequest{
		PartnerReferenceNo: "ref-1",
		FromDateTime:       "2019-07-03T12:08:56+07:00",
		ToDateTime:         "2019-07-04T12:08:56+07:00",
		PageSize:           "10",
		PageNumber:         "2",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
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
	if got["fromDateTime"] != "2019-07-03T12:08:56+07:00" {
		t.Errorf(`wire body["fromDateTime"] = %v, want "2019-07-03T12:08:56+07:00"`, got["fromDateTime"])
	}
	if got["toDateTime"] != "2019-07-04T12:08:56+07:00" {
		t.Errorf(`wire body["toDateTime"] = %v, want "2019-07-04T12:08:56+07:00"`, got["toDateTime"])
	}
	if got["pageSize"] != "10" {
		t.Errorf(`wire body["pageSize"] = %v (type %T), want string "10"`, got["pageSize"], got["pageSize"])
	}
	if got["pageNumber"] != "2" {
		t.Errorf(`wire body["pageNumber"] = %v, want "2"`, got["pageNumber"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

func TestTransactionHistoryList_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001200","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-list"
	tr := &snap.Transport{}
	_, err := TransactionHistoryList(context.Background(), tr, hb, TransactionHistoryListRequest{})
	if err == nil {
		t.Fatal("TransactionHistoryList() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("TransactionHistoryList() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

// TestTransactionHistoryList_TwoXXStatusWithNoResponseCodeIsError is the
// binding-level regression test for a santa-loop finding: the "HTTP 200,
// but the JSON body itself carries no responseCode" path had never been
// exercised at HTTP 200 anywhere in the package (every other
// no-responseCode test pairs it with a non-2xx status).
func TestTransactionHistoryList_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"detailData":[]}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-list"
	tr := &snap.Transport{}
	resp, err := TransactionHistoryList(context.Background(), tr, hb, TransactionHistoryListRequest{})
	if err == nil {
		t.Fatalf("TransactionHistoryList() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}

// TestTransactionHistoryList_NonTwoXXStatusWithTwoXXBodyIsError is the
// binding-specific regression test for the Phase 2 santa-loop round-3 HIGH
// finding: an HTTP 500 whose body still carries a 2xx-class responseCode
// must not be treated as success. Unlike the plain non-2xx-responseCode
// test above (which would pass even under the old, pre-fix pattern, since
// envelopeError alone already classifies a genuinely non-2xx code
// correctly), this specifically exercises snap.CheckResponseStatus's
// transport-status-is-authoritative behavior at this binding's own call
// site.
func TestTransactionHistoryList_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(standardWorkedExampleTransactionHistoryListResponse)) // responseCode "2001200"
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-history-list"
	tr := &snap.Transport{}
	resp, err := TransactionHistoryList(context.Background(), tr, hb, TransactionHistoryListRequest{})
	if err == nil {
		t.Fatalf("TransactionHistoryList() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("TransactionHistoryList() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}
