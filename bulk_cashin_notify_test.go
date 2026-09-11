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

func TestNotifyBulkCashIn_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004100",
   "responseMessage":"Request has been processed successfully",
   "bulkId":"BULK000001",
   "partnerBulkId":"partner-bulk-1"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/notify-bulk-cash-in"
	tr := &Transport{}
	resp, err := NotifyBulkCashIn(context.Background(), tr, hb, NotifyBulkCashInRequest{
		BulkID:        "BULK000001",
		PartnerBulkID: "partner-bulk-1",
	})
	if err != nil {
		t.Fatalf("NotifyBulkCashIn() error = %v", err)
	}

	want := NotifyBulkCashInResponse{
		ResponseCode:    "2004100",
		ResponseMessage: "Request has been processed successfully",
		BulkID:          "BULK000001",
		PartnerBulkID:   "partner-bulk-1",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("NotifyBulkCashIn() = %+v, want %+v", resp, want)
	}
}

func TestNotifyBulkCashIn_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004100","responseMessage":"ok","bulkId":"BULK000001","partnerBulkId":"partner-bulk-1"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/notify-bulk-cash-in"
	tr := &Transport{}
	req := NotifyBulkCashInRequest{
		BulkID:        "BULK000001",
		PartnerBulkID: "partner-bulk-1",
		BulkObject: []BulkCashInNotificationItem{
			{
				CustomerNumber:     "98765",
				ReferenceNo:        "ref-1",
				PartnerReferenceNo: "partner-ref-1",
				ResponseCode:       "2004100",
				ResponseMessage:    "Success",
			},
		},
	}
	if _, err := NotifyBulkCashIn(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("NotifyBulkCashIn() error = %v", err)
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
	if item["customerNumber"] != "98765" {
		t.Errorf(`wire body bulkObject[0]["customerNumber"] = %v, want "98765"`, item["customerNumber"])
	}
	if item["responseCode"] != "2004100" {
		t.Errorf(`wire body bulkObject[0]["responseCode"] = %v, want "2004100"`, item["responseCode"])
	}
}

// TestNotifyBulkCashIn_MandatoryFieldsAlwaysSerialized pins that
// BulkID and PartnerBulkID — the two top-level request fields without
// omitempty — are always present on the wire, even as "".
func TestNotifyBulkCashIn_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004100","responseMessage":"ok","bulkId":"BULK000001","partnerBulkId":"partner-bulk-1"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/notify-bulk-cash-in"
	tr := &Transport{}
	if _, err := NotifyBulkCashIn(context.Background(), tr, hb, NotifyBulkCashInRequest{}); err != nil {
		t.Fatalf("NotifyBulkCashIn() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"bulkId", "partnerBulkId"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	if _, ok := got["bulkObject"]; ok {
		t.Error(`wire body has "bulkObject" key, want it omitted when empty`)
	}
}

func TestNotifyBulkCashIn_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004100","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/notify-bulk-cash-in"
	tr := &Transport{}
	_, err := NotifyBulkCashIn(context.Background(), tr, hb, NotifyBulkCashInRequest{})
	if err == nil {
		t.Fatal("NotifyBulkCashIn() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("NotifyBulkCashIn() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestNotifyBulkCashIn_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004100","responseMessage":"ok","bulkId":"BULK000001","partnerBulkId":"partner-bulk-1"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/notify-bulk-cash-in"
	tr := &Transport{}
	resp, err := NotifyBulkCashIn(context.Background(), tr, hb, NotifyBulkCashInRequest{})
	if err == nil {
		t.Fatalf("NotifyBulkCashIn() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("NotifyBulkCashIn() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestNotifyBulkCashIn_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bulkId":"BULK000001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/notify-bulk-cash-in"
	tr := &Transport{}
	resp, err := NotifyBulkCashIn(context.Background(), tr, hb, NotifyBulkCashInRequest{})
	if err == nil {
		t.Fatalf("NotifyBulkCashIn() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
