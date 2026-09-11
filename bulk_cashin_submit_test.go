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
	"sync/atomic"
	"testing"
)

func TestSubmitBulkCashIn_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004000",
   "responseMessage":"Request has been processed successfully",
   "bulkid":"BULK000001",
   "partnerBulkId":"partner-bulk-1"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/submit-bulk-cash-in"
	tr := &Transport{}
	resp, err := SubmitBulkCashIn(context.Background(), tr, hb, SubmitBulkCashInRequest{
		TransactionDate: "2020-12-20T10:00:00+07:00",
	})
	if err != nil {
		t.Fatalf("SubmitBulkCashIn() error = %v", err)
	}

	want := SubmitBulkCashInResponse{
		ResponseCode:    "2004000",
		ResponseMessage: "Request has been processed successfully",
		BulkID:          "BULK000001",
		PartnerBulkID:   "partner-bulk-1",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("SubmitBulkCashIn() = %+v, want %+v", resp, want)
	}
}

func TestSubmitBulkCashIn_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004000","responseMessage":"ok","bulkid":"BULK000001"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/submit-bulk-cash-in"
	tr := &Transport{}
	req := SubmitBulkCashInRequest{
		PartnerBulkID:   "partner-bulk-1",
		TransactionDate: "2020-12-20T10:00:00+07:00",
		BulkObject: []BulkCashInItem{
			{AccountNumber: "1122334455", PartnerReferenceNo: "ref-1", Amount: &Money{Value: "100000.00", Currency: "IDR"}},
		},
	}
	if _, err := SubmitBulkCashIn(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("SubmitBulkCashIn() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	bulkObject, ok := got["bulkObject"].([]any)
	if !ok || len(bulkObject) != 1 {
		t.Fatalf(`wire body["bulkObject"] = %v, want a 1-element array`, got["bulkObject"])
	}
	item, ok := bulkObject[0].(map[string]any)
	if !ok {
		t.Fatalf("wire body bulkObject[0] = %v, want an object", bulkObject[0])
	}
	if item["accountNumber"] != "1122334455" {
		t.Errorf(`wire body bulkObject[0]["accountNumber"] = %v, want "1122334455"`, item["accountNumber"])
	}
	if item["partnerReferenceNo"] != "ref-1" {
		t.Errorf(`wire body bulkObject[0]["partnerReferenceNo"] = %v, want "ref-1"`, item["partnerReferenceNo"])
	}
}

// TestSubmitBulkCashIn_MandatoryFieldAlwaysSerialized pins that
// TransactionDate — the only top-level request field without
// omitempty — is always present on the wire, even as "".
func TestSubmitBulkCashIn_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004000","responseMessage":"ok","bulkid":"BULK000001"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/submit-bulk-cash-in"
	tr := &Transport{}
	if _, err := SubmitBulkCashIn(context.Background(), tr, hb, SubmitBulkCashInRequest{}); err != nil {
		t.Fatalf("SubmitBulkCashIn() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	v, ok := got["transactionDate"]
	if !ok {
		t.Fatal(`wire body missing "transactionDate" key; want it always present, even as ""`)
	}
	if v != "" {
		t.Errorf(`wire body["transactionDate"] = %v, want ""`, v)
	}
	if _, ok := got["bulkObject"]; ok {
		t.Error(`wire body has "bulkObject" key, want it omitted when empty`)
	}
}

// TestSubmitBulkCashInResponse_MarshalsBulkIDAsLowercaseD pins the
// deliberate "bulkid" (lowercase d) casing decision on this type's
// BulkID field, distinct from NotifyBulkCashInResponse's camelCase
// "bulkId" — see the type's doc comment for the unresolved research
// contradiction this reflects. encoding/json matches tag keys
// case-insensitively on decode, so nothing else in the package would
// catch a future edit accidentally "fixing" this tag to camelCase;
// only a marshal-based assertion observes the literal casing.
func TestSubmitBulkCashInResponse_MarshalsBulkIDAsLowercaseD(t *testing.T) {
	b, err := json.Marshal(SubmitBulkCashInResponse{BulkID: "BULK000001"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled bytes: %v", err)
	}
	if _, ok := got["bulkid"]; !ok {
		t.Errorf(`marshaled SubmitBulkCashInResponse missing "bulkid" (lowercase d) key; got keys %v`, got)
	}
	if _, ok := got["bulkId"]; ok {
		t.Error(`marshaled SubmitBulkCashInResponse has "bulkId" (camelCase) key, want only lowercase-d "bulkid"`)
	}
}

func TestSubmitBulkCashIn_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2004000","responseMessage":"ok","bulkid":"BULK000001"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/submit-bulk-cash-in"
	tr := &Transport{}
	_, err := SubmitBulkCashIn(context.Background(), tr, hb, SubmitBulkCashInRequest{
		AdditionalInfo: json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("SubmitBulkCashIn() error = nil, want non-nil for malformed AdditionalInfo JSON")
	}
	if requested.Load() {
		t.Error("SubmitBulkCashIn() sent an HTTP request despite a request-encoding failure")
	}
}

func TestSubmitBulkCashIn_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004000","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/submit-bulk-cash-in"
	tr := &Transport{}
	_, err := SubmitBulkCashIn(context.Background(), tr, hb, SubmitBulkCashInRequest{})
	if err == nil {
		t.Fatal("SubmitBulkCashIn() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("SubmitBulkCashIn() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestSubmitBulkCashIn_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004000","responseMessage":"ok","bulkid":"BULK000001"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/submit-bulk-cash-in"
	tr := &Transport{}
	resp, err := SubmitBulkCashIn(context.Background(), tr, hb, SubmitBulkCashInRequest{})
	if err == nil {
		t.Fatalf("SubmitBulkCashIn() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("SubmitBulkCashIn() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestSubmitBulkCashIn_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bulkid":"BULK000001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/submit-bulk-cash-in"
	tr := &Transport{}
	resp, err := SubmitBulkCashIn(context.Background(), tr, hb, SubmitBulkCashInRequest{})
	if err == nil {
		t.Fatalf("SubmitBulkCashIn() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
