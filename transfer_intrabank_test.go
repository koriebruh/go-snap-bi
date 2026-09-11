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

func TestIntrabankTransfer_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2001700",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "amount":{"value":"50000.00","currency":"IDR"},
   "beneficiaryAccountNo":"1234567890",
   "currency":"IDR",
   "customerReference":"cust-ref-1",
   "sourceAccountNo":"9876543210",
   "transactionDate":"2020-12-21T14:56:11+07:00",
   "originatorInfos":[{"originatorCustomerNo":"cust-1","originatorCustomerName":"John Doe","originatorBankCode":"014"}],
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-intrabank"
	tr := &Transport{}
	resp, err := IntrabankTransfer(context.Background(), tr, hb, IntrabankTransferRequest{
		PartnerReferenceNo:   "2020102900000000000001",
		Amount:               TransferAmount{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountNo: "1234567890",
		SourceAccountNo:      "9876543210",
		TransactionDate:      "2020-12-21T14:56:11+07:00",
	})
	if err != nil {
		t.Fatalf("IntrabankTransfer() error = %v", err)
	}

	want := IntrabankTransferResponse{
		ResponseCode:         "2001700",
		ResponseMessage:      "Request has been processed successfully",
		ReferenceNo:          "2020102977770000000009",
		PartnerReferenceNo:   "2020102900000000000001",
		Amount:               &TransferAmount{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountNo: "1234567890",
		Currency:             "IDR",
		CustomerReference:    "cust-ref-1",
		SourceAccountNo:      "9876543210",
		TransactionDate:      "2020-12-21T14:56:11+07:00",
		OriginatorInfos: []TransferOriginatorInfo{
			{OriginatorCustomerNo: "cust-1", OriginatorCustomerName: "John Doe", OriginatorBankCode: "014"},
		},
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("IntrabankTransfer() = %+v, want %+v", resp, want)
	}
}

func TestIntrabankTransfer_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-intrabank"
	tr := &Transport{}
	req := IntrabankTransferRequest{
		PartnerReferenceNo:   "2020102900000000000001",
		Amount:               TransferAmount{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountNo: "1234567890",
		SourceAccountNo:      "9876543210",
		TransactionDate:      "2020-12-21T14:56:11+07:00",
		OriginatorInfos: []TransferOriginatorInfo{
			{OriginatorCustomerNo: "cust-1", OriginatorCustomerName: "John Doe", OriginatorBankCode: "014"},
		},
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := IntrabankTransfer(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("IntrabankTransfer() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerReferenceNo"] != "2020102900000000000001" {
		t.Errorf(`wire body["partnerReferenceNo"] = %v, want "2020102900000000000001"`, got["partnerReferenceNo"])
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok || amount["value"] != "50000.00" || amount["currency"] != "IDR" {
		t.Errorf(`wire body["amount"] = %v, want {"value":"50000.00","currency":"IDR"}`, got["amount"])
	}
	if got["beneficiaryAccountNo"] != "1234567890" {
		t.Errorf(`wire body["beneficiaryAccountNo"] = %v, want "1234567890"`, got["beneficiaryAccountNo"])
	}
	if got["sourceAccountNo"] != "9876543210" {
		t.Errorf(`wire body["sourceAccountNo"] = %v, want "9876543210"`, got["sourceAccountNo"])
	}
	if got["transactionDate"] != "2020-12-21T14:56:11+07:00" {
		t.Errorf(`wire body["transactionDate"] = %v, want "2020-12-21T14:56:11+07:00"`, got["transactionDate"])
	}
	originatorInfos, ok := got["originatorInfos"].([]any)
	if !ok || len(originatorInfos) != 1 {
		t.Fatalf(`wire body["originatorInfos"] = %v, want a 1-element array`, got["originatorInfos"])
	}
	originator, ok := originatorInfos[0].(map[string]any)
	if !ok || originator["originatorCustomerNo"] != "cust-1" || originator["originatorCustomerName"] != "John Doe" || originator["originatorBankCode"] != "014" {
		t.Errorf(`wire body["originatorInfos"][0] = %v, want the test's originator info`, originatorInfos[0])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestIntrabankTransfer_MandatoryFieldsAlwaysSerialized pins that
// PartnerReferenceNo, BeneficiaryAccountNo, SourceAccountNo, and
// TransactionDate — the request fields without omitempty — are always
// present on the wire, even as "", mirroring
// TestCardRegistration_MandatoryFieldsAlwaysSerialized.
func TestIntrabankTransfer_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-intrabank"
	tr := &Transport{}
	if _, err := IntrabankTransfer(context.Background(), tr, hb, IntrabankTransferRequest{}); err != nil {
		t.Fatalf("IntrabankTransfer() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerReferenceNo", "beneficiaryAccountNo", "sourceAccountNo", "transactionDate"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	if _, ok := got["amount"]; !ok {
		t.Error(`wire body missing "amount" key; Amount is a plain (non-pointer) struct field and must always be present`)
	}
}

func TestIntrabankTransfer_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001700","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-intrabank"
	tr := &Transport{}
	_, err := IntrabankTransfer(context.Background(), tr, hb, IntrabankTransferRequest{})
	if err == nil {
		t.Fatal("IntrabankTransfer() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("IntrabankTransfer() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestIntrabankTransfer_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2001700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-intrabank"
	tr := &Transport{}
	resp, err := IntrabankTransfer(context.Background(), tr, hb, IntrabankTransferRequest{})
	if err == nil {
		t.Fatalf("IntrabankTransfer() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("IntrabankTransfer() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestIntrabankTransfer_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1234567890"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-intrabank"
	tr := &Transport{}
	resp, err := IntrabankTransfer(context.Background(), tr, hb, IntrabankTransferRequest{})
	if err == nil {
		t.Fatalf("IntrabankTransfer() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
