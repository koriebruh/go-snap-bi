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

func TestQRMPMQueryPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2005100",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "originalExternalId":"ext-1",
   "serviceCode":"51",
   "transactionDate":"2020-12-20T10:00:00+07:00",
   "amount":{"value":"25000.00","currency":"IDR"},
   "beneficiaryAccountNo":"1122334455",
   "beneficiaryBankCode":"014",
   "previousResponseCode":"2005000",
   "referenceNumber":"refnum-1",
   "sourceAccountNo":"9988776655",
   "transactionId":"TX000001",
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "additionalInfo":{"channel":"mobilephone"},
   "paidTime":"2020-12-20T10:05:00+07:00",
   "terminalId":"TERM01"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-query"
	tr := &snap.Transport{}
	resp, err := QRMPMQueryPayment(context.Background(), tr, hb, QRMPMQueryPaymentRequest{
		ServiceCode: "51",
	})
	if err != nil {
		t.Fatalf("QRMPMQueryPayment() error = %v", err)
	}

	want := QRMPMQueryPaymentResponse{
		ResponseCode:               "2005100",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		ServiceCode:                "51",
		TransactionDate:            "2020-12-20T10:00:00+07:00",
		Amount:                     &snap.Money{Value: "25000.00", Currency: "IDR"},
		BeneficiaryAccountNo:       "1122334455",
		BeneficiaryBankCode:        "014",
		PreviousResponseCode:       "2005000",
		ReferenceNumber:            "refnum-1",
		SourceAccountNo:            "9988776655",
		TransactionID:              "TX000001",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
		PaidTime:                   "2020-12-20T10:05:00+07:00",
		TerminalID:                 "TERM01",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("QRMPMQueryPayment() = %+v, want %+v", resp, want)
	}
}

func TestQRMPMQueryPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005100","responseMessage":"ok","beneficiaryAccountNo":"","referenceNumber":"","sourceAccountNo":"","latestTransactionStatus":""}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-query"
	tr := &snap.Transport{}
	req := QRMPMQueryPaymentRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		ServiceCode:                "51",
		TransactionDate:            "2020-12-20T10:00:00+07:00",
		Amount:                     &snap.Money{Value: "25000.00", Currency: "IDR"},
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		ExternalStoreID:            "STORE01",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := QRMPMQueryPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("QRMPMQueryPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"originalPartnerReferenceNo": "partner-ref-1",
		"originalReferenceNo":        "ref-1",
		"originalExternalId":         "ext-1",
		"serviceCode":                "51",
		"transactionDate":            "2020-12-20T10:00:00+07:00",
		"amount":                     map[string]any{"value": "25000.00", "currency": "IDR"},
		"merchantId":                 "MERCH01",
		"subMerchantId":              "SUBMERCH01",
		"externalStoreId":            "STORE01",
		"additionalInfo":             map[string]any{"channel": "mobilephone"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestQRMPMQueryPayment_MandatoryFieldAlwaysSerialized pins that
// ServiceCode — the request field without omitempty — is always
// present on the wire, even as "".
func TestQRMPMQueryPayment_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005100","responseMessage":"ok","beneficiaryAccountNo":"","referenceNumber":"","sourceAccountNo":"","latestTransactionStatus":""}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-query"
	tr := &snap.Transport{}
	if _, err := QRMPMQueryPayment(context.Background(), tr, hb, QRMPMQueryPaymentRequest{}); err != nil {
		t.Fatalf("QRMPMQueryPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"serviceCode": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestQRMPMQueryPaymentResponse_AlwaysPresentFieldsHaveNoOmitempty
// pins that ResponseCode, ResponseMessage, BeneficiaryAccountNo,
// ReferenceNumber, SourceAccountNo, and LatestTransactionStatus — the
// only fields without omitempty, carried over from
// TransactionStatusInquiryBankResponse — are exactly the keys a
// zero-value marshal produces. An exact-map comparison (not just a
// presence check) also catches a mutation that dropped omitempty from
// any of the other, Optional fields.
func TestQRMPMQueryPaymentResponse_AlwaysPresentFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(QRMPMQueryPaymentResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{
		"responseCode":            "",
		"responseMessage":         "",
		"beneficiaryAccountNo":    "",
		"referenceNumber":         "",
		"sourceAccountNo":         "",
		"latestTransactionStatus": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestQRMPMQueryPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4005100","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-query"
	tr := &snap.Transport{}
	_, err := QRMPMQueryPayment(context.Background(), tr, hb, QRMPMQueryPaymentRequest{})
	if err == nil {
		t.Fatal("QRMPMQueryPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("QRMPMQueryPayment() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestQRMPMQueryPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2005100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-query"
	tr := &snap.Transport{}
	resp, err := QRMPMQueryPayment(context.Background(), tr, hb, QRMPMQueryPaymentRequest{})
	if err == nil {
		t.Fatalf("QRMPMQueryPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("QRMPMQueryPayment() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestQRMPMQueryPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1122334455"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-query"
	tr := &snap.Transport{}
	resp, err := QRMPMQueryPayment(context.Background(), tr, hb, QRMPMQueryPaymentRequest{})
	if err == nil {
		t.Fatalf("QRMPMQueryPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
