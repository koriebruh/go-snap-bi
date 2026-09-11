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

func TestInterbankBulkTransfer_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2002000",
   "responseMessage":"Request has been processed successfully",
   "bulkId":"BULK123456",
   "partnerBulkId":"partner-bulk-1",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank-bulk"
	tr := &Transport{}
	resp, err := InterbankBulkTransfer(context.Background(), tr, hb, InterbankBulkTransferRequest{
		CustomerReference: "cust-ref-1",
		SourceAccountNo:   "9876543210",
		TransactionDate:   "2020-12-21T14:56:11+07:00",
		BulkObject: []InterbankBulkTransferItem{
			{PartnerReferenceNo: "pr-1", BankCode: "014", BeneficiaryAccountNo: "1234567890", BeneficiaryAccountName: "Jane Doe", Amount: Money{Value: "50000.00", Currency: "IDR"}},
		},
	})
	if err != nil {
		t.Fatalf("InterbankBulkTransfer() error = %v", err)
	}

	want := InterbankBulkTransferResponse{
		ResponseCode:    "2002000",
		ResponseMessage: "Request has been processed successfully",
		BulkID:          "BULK123456",
		PartnerBulkID:   "partner-bulk-1",
		AdditionalInfo:  json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("InterbankBulkTransfer() = %+v, want %+v", resp, want)
	}
}

func TestInterbankBulkTransfer_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank-bulk"
	tr := &Transport{}
	req := InterbankBulkTransferRequest{
		PartnerBulkID:     "partner-bulk-1",
		CustomerReference: "cust-ref-1",
		SourceAccountNo:   "9876543210",
		TransactionDate:   "2020-12-21T14:56:11+07:00",
		BulkObject: []InterbankBulkTransferItem{
			{
				PartnerReferenceNo:     "pr-1",
				BankCode:               "014",
				BeneficiaryAccountNo:   "1234567890",
				BeneficiaryAccountName: "Jane Doe",
				Amount:                 Money{Value: "50000.00", Currency: "IDR"},
				OriginatorInfos: []TransferOriginatorInfo{
					{OriginatorCustomerNo: "cust-1", OriginatorCustomerName: "John Doe", OriginatorBankCode: "014"},
				},
			},
		},
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := InterbankBulkTransfer(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("InterbankBulkTransfer() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerBulkId"] != "partner-bulk-1" {
		t.Errorf(`wire body["partnerBulkId"] = %v, want "partner-bulk-1"`, got["partnerBulkId"])
	}
	if got["customerReference"] != "cust-ref-1" {
		t.Errorf(`wire body["customerReference"] = %v, want "cust-ref-1"`, got["customerReference"])
	}
	bulkObject, ok := got["bulkObject"].([]any)
	if !ok || len(bulkObject) != 1 {
		t.Fatalf(`wire body["bulkObject"] = %v, want a 1-element array`, got["bulkObject"])
	}
	item, ok := bulkObject[0].(map[string]any)
	if !ok || item["partnerReferenceNo"] != "pr-1" || item["bankCode"] != "014" || item["beneficiaryAccountNo"] != "1234567890" || item["beneficiaryAccountName"] != "Jane Doe" {
		t.Errorf(`wire body["bulkObject"][0] = %v, want the test's item`, bulkObject[0])
	}
	amount, ok := item["amount"].(map[string]any)
	if !ok || amount["value"] != "50000.00" || amount["currency"] != "IDR" {
		t.Errorf(`wire body["bulkObject"][0]["amount"] = %v, want {"value":"50000.00","currency":"IDR"}`, item["amount"])
	}
	originatorInfos, ok := item["originatorInfos"].([]any)
	if !ok || len(originatorInfos) != 1 {
		t.Fatalf(`wire body["bulkObject"][0]["originatorInfos"] = %v, want a 1-element array`, item["originatorInfos"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestInterbankBulkTransfer_MandatoryFieldsAlwaysSerialized pins that
// CustomerReference, SourceAccountNo, TransactionDate, and BulkObject —
// the request fields without omitempty — are always present on the
// wire. BulkObject is a nil slice without omitempty, which
// encoding/json marshals as the JSON literal null, not an empty array —
// this test pins that actual wire behavior.
func TestInterbankBulkTransfer_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank-bulk"
	tr := &Transport{}
	if _, err := InterbankBulkTransfer(context.Background(), tr, hb, InterbankBulkTransferRequest{}); err != nil {
		t.Fatalf("InterbankBulkTransfer() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"customerReference", "sourceAccountNo", "transactionDate"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	bulkObject, ok := got["bulkObject"]
	if !ok {
		t.Fatal(`wire body missing "bulkObject" key; BulkObject lacks omitempty and must always be present`)
	}
	if bulkObject != nil {
		t.Errorf(`wire body["bulkObject"] = %v, want null (nil slice without omitempty)`, bulkObject)
	}
}

func TestInterbankBulkTransfer_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4002000","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank-bulk"
	tr := &Transport{}
	_, err := InterbankBulkTransfer(context.Background(), tr, hb, InterbankBulkTransferRequest{})
	if err == nil {
		t.Fatal("InterbankBulkTransfer() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("InterbankBulkTransfer() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestInterbankBulkTransfer_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2002000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank-bulk"
	tr := &Transport{}
	resp, err := InterbankBulkTransfer(context.Background(), tr, hb, InterbankBulkTransferRequest{})
	if err == nil {
		t.Fatalf("InterbankBulkTransfer() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("InterbankBulkTransfer() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestInterbankBulkTransfer_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bulkId":"BULK123456"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank-bulk"
	tr := &Transport{}
	resp, err := InterbankBulkTransfer(context.Background(), tr, hb, InterbankBulkTransferRequest{})
	if err == nil {
		t.Fatalf("InterbankBulkTransfer() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
