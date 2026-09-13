package transferdebit

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

func TestCPMPaymentTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(CPMPaymentScannerInfo{}).NumField(); n != 4 {
		t.Errorf("CPMPaymentScannerInfo has %d fields, want 4", n)
	}
	if n := reflect.TypeOf(CPMPaymentRequest{}).NumField(); n != 16 {
		t.Errorf("CPMPaymentRequest has %d fields, want 16", n)
	}
	if n := reflect.TypeOf(CPMPaymentResponse{}).NumField(); n != 6 {
		t.Errorf("CPMPaymentResponse has %d fields, want 6", n)
	}
}

func TestCPMPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2006000",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"REF001",
   "partnerReferenceNo":"PARTNER001",
   "transactionDate":"2026-09-13T09:00:00+07:00",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-payment"
	tr := &snap.Transport{}
	resp, err := CPMPayment(context.Background(), tr, hb, CPMPaymentRequest{
		PartnerReferenceNo: "PARTNER001",
		QRContent:          "00020101021226610014ID.CO.QRIS.WWW",
		MerchantID:         "MERCH01",
	})
	if err != nil {
		t.Fatalf("CPMPayment() error = %v", err)
	}

	want := CPMPaymentResponse{
		ResponseCode:       "2006000",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "REF001",
		PartnerReferenceNo: "PARTNER001",
		TransactionDate:    "2026-09-13T09:00:00+07:00",
		AdditionalInfo:     json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CPMPayment() = %+v, want %+v", resp, want)
	}
}

// Items below is populated with an illustrative array shape as a
// stand-in payload — research documents the field only as an
// unstructured "Object" with no worked example, so this is not a
// spec-derived shape; json.RawMessage round-trips any valid JSON value
// regardless of shape, which is the point of using it here.
func TestCPMPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-payment"
	tr := &snap.Transport{}
	req := CPMPaymentRequest{
		PartnerReferenceNo: "PARTNER001",
		QRContent:          "00020101021226610014ID.CO.QRIS.WWW",
		Amount:             &snap.Money{Value: "50000.00", Currency: "IDR"},
		FeeAmount:          &snap.Money{Value: "1000.00", Currency: "IDR"},
		MerchantID:         "MERCH01",
		SubMerchantID:      "SUBMERCH01",
		Title:              "Coffee order",
		ExpiryTime:         "2026-09-13T10:00:00+07:00",
		Items:              json.RawMessage(`[{"name":"Latte","qty":1}]`),
		ExternalStoreID:    "STORE01",
		MerchantName:       "Merchant Name",
		MerchantLocation:   "Jakarta",
		AcquirerName:       "Acquirer Name",
		TerminalID:         "TERM01",
		ScannerInfo: &CPMPaymentScannerInfo{
			DeviceID:      "DEVICE01",
			DeviceVersion: "1.0.0",
			DeviceModel:   "Model X",
			DeviceIP:      "10.0.0.1",
		},
		AdditionalInfo: json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := CPMPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CPMPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"partnerReferenceNo": "PARTNER001",
		"qrContent":          "00020101021226610014ID.CO.QRIS.WWW",
		"amount":             map[string]any{"value": "50000.00", "currency": "IDR"},
		"feeAmount":          map[string]any{"value": "1000.00", "currency": "IDR"},
		"merchantId":         "MERCH01",
		"subMerchantId":      "SUBMERCH01",
		"title":              "Coffee order",
		"expiryTime":         "2026-09-13T10:00:00+07:00",
		"items":              []any{map[string]any{"name": "Latte", "qty": float64(1)}},
		"externalStoreId":    "STORE01",
		"merchantName":       "Merchant Name",
		"merchantLocation":   "Jakarta",
		"acquirerName":       "Acquirer Name",
		"terminalId":         "TERM01",
		"scannerInfo": map[string]any{
			"deviceId":      "DEVICE01",
			"deviceVersion": "1.0.0",
			"deviceModel":   "Model X",
			"deviceIp":      "10.0.0.1",
		},
		"additionalInfo": map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestCPMPayment_MandatoryFieldsAlwaysSerialized pins that
// PartnerReferenceNo, QRContent, and MerchantID — the request fields
// without omitempty — always serialize, even as "", and every other
// field is omitted when unset.
func TestCPMPayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-payment"
	tr := &snap.Transport{}
	if _, err := CPMPayment(context.Background(), tr, hb, CPMPaymentRequest{}); err != nil {
		t.Fatalf("CPMPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"partnerReferenceNo": "", "qrContent": "", "merchantId": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestCPMPaymentScannerInfo_ZeroValueOmitsAllFields pins that all four
// members of CPMPaymentScannerInfo are Optional, so a zero-value item
// marshals to an empty object.
func TestCPMPaymentScannerInfo_ZeroValueOmitsAllFields(t *testing.T) {
	b, err := json.Marshal(CPMPaymentScannerInfo{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value CPMPaymentScannerInfo: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("marshaled zero-value CPMPaymentScannerInfo = %v, want empty object (all fields Optional)", got)
	}
}

// TestCPMPaymentResponse_ZeroValueOmitsOptionalFields pins that a
// zero-value response marshals to just the two envelope fields.
func TestCPMPaymentResponse_ZeroValueOmitsOptionalFields(t *testing.T) {
	b, err := json.Marshal(CPMPaymentResponse{})
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

func TestCPMPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4006000","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-payment"
	tr := &snap.Transport{}
	_, err := CPMPayment(context.Background(), tr, hb, CPMPaymentRequest{})
	if err == nil {
		t.Fatal("CPMPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("CPMPayment() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestCPMPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2006000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-payment"
	tr := &snap.Transport{}
	resp, err := CPMPayment(context.Background(), tr, hb, CPMPaymentRequest{})
	if err == nil {
		t.Fatalf("CPMPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("CPMPayment() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestCPMPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"REF001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-payment"
	tr := &snap.Transport{}
	resp, err := CPMPayment(context.Background(), tr, hb, CPMPaymentRequest{})
	if err == nil {
		t.Fatalf("CPMPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
