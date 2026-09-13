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

func TestTransactionStatusInquiryBank_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003600",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "originalExternalId":"ext-1",
   "serviceCode":"17",
   "transactionDate":"2020-12-20T10:00:00+07:00",
   "amount":{"value":"100000.00","currency":"IDR"},
   "beneficiaryAccountNo":"1122334455",
   "beneficiaryBankCode":"014",
   "previousResponseCode":"2001700",
   "referenceNumber":"refnum-1",
   "sourceAccountNo":"9988776655",
   "transactionId":"TX000001",
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-status-inquiry-bank"
	tr := &snap.Transport{}
	resp, err := TransactionStatusInquiryBank(context.Background(), tr, hb, TransactionStatusInquiryBankRequest{
		ServiceCode: "17",
	})
	if err != nil {
		t.Fatalf("TransactionStatusInquiryBank() error = %v", err)
	}

	want := TransactionStatusInquiryBankResponse{
		ResponseCode:               "2003600",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		ServiceCode:                "17",
		TransactionDate:            "2020-12-20T10:00:00+07:00",
		Amount:                     &snap.Money{Value: "100000.00", Currency: "IDR"},
		BeneficiaryAccountNo:       "1122334455",
		BeneficiaryBankCode:        "014",
		PreviousResponseCode:       "2001700",
		ReferenceNumber:            "refnum-1",
		SourceAccountNo:            "9988776655",
		TransactionID:              "TX000001",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("TransactionStatusInquiryBank() = %+v, want %+v", resp, want)
	}
}

func TestTransactionStatusInquiryBank_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-status-inquiry-bank"
	tr := &snap.Transport{}
	req := TransactionStatusInquiryBankRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		ServiceCode:                "17",
		Amount:                     &snap.Money{Value: "100000.00", Currency: "IDR"},
	}
	if _, err := TransactionStatusInquiryBank(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("TransactionStatusInquiryBank() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["serviceCode"] != "17" {
		t.Errorf(`wire body["serviceCode"] = %v, want "17"`, got["serviceCode"])
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok {
		t.Fatalf(`wire body["amount"] = %v, want an object`, got["amount"])
	}
	if amount["value"] != "100000.00" {
		t.Errorf(`wire body["amount"]["value"] = %v, want "100000.00"`, amount["value"])
	}
}

// TestTransactionStatusInquiryBank_MandatoryFieldAlwaysSerialized pins
// that ServiceCode — the only request field without omitempty — is
// always present on the wire, even as "".
func TestTransactionStatusInquiryBank_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-status-inquiry-bank"
	tr := &snap.Transport{}
	if _, err := TransactionStatusInquiryBank(context.Background(), tr, hb, TransactionStatusInquiryBankRequest{}); err != nil {
		t.Fatalf("TransactionStatusInquiryBank() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	v, ok := got["serviceCode"]
	if !ok {
		t.Fatal(`wire body missing "serviceCode" key; want it always present, even as ""`)
	}
	if v != "" {
		t.Errorf(`wire body["serviceCode"] = %v, want ""`, v)
	}
	if _, ok := got["originalPartnerReferenceNo"]; ok {
		t.Error(`wire body has "originalPartnerReferenceNo" key, want it omitted (Optional here)`)
	}
}

func TestTransactionStatusInquiryBank_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003600","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-status-inquiry-bank"
	tr := &snap.Transport{}
	_, err := TransactionStatusInquiryBank(context.Background(), tr, hb, TransactionStatusInquiryBankRequest{})
	if err == nil {
		t.Fatal("TransactionStatusInquiryBank() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("TransactionStatusInquiryBank() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestTransactionStatusInquiryBank_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-status-inquiry-bank"
	tr := &snap.Transport{}
	resp, err := TransactionStatusInquiryBank(context.Background(), tr, hb, TransactionStatusInquiryBankRequest{})
	if err == nil {
		t.Fatalf("TransactionStatusInquiryBank() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("TransactionStatusInquiryBank() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestTransactionStatusInquiryBank_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1122334455"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transaction-status-inquiry-bank"
	tr := &snap.Transport{}
	resp, err := TransactionStatusInquiryBank(context.Background(), tr, hb, TransactionStatusInquiryBankRequest{})
	if err == nil {
		t.Fatalf("TransactionStatusInquiryBank() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
