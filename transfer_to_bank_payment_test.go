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

func TestTransferToBankPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004300",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"ref-1",
   "transactionDate":"2020-12-20T10:00:00+07:00",
   "referenceNumber":"REF993883"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/transfer-bank"
	tr := &Transport{}
	resp, err := TransferToBankPayment(context.Background(), tr, hb, TransferToBankPaymentRequest{
		PartnerReferenceNo:       "partner-ref-1",
		CustomerNumber:           "98765",
		BeneficiaryAccountNumber: "1122334455",
		Amount:                   Money{Value: "100000.00", Currency: "IDR"},
	})
	if err != nil {
		t.Fatalf("TransferToBankPayment() error = %v", err)
	}

	want := TransferToBankPaymentResponse{
		ResponseCode:    "2004300",
		ResponseMessage: "Request has been processed successfully",
		ReferenceNo:     "ref-1",
		TransactionDate: "2020-12-20T10:00:00+07:00",
		ReferenceNumber: "REF993883",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("TransferToBankPayment() = %+v, want %+v", resp, want)
	}
}

func TestTransferToBankPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004300","responseMessage":"ok","referenceNumber":"REF993883"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/transfer-bank"
	tr := &Transport{}
	req := TransferToBankPaymentRequest{
		PartnerReferenceNo:       "partner-ref-1",
		CustomerNumber:           "98765",
		AccountType:              "Savings",
		BeneficiaryAccountNumber: "1122334455",
		BeneficiaryBankCode:      "014",
		Amount:                   Money{Value: "100000.00", Currency: "IDR"},
		SessionID:                "sess-1",
		FeeType:                  "01",
	}
	if _, err := TransferToBankPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("TransferToBankPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["customerNumber"] != "98765" {
		t.Errorf(`wire body["customerNumber"] = %v, want "98765"`, got["customerNumber"])
	}
	if _, ok := got["CustomerNumber"]; ok {
		t.Error(`wire body has capital-C "CustomerNumber" key, want only lowercase "customerNumber" for this endpoint`)
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok || amount["value"] != "100000.00" {
		t.Errorf(`wire body["amount"] = %v, want {"value":"100000.00","currency":"IDR"}`, got["amount"])
	}
	if got["feeType"] != "01" {
		t.Errorf(`wire body["feeType"] = %v, want "01"`, got["feeType"])
	}
}

// TestTransferToBankPayment_MandatoryFieldsAlwaysSerialized pins that
// PartnerReferenceNo, CustomerNumber, BeneficiaryAccountNumber, and
// Amount — the request fields without omitempty — are always present
// on the wire.
func TestTransferToBankPayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004300","responseMessage":"ok","referenceNumber":"REF993883"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/transfer-bank"
	tr := &Transport{}
	if _, err := TransferToBankPayment(context.Background(), tr, hb, TransferToBankPaymentRequest{}); err != nil {
		t.Fatalf("TransferToBankPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerReferenceNo", "customerNumber", "beneficiaryAccountNumber"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	if _, ok := got["amount"].(map[string]any); !ok {
		t.Errorf(`wire body["amount"] = %v, want an object (mandatory nested Money is a plain struct)`, got["amount"])
	}
	if _, ok := got["feeType"]; ok {
		t.Error(`wire body has "feeType" key, want it omitted (Optional here)`)
	}
}

func TestTransferToBankPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004300","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/transfer-bank"
	tr := &Transport{}
	_, err := TransferToBankPayment(context.Background(), tr, hb, TransferToBankPaymentRequest{})
	if err == nil {
		t.Fatal("TransferToBankPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("TransferToBankPayment() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestTransferToBankPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004300","responseMessage":"ok","referenceNumber":"REF993883"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/transfer-bank"
	tr := &Transport{}
	resp, err := TransferToBankPayment(context.Background(), tr, hb, TransferToBankPaymentRequest{})
	if err == nil {
		t.Fatalf("TransferToBankPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("TransferToBankPayment() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestTransferToBankPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNumber":"REF993883"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/transfer-bank"
	tr := &Transport{}
	resp, err := TransferToBankPayment(context.Background(), tr, hb, TransferToBankPaymentRequest{})
	if err == nil {
		t.Fatalf("TransferToBankPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
