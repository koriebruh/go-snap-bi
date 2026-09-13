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

func TestSKNBITransfer_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2002300",
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
   "transactionStatus":"00",
   "transactionStatusDesc":"Success",
   "beneficiaryAccountType":"D",
   "originatorInfos":[{"originatorCustomerNo":"cust-1","originatorCustomerName":"John Doe","originatorBankCode":"014"}],
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-skn"
	tr := &snap.Transport{}
	resp, err := SKNBITransfer(context.Background(), tr, hb, SKNBITransferRequest{
		PartnerReferenceNo:           "2020102900000000000001",
		Amount:                       snap.Money{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountNo:         "1234567890",
		BeneficiaryAccountName:       "Jane Doe",
		BeneficiaryBankCode:          "014",
		SourceAccountNo:              "9876543210",
		TransactionDate:              "2020-12-21T14:56:11+07:00",
		BeneficiaryCustomerResidence: "1",
		BeneficiaryCustomerType:      "1",
	})
	if err != nil {
		t.Fatalf("SKNBITransfer() error = %v", err)
	}

	want := SKNBITransferResponse{
		ResponseCode:           "2002300",
		ResponseMessage:        "Request has been processed successfully",
		ReferenceNo:            "2020102977770000000009",
		PartnerReferenceNo:     "2020102900000000000001",
		Amount:                 &snap.Money{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountNo:   "1234567890",
		Currency:               "IDR",
		CustomerReference:      "cust-ref-1",
		SourceAccountNo:        "9876543210",
		TransactionDate:        "2020-12-21T14:56:11+07:00",
		TraceNo:                "TRACE123456",
		TransactionStatus:      "00",
		TransactionStatusDesc:  "Success",
		BeneficiaryAccountType: "D",
		OriginatorInfos: []TransferOriginatorInfo{
			{OriginatorCustomerNo: "cust-1", OriginatorCustomerName: "John Doe", OriginatorBankCode: "014"},
		},
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("SKNBITransfer() = %+v, want %+v", resp, want)
	}
}

func TestSKNBITransfer_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-skn"
	tr := &snap.Transport{}
	req := SKNBITransferRequest{
		PartnerReferenceNo:           "2020102900000000000001",
		Amount:                       snap.Money{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountNo:         "1234567890",
		BeneficiaryAccountName:       "Jane Doe",
		BeneficiaryBankCode:          "014",
		SourceAccountNo:              "9876543210",
		TransactionDate:              "2020-12-21T14:56:11+07:00",
		BeneficiaryCustomerResidence: "1",
		BeneficiaryCustomerType:      "1",
		Kodepos:                      "12345",
		ReceiverPhone:                "0812345678",
		AdditionalInfo:               json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := SKNBITransfer(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("SKNBITransfer() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["beneficiaryCustomerResidence"] != "1" {
		t.Errorf(`wire body["beneficiaryCustomerResidence"] = %v, want "1"`, got["beneficiaryCustomerResidence"])
	}
	if got["beneficiaryCustomerType"] != "1" {
		t.Errorf(`wire body["beneficiaryCustomerType"] = %v, want "1"`, got["beneficiaryCustomerType"])
	}
	if got["kodepos"] != "12345" {
		t.Errorf(`wire body["kodepos"] = %v, want "12345"`, got["kodepos"])
	}
	if got["receiverPhone"] != "0812345678" {
		t.Errorf(`wire body["receiverPhone"] = %v, want "0812345678"`, got["receiverPhone"])
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok || amount["value"] != "50000.00" || amount["currency"] != "IDR" {
		t.Errorf(`wire body["amount"] = %v, want {"value":"50000.00","currency":"IDR"}`, got["amount"])
	}
}

// TestSKNBITransfer_MandatoryFieldsAlwaysSerialized pins that
// PartnerReferenceNo, Amount, BeneficiaryAccountNo,
// BeneficiaryAccountName, BeneficiaryBankCode, SourceAccountNo,
// TransactionDate, BeneficiaryCustomerResidence, and
// BeneficiaryCustomerType — the nine request fields without omitempty —
// are always present on the wire, even as their zero value.
func TestSKNBITransfer_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-skn"
	tr := &snap.Transport{}
	if _, err := SKNBITransfer(context.Background(), tr, hb, SKNBITransferRequest{}); err != nil {
		t.Fatalf("SKNBITransfer() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerReferenceNo", "beneficiaryAccountNo", "beneficiaryAccountName", "beneficiaryBankCode", "sourceAccountNo", "transactionDate", "beneficiaryCustomerResidence", "beneficiaryCustomerType"} {
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
		t.Errorf(`wire body["amount"] = %v, want %v`, got["amount"], wantAmount)
	}
}

func TestSKNBITransfer_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4002300","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-skn"
	tr := &snap.Transport{}
	_, err := SKNBITransfer(context.Background(), tr, hb, SKNBITransferRequest{})
	if err == nil {
		t.Fatal("SKNBITransfer() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("SKNBITransfer() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestSKNBITransfer_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2002300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-skn"
	tr := &snap.Transport{}
	resp, err := SKNBITransfer(context.Background(), tr, hb, SKNBITransferRequest{})
	if err == nil {
		t.Fatalf("SKNBITransfer() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("SKNBITransfer() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestSKNBITransfer_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1234567890"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-skn"
	tr := &snap.Transport{}
	resp, err := SKNBITransfer(context.Background(), tr, hb, SKNBITransferRequest{})
	if err == nil {
		t.Fatalf("SKNBITransfer() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
