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

func TestDirectDebitBIFASTPaymentTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(DirectDebitBIFASTPaymentRequest{}).NumField(); n != 14 {
		t.Errorf("DirectDebitBIFASTPaymentRequest has %d fields, want 14", n)
	}
	if n := reflect.TypeOf(DirectDebitBIFASTPaymentResponse{}).NumField(); n != 5 {
		t.Errorf("DirectDebitBIFASTPaymentResponse has %d fields, want 5", n)
	}
}

func TestDirectDebitBIFASTPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2007100",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"REF001",
   "partnerReferenceNo":"PARTNER001",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-payment"
	tr := &Transport{}
	resp, err := DirectDebitBIFASTPayment(context.Background(), tr, hb, DirectDebitBIFASTPaymentRequest{
		PartnerReferenceNo:     "PARTNER001",
		CustomerReference:      "CUSTREF001",
		BeneficiaryAccountNo:   "9876543210987",
		BeneficiaryAccountName: "John Smith",
		TransactionDate:        "2026-09-13T09:00:00+07:00",
		BankCode:               "014",
		SourceAccountNo:        "1234567890123456789012345678901234",
		SourceAccountName:      "Jane Doe",
		EMandateReffID:         "EMANDATE001",
	})
	if err != nil {
		t.Fatalf("DirectDebitBIFASTPayment() error = %v", err)
	}

	want := DirectDebitBIFASTPaymentResponse{
		ResponseCode:       "2007100",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "REF001",
		PartnerReferenceNo: "PARTNER001",
		AdditionalInfo:     json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("DirectDebitBIFASTPayment() = %+v, want %+v", resp, want)
	}
}

func TestDirectDebitBIFASTPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2007100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-payment"
	tr := &Transport{}
	req := DirectDebitBIFASTPaymentRequest{
		PartnerReferenceNo:     "PARTNER001",
		Currency:               "IDR",
		CustomerReference:      "CUSTREF001",
		FeeType:                "01",
		Remark:                 "invoice payment",
		BeneficiaryAccountNo:   "9876543210987",
		BeneficiaryAccountName: "John Smith",
		TransactionDate:        "2026-09-13T09:00:00+07:00",
		BankCode:               "014",
		SourceAccountNo:        "1234567890123456789012345678901234",
		SourceAccountName:      "Jane Doe",
		Amount:                 &Money{Value: "50000.00", Currency: "IDR"},
		EMandateReffID:         "EMANDATE001",
		AdditionalInfo:         json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := DirectDebitBIFASTPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("DirectDebitBIFASTPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"partnerReferenceNo":     "PARTNER001",
		"currency":               "IDR",
		"customerReference":      "CUSTREF001",
		"feeType":                "01",
		"remark":                 "invoice payment",
		"beneficiaryAccountNo":   "9876543210987",
		"beneficiaryAccountName": "John Smith",
		"transactionDate":        "2026-09-13T09:00:00+07:00",
		"bankCode":               "014",
		"sourceAccountNo":        "1234567890123456789012345678901234",
		"sourceAccountName":      "Jane Doe",
		"amount":                 map[string]any{"value": "50000.00", "currency": "IDR"},
		"eMandateReffId":         "EMANDATE001",
		"additionalInfo":         map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestDirectDebitBIFASTPayment_MandatoryFieldsAlwaysSerialized pins
// that PartnerReferenceNo, CustomerReference, BeneficiaryAccountNo,
// BeneficiaryAccountName, TransactionDate, BankCode, SourceAccountNo,
// SourceAccountName, and EMandateReffID — the request fields without
// omitempty — always serialize, even as "", and every other field is
// omitted when unset.
func TestDirectDebitBIFASTPayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2007100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-payment"
	tr := &Transport{}
	if _, err := DirectDebitBIFASTPayment(context.Background(), tr, hb, DirectDebitBIFASTPaymentRequest{}); err != nil {
		t.Fatalf("DirectDebitBIFASTPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"partnerReferenceNo":     "",
		"customerReference":      "",
		"beneficiaryAccountNo":   "",
		"beneficiaryAccountName": "",
		"transactionDate":        "",
		"bankCode":               "",
		"sourceAccountNo":        "",
		"sourceAccountName":      "",
		"eMandateReffId":         "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestDirectDebitBIFASTPaymentResponse_ZeroValueOmitsOptionalFields
// pins that a zero-value response marshals to just the two envelope
// fields.
func TestDirectDebitBIFASTPaymentResponse_ZeroValueOmitsOptionalFields(t *testing.T) {
	b, err := json.Marshal(DirectDebitBIFASTPaymentResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{"responseCode": "", "responseMessage": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestDirectDebitBIFASTPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4007100","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-payment"
	tr := &Transport{}
	_, err := DirectDebitBIFASTPayment(context.Background(), tr, hb, DirectDebitBIFASTPaymentRequest{})
	if err == nil {
		t.Fatal("DirectDebitBIFASTPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("DirectDebitBIFASTPayment() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestDirectDebitBIFASTPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2007100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-payment"
	tr := &Transport{}
	resp, err := DirectDebitBIFASTPayment(context.Background(), tr, hb, DirectDebitBIFASTPaymentRequest{})
	if err == nil {
		t.Fatalf("DirectDebitBIFASTPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("DirectDebitBIFASTPayment() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestDirectDebitBIFASTPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"REF001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-payment"
	tr := &Transport{}
	resp, err := DirectDebitBIFASTPayment(context.Background(), tr, hb, DirectDebitBIFASTPaymentRequest{})
	if err == nil {
		t.Fatalf("DirectDebitBIFASTPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
