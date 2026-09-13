package transfercredit

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

func TestTransferToBankAccountInquiry_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004200",
   "responseMessage":"Request has been processed successfully",
   "accountType":"Savings",
   "beneficiaryAccountNumber":"1122334455",
   "beneficiaryAccountName":"Jane Doe",
   "beneficiaryBankCode":"014",
   "beneficiaryBankShortName":"BCA",
   "beneficiaryBankName":"Bank Central Asia",
   "amount":{"value":"100000.00","currency":"IDR"},
   "sessionId":"sess-1"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/bank-account-inquiry"
	tr := &snap.Transport{}
	resp, err := TransferToBankAccountInquiry(context.Background(), tr, hb, TransferToBankAccountInquiryRequest{
		CustomerNumber: "98765",
		Amount:         snap.Money{Value: "100000.00", Currency: "IDR"},
	})
	if err != nil {
		t.Fatalf("TransferToBankAccountInquiry() error = %v", err)
	}

	want := TransferToBankAccountInquiryResponse{
		ResponseCode:             "2004200",
		ResponseMessage:          "Request has been processed successfully",
		AccountType:              "Savings",
		BeneficiaryAccountNumber: "1122334455",
		BeneficiaryAccountName:   "Jane Doe",
		BeneficiaryBankCode:      "014",
		BeneficiaryBankShortName: "BCA",
		BeneficiaryBankName:      "Bank Central Asia",
		Amount:                   snap.Money{Value: "100000.00", Currency: "IDR"},
		SessionID:                "sess-1",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("TransferToBankAccountInquiry() = %+v, want %+v", resp, want)
	}
}

func TestTransferToBankAccountInquiry_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004200","responseMessage":"ok","beneficiaryAccountNumber":"1122334455","beneficiaryAccountName":"Jane Doe","amount":{"value":"0","currency":"IDR"}}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/bank-account-inquiry"
	tr := &snap.Transport{}
	req := TransferToBankAccountInquiryRequest{
		PartnerReferenceNo:       "partner-ref-1",
		CustomerNumber:           "98765",
		Amount:                   snap.Money{Value: "100000.00", Currency: "IDR"},
		BeneficiaryAccountNumber: "1122334455",
	}
	if _, err := TransferToBankAccountInquiry(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("TransferToBankAccountInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["CustomerNumber"] != "98765" {
		t.Errorf(`wire body["CustomerNumber"] (capital C) = %v, want "98765"`, got["CustomerNumber"])
	}
	if _, ok := got["customerNumber"]; ok {
		t.Error(`wire body has lowercase "customerNumber" key, want only capital-C "CustomerNumber" for this endpoint`)
	}
	if got["beneficiaryAccountNumber"] != "1122334455" {
		t.Errorf(`wire body["beneficiaryAccountNumber"] = %v, want "1122334455"`, got["beneficiaryAccountNumber"])
	}
}

// TestTransferToBankAccountInquiry_MandatoryFieldsAlwaysSerialized
// pins that CustomerNumber and Amount — the two request fields
// without omitempty — are always present on the wire. Amount is a
// mandatory nested object (plain snap.Money, not *snap.Money), so it always
// serializes as an object.
func TestTransferToBankAccountInquiry_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004200","responseMessage":"ok","beneficiaryAccountNumber":"1122334455","beneficiaryAccountName":"Jane Doe","amount":{"value":"0","currency":"IDR"}}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/bank-account-inquiry"
	tr := &snap.Transport{}
	if _, err := TransferToBankAccountInquiry(context.Background(), tr, hb, TransferToBankAccountInquiryRequest{}); err != nil {
		t.Fatalf("TransferToBankAccountInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	v, ok := got["CustomerNumber"]
	if !ok {
		t.Fatal(`wire body missing "CustomerNumber" key; want it always present, even as ""`)
	}
	if v != "" {
		t.Errorf(`wire body["CustomerNumber"] = %v, want ""`, v)
	}
	if _, ok := got["amount"].(map[string]any); !ok {
		t.Errorf(`wire body["amount"] = %v, want an object (mandatory nested snap.Money is a plain struct)`, got["amount"])
	}
	if _, ok := got["partnerReferenceNo"]; ok {
		t.Error(`wire body has "partnerReferenceNo" key, want it omitted (Optional here)`)
	}
}

func TestTransferToBankAccountInquiry_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004200","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/bank-account-inquiry"
	tr := &snap.Transport{}
	_, err := TransferToBankAccountInquiry(context.Background(), tr, hb, TransferToBankAccountInquiryRequest{})
	if err == nil {
		t.Fatal("TransferToBankAccountInquiry() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("TransferToBankAccountInquiry() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestTransferToBankAccountInquiry_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004200","responseMessage":"ok","beneficiaryAccountNumber":"1122334455","beneficiaryAccountName":"Jane Doe","amount":{"value":"0","currency":"IDR"}}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/bank-account-inquiry"
	tr := &snap.Transport{}
	resp, err := TransferToBankAccountInquiry(context.Background(), tr, hb, TransferToBankAccountInquiryRequest{})
	if err == nil {
		t.Fatalf("TransferToBankAccountInquiry() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("TransferToBankAccountInquiry() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestTransferToBankAccountInquiry_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNumber":"1122334455"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/bank-account-inquiry"
	tr := &snap.Transport{}
	resp, err := TransferToBankAccountInquiry(context.Background(), tr, hb, TransferToBankAccountInquiryRequest{})
	if err == nil {
		t.Fatalf("TransferToBankAccountInquiry() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
