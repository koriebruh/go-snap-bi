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

func TestRequestForPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2001900",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-request-for-payment"
	tr := &snap.Transport{}
	resp, err := RequestForPayment(context.Background(), tr, hb, RequestForPaymentRequest{
		PartnerReferenceNo:     "2020102900000000000001",
		BankCode:               "014",
		BeneficiaryAccountNo:   "1234567890",
		BeneficiaryAccountName: "Jane Doe",
		ExpiredDatetime:        "2020-12-21T14:56:11+07:00",
		SourceAccountNo:        "9876543210",
		SourceAccountName:      "John Doe",
	})
	if err != nil {
		t.Fatalf("RequestForPayment() error = %v", err)
	}

	want := RequestForPaymentResponse{
		ResponseCode:       "2001900",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("RequestForPayment() = %+v, want %+v", resp, want)
	}
}

func TestRequestForPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-request-for-payment"
	tr := &snap.Transport{}
	req := RequestForPaymentRequest{
		PartnerReferenceNo:     "2020102900000000000001",
		BankCode:               "014",
		BeneficiaryAccountNo:   "1234567890",
		BeneficiaryAccountName: "Jane Doe",
		ExpiredDatetime:        "2020-12-21T14:56:11+07:00",
		SourceAccountNo:        "9876543210",
		SourceAccountName:      "John Doe",
		Amount:                 &snap.Money{Value: "50000.00", Currency: "IDR"},
		AdditionalInfo:         json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := RequestForPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("RequestForPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["bankCode"] != "014" {
		t.Errorf(`wire body["bankCode"] = %v, want "014"`, got["bankCode"])
	}
	if got["expiredDatetime"] != "2020-12-21T14:56:11+07:00" {
		t.Errorf(`wire body["expiredDatetime"] = %v, want "2020-12-21T14:56:11+07:00"`, got["expiredDatetime"])
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok || amount["value"] != "50000.00" || amount["currency"] != "IDR" {
		t.Errorf(`wire body["amount"] = %v, want {"value":"50000.00","currency":"IDR"}`, got["amount"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestRequestForPayment_AmountOmittedWhenNil pins that the Optional
// *snap.Money Amount field is dropped from the wire when nil, unlike the
// Mandatory plain-snap.Money Amount fields on other Trigger Transfer
// endpoints.
func TestRequestForPayment_AmountOmittedWhenNil(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-request-for-payment"
	tr := &snap.Transport{}
	if _, err := RequestForPayment(context.Background(), tr, hb, RequestForPaymentRequest{}); err != nil {
		t.Fatalf("RequestForPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if _, ok := got["amount"]; ok {
		t.Errorf(`wire body has "amount" key = %v, want it omitted when Amount is nil`, got["amount"])
	}
}

// TestRequestForPayment_MandatoryFieldsAlwaysSerialized pins that
// PartnerReferenceNo, BankCode, BeneficiaryAccountNo,
// BeneficiaryAccountName, ExpiredDatetime, SourceAccountNo, and
// SourceAccountName — the seven request fields without omitempty — are
// always present on the wire, even as their zero value.
func TestRequestForPayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-request-for-payment"
	tr := &snap.Transport{}
	if _, err := RequestForPayment(context.Background(), tr, hb, RequestForPaymentRequest{}); err != nil {
		t.Fatalf("RequestForPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerReferenceNo", "bankCode", "beneficiaryAccountNo", "beneficiaryAccountName", "expiredDatetime", "sourceAccountNo", "sourceAccountName"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
}

func TestRequestForPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001900","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-request-for-payment"
	tr := &snap.Transport{}
	_, err := RequestForPayment(context.Background(), tr, hb, RequestForPaymentRequest{})
	if err == nil {
		t.Fatal("RequestForPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("RequestForPayment() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestRequestForPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2001900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-request-for-payment"
	tr := &snap.Transport{}
	resp, err := RequestForPayment(context.Background(), tr, hb, RequestForPaymentRequest{})
	if err == nil {
		t.Fatalf("RequestForPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("RequestForPayment() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestRequestForPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"ref-1"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-request-for-payment"
	tr := &snap.Transport{}
	resp, err := RequestForPayment(context.Background(), tr, hb, RequestForPaymentRequest{})
	if err == nil {
		t.Fatalf("RequestForPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
