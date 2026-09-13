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

func TestInterbankTransfer_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2001800",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "amount":{"value":"50000.00","currency":"IDR"},
   "beneficiaryAccountNo":"1234567890",
   "currency":"IDR",
   "customerReference":"cust-ref-1",
   "sourceAccountNo":"9876543210",
   "transactionDate":"2020-12-21T14:56:11+07:00",
   "traceNo":"TRACE123456",
   "originatorInfos":[{"originatorCustomerNo":"cust-1","originatorCustomerName":"John Doe","originatorBankCode":"014"}],
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank"
	tr := &snap.Transport{}
	resp, err := InterbankTransfer(context.Background(), tr, hb, InterbankTransferRequest{
		PartnerReferenceNo:     "2020102900000000000001",
		Amount:                 snap.Money{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountNo:   "1234567890",
		BeneficiaryAccountName: "Jane Doe",
		BeneficiaryBankCode:    "014",
		SourceAccountNo:        "9876543210",
		TransactionDate:        "2020-12-21T14:56:11+07:00",
	})
	if err != nil {
		t.Fatalf("InterbankTransfer() error = %v", err)
	}

	want := InterbankTransferResponse{
		ResponseCode:         "2001800",
		ResponseMessage:      "Request has been processed successfully",
		ReferenceNo:          "2020102977770000000009",
		PartnerReferenceNo:   "2020102900000000000001",
		Amount:               &snap.Money{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountNo: "1234567890",
		Currency:             "IDR",
		CustomerReference:    "cust-ref-1",
		SourceAccountNo:      "9876543210",
		TransactionDate:      "2020-12-21T14:56:11+07:00",
		TraceNo:              "TRACE123456",
		OriginatorInfos: []TransferOriginatorInfo{
			{OriginatorCustomerNo: "cust-1", OriginatorCustomerName: "John Doe", OriginatorBankCode: "014"},
		},
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("InterbankTransfer() = %+v, want %+v", resp, want)
	}
}

func TestInterbankTransfer_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank"
	tr := &snap.Transport{}
	req := InterbankTransferRequest{
		PartnerReferenceNo:     "2020102900000000000001",
		Amount:                 snap.Money{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountNo:   "1234567890",
		BeneficiaryAccountName: "Jane Doe",
		BeneficiaryBankCode:    "014",
		SourceAccountNo:        "9876543210",
		TransactionDate:        "2020-12-21T14:56:11+07:00",
		OriginatorInfos: []TransferOriginatorInfo{
			{OriginatorCustomerNo: "cust-1", OriginatorCustomerName: "John Doe", OriginatorBankCode: "014"},
		},
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := InterbankTransfer(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("InterbankTransfer() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["beneficiaryAccountName"] != "Jane Doe" {
		t.Errorf(`wire body["beneficiaryAccountName"] = %v, want "Jane Doe"`, got["beneficiaryAccountName"])
	}
	if got["beneficiaryBankCode"] != "014" {
		t.Errorf(`wire body["beneficiaryBankCode"] = %v, want "014"`, got["beneficiaryBankCode"])
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok || amount["value"] != "50000.00" || amount["currency"] != "IDR" {
		t.Errorf(`wire body["amount"] = %v, want {"value":"50000.00","currency":"IDR"}`, got["amount"])
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

// TestInterbankTransfer_MandatoryFieldsAlwaysSerialized pins that
// PartnerReferenceNo, Amount, BeneficiaryAccountNo,
// BeneficiaryAccountName, BeneficiaryBankCode, SourceAccountNo, and
// TransactionDate — the seven request fields without omitempty — are
// always present on the wire, even as their zero value, mirroring
// TestCardRegistration_MandatoryFieldsAlwaysSerialized.
func TestInterbankTransfer_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank"
	tr := &snap.Transport{}
	if _, err := InterbankTransfer(context.Background(), tr, hb, InterbankTransferRequest{}); err != nil {
		t.Fatalf("InterbankTransfer() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerReferenceNo", "beneficiaryAccountNo", "beneficiaryAccountName", "beneficiaryBankCode", "sourceAccountNo", "transactionDate"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	wantAmount := map[string]any{"value": "", "currency": ""}
	if !reflect.DeepEqual(got["amount"], wantAmount) {
		t.Errorf(`wire body["amount"] = %v, want %v (Amount is a plain non-pointer struct field, always present with all its own subfields)`, got["amount"], wantAmount)
	}
}

func TestInterbankTransfer_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank"
	tr := &snap.Transport{}
	_, err := InterbankTransfer(context.Background(), tr, hb, InterbankTransferRequest{})
	if err == nil {
		t.Fatal("InterbankTransfer() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("InterbankTransfer() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestInterbankTransfer_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2001800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank"
	tr := &snap.Transport{}
	resp, err := InterbankTransfer(context.Background(), tr, hb, InterbankTransferRequest{})
	if err == nil {
		t.Fatalf("InterbankTransfer() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("InterbankTransfer() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestInterbankTransfer_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1234567890"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-interbank"
	tr := &snap.Transport{}
	resp, err := InterbankTransfer(context.Background(), tr, hb, InterbankTransferRequest{})
	if err == nil {
		t.Fatalf("InterbankTransfer() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
