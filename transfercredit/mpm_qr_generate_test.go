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

func TestGenerateQRMPM_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004700",
   "responseMessage":"Request has been processed successfully",
   "qrContent":"00020101021226610014ID.CO.QRIS.WWW",
   "qrUrl":"https://example.com/qr/abc",
   "qrImage":"base64imagedata",
   "redirectUrl":"https://example.com/redirect",
   "merchantName":"Toko Maju",
   "storeId":"STORE01",
   "terminalId":"TERM01"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &snap.Transport{}
	resp, err := GenerateQRMPM(context.Background(), tr, hb, GenerateQRMPMRequest{
		PartnerReferenceNo: "partner-ref-1",
		MerchantID:         "MERCH01",
	})
	if err != nil {
		t.Fatalf("GenerateQRMPM() error = %v", err)
	}

	want := GenerateQRMPMResponse{
		ResponseCode:    "2004700",
		ResponseMessage: "Request has been processed successfully",
		QRContent:       "00020101021226610014ID.CO.QRIS.WWW",
		QRURL:           "https://example.com/qr/abc",
		QRImage:         "base64imagedata",
		RedirectURL:     "https://example.com/redirect",
		MerchantName:    "Toko Maju",
		StoreID:         "STORE01",
		TerminalID:      "TERM01",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("GenerateQRMPM() = %+v, want %+v", resp, want)
	}
}

func TestGenerateQRMPM_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &snap.Transport{}
	req := GenerateQRMPMRequest{
		PartnerReferenceNo: "partner-ref-1",
		Amount:             &snap.Money{Value: "50000.00", Currency: "IDR"},
		FeeAmount:          &snap.Money{Value: "500.00", Currency: "IDR"},
		MerchantID:         "MERCH01",
		SubMerchantID:      "SUBMERCH01",
		StoreID:            "STORE01",
		TerminalID:         "TERM01",
		ValidityPeriod:     "2020-12-20T10:00:00+07:00",
	}
	if _, err := GenerateQRMPM(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("GenerateQRMPM() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"partnerReferenceNo": "partner-ref-1",
		"amount":             map[string]any{"value": "50000.00", "currency": "IDR"},
		"feeAmount":          map[string]any{"value": "500.00", "currency": "IDR"},
		"merchantId":         "MERCH01",
		"subMerchantId":      "SUBMERCH01",
		"storeId":            "STORE01",
		"terminalId":         "TERM01",
		"validityPeriod":     "2020-12-20T10:00:00+07:00",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestGenerateQRMPM_AllFieldsOmittedWhenUnset pins that every request
// field is Optional (no field in this request is documented Mandatory,
// unusual for the package but true per research §5.9 line 192) — a
// zero-value request serializes to an empty JSON object.
func TestGenerateQRMPM_AllFieldsOmittedWhenUnset(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &snap.Transport{}
	if _, err := GenerateQRMPM(context.Background(), tr, hb, GenerateQRMPMRequest{}); err != nil {
		t.Fatalf("GenerateQRMPM() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("wire body = %v, want empty object (every field Optional)", got)
	}
}

func TestGenerateQRMPM_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004700","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &snap.Transport{}
	_, err := GenerateQRMPM(context.Background(), tr, hb, GenerateQRMPMRequest{})
	if err == nil {
		t.Fatal("GenerateQRMPM() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("GenerateQRMPM() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestGenerateQRMPM_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &snap.Transport{}
	resp, err := GenerateQRMPM(context.Background(), tr, hb, GenerateQRMPMRequest{})
	if err == nil {
		t.Fatalf("GenerateQRMPM() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("GenerateQRMPM() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestGenerateQRMPM_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"qrUrl":"https://example.com/qr/abc"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &snap.Transport{}
	resp, err := GenerateQRMPM(context.Background(), tr, hb, GenerateQRMPMRequest{})
	if err == nil {
		t.Fatalf("GenerateQRMPM() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
