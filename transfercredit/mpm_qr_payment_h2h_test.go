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

func TestQRMPMPaymentH2H_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2005000",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"ref-1",
   "transactionDate":"2020-12-20T10:00:00+07:00",
   "verificationId":"verification-token-64chars"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-payment"
	tr := &snap.Transport{}
	resp, err := QRMPMPaymentH2H(context.Background(), tr, hb, QRMPMPaymentH2HRequest{
		PartnerReferenceNo: "partner-ref-1",
	})
	if err != nil {
		t.Fatalf("QRMPMPaymentH2H() error = %v", err)
	}

	want := QRMPMPaymentH2HResponse{
		ResponseCode:    "2005000",
		ResponseMessage: "Request has been processed successfully",
		ReferenceNo:     "ref-1",
		TransactionDate: "2020-12-20T10:00:00+07:00",
		VerificationID:  "verification-token-64chars",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("QRMPMPaymentH2H() = %+v, want %+v", resp, want)
	}
}

func TestQRMPMPaymentH2H_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-payment"
	tr := &snap.Transport{}
	req := QRMPMPaymentH2HRequest{
		PartnerReferenceNo: "partner-ref-1",
		MerchantID:         "MERCH01",
		SubMerchantID:      "SUBMERCH01",
		Amount:             &snap.Money{Value: "25000.00", Currency: "IDR"},
		FeeAmount:          &snap.Money{Value: "500.00", Currency: "IDR"},
		OTP:                "12345678",
		VerificationID:     "verify-32chars",
	}
	if _, err := QRMPMPaymentH2H(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("QRMPMPaymentH2H() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"partnerReferenceNo": "partner-ref-1",
		"merchantId":         "MERCH01",
		"subMerchantId":      "SUBMERCH01",
		"amount":             map[string]any{"value": "25000.00", "currency": "IDR"},
		"feeAmount":          map[string]any{"value": "500.00", "currency": "IDR"},
		"otp":                "12345678",
		"verificationId":     "verify-32chars",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestQRMPMPaymentH2H_MandatoryFieldAlwaysSerialized pins that
// PartnerReferenceNo — the request field without omitempty — is
// always present on the wire, even as "", and that every other field
// (all Optional) is omitted when unset.
func TestQRMPMPaymentH2H_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-payment"
	tr := &snap.Transport{}
	if _, err := QRMPMPaymentH2H(context.Background(), tr, hb, QRMPMPaymentH2HRequest{}); err != nil {
		t.Fatalf("QRMPMPaymentH2H() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"partnerReferenceNo": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

func TestQRMPMPaymentH2H_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4005000","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-payment"
	tr := &snap.Transport{}
	_, err := QRMPMPaymentH2H(context.Background(), tr, hb, QRMPMPaymentH2HRequest{})
	if err == nil {
		t.Fatal("QRMPMPaymentH2H() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("QRMPMPaymentH2H() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestQRMPMPaymentH2H_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2005000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-payment"
	tr := &snap.Transport{}
	resp, err := QRMPMPaymentH2H(context.Background(), tr, hb, QRMPMPaymentH2HRequest{})
	if err == nil {
		t.Fatalf("QRMPMPaymentH2H() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("QRMPMPaymentH2H() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestQRMPMPaymentH2H_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"ref-1"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-payment"
	tr := &snap.Transport{}
	resp, err := QRMPMPaymentH2H(context.Background(), tr, hb, QRMPMPaymentH2HRequest{})
	if err == nil {
		t.Fatalf("QRMPMPaymentH2H() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
