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

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &Transport{}
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
		QRUrl:           "https://example.com/qr/abc",
		QRImage:         "base64imagedata",
		RedirectUrl:     "https://example.com/redirect",
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

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &Transport{}
	req := GenerateQRMPMRequest{
		PartnerReferenceNo: "partner-ref-1",
		Amount:             &Money{Value: "50000.00", Currency: "IDR"},
		MerchantID:         "MERCH01",
		StoreID:            "STORE01",
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
	if got["merchantId"] != "MERCH01" {
		t.Errorf(`wire body["merchantId"] = %v, want "MERCH01"`, got["merchantId"])
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok || amount["value"] != "50000.00" {
		t.Errorf(`wire body["amount"] = %v, want {"value":"50000.00","currency":"IDR"}`, got["amount"])
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

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &Transport{}
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

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &Transport{}
	_, err := GenerateQRMPM(context.Background(), tr, hb, GenerateQRMPMRequest{})
	if err == nil {
		t.Fatal("GenerateQRMPM() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("GenerateQRMPM() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestGenerateQRMPM_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &Transport{}
	resp, err := GenerateQRMPM(context.Background(), tr, hb, GenerateQRMPMRequest{})
	if err == nil {
		t.Fatalf("GenerateQRMPM() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("GenerateQRMPM() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestGenerateQRMPM_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"qrUrl":"https://example.com/qr/abc"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-generate"
	tr := &Transport{}
	resp, err := GenerateQRMPM(context.Background(), tr, hb, GenerateQRMPMRequest{})
	if err == nil {
		t.Fatalf("GenerateQRMPM() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
