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

func TestCPMGenerateQRTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(CPMGenerateQRRequest{}).NumField(); n != 6 {
		t.Errorf("CPMGenerateQRRequest has %d fields, want 6", n)
	}
	if n := reflect.TypeOf(CPMGenerateQRResponse{}).NumField(); n != 8 {
		t.Errorf("CPMGenerateQRResponse has %d fields, want 8", n)
	}
}

func TestCPMGenerateQR_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2005900",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"REF001",
   "partnerReferenceNo":"PARTNER001",
   "qrContent":"00020101021226610014ID.CO.QRIS.WWW",
   "qrUrl":"https://example.com/qr/REF001",
   "expiryTime":"2026-09-13T10:00:00+07:00",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-generate"
	tr := &Transport{}
	resp, err := CPMGenerateQR(context.Background(), tr, hb, CPMGenerateQRRequest{
		PartnerTrxDate: "2026-09-13T09:00:00+07:00",
	})
	if err != nil {
		t.Fatalf("CPMGenerateQR() error = %v", err)
	}

	want := CPMGenerateQRResponse{
		ResponseCode:       "2005900",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "REF001",
		PartnerReferenceNo: "PARTNER001",
		QRContent:          "00020101021226610014ID.CO.QRIS.WWW",
		QRURL:              "https://example.com/qr/REF001",
		ExpiryTime:         "2026-09-13T10:00:00+07:00",
		AdditionalInfo:     json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CPMGenerateQR() = %+v, want %+v", resp, want)
	}
}

func TestCPMGenerateQR_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005900","responseMessage":"ok","expiryTime":"2026-09-13T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-generate"
	tr := &Transport{}
	req := CPMGenerateQRRequest{
		PartnerReferenceNo: "PARTNER001",
		UserAccessToken:    "TOKEN001",
		MerchantID:         "MERCH01",
		SubMerchantID:      "SUBMERCH01",
		PartnerTrxDate:     "2026-09-13T09:00:00+07:00",
		AdditionalInfo:     json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := CPMGenerateQR(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CPMGenerateQR() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"partnerReferenceNo": "PARTNER001",
		"userAccessToken":    "TOKEN001",
		"merchantId":         "MERCH01",
		"subMerchantId":      "SUBMERCH01",
		"partnerTrxDate":     "2026-09-13T09:00:00+07:00",
		"additionalInfo":     map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestCPMGenerateQR_MandatoryFieldAlwaysSerialized pins that
// PartnerTrxDate — the only request field without omitempty — is
// always present on the wire, even as "", and every other field is
// omitted when unset.
func TestCPMGenerateQR_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005900","responseMessage":"ok","expiryTime":"2026-09-13T10:00:00+07:00"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-generate"
	tr := &Transport{}
	if _, err := CPMGenerateQR(context.Background(), tr, hb, CPMGenerateQRRequest{}); err != nil {
		t.Fatalf("CPMGenerateQR() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"partnerTrxDate": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestCPMGenerateQRResponse_MandatoryFieldAlwaysSerialized pins that
// ExpiryTime — the only response field without omitempty beyond the
// envelope — always serializes from a zero-value response.
func TestCPMGenerateQRResponse_MandatoryFieldAlwaysSerialized(t *testing.T) {
	b, err := json.Marshal(CPMGenerateQRResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{"responseCode": "", "responseMessage": "", "expiryTime": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestCPMGenerateQR_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4005900","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-generate"
	tr := &Transport{}
	_, err := CPMGenerateQR(context.Background(), tr, hb, CPMGenerateQRRequest{})
	if err == nil {
		t.Fatal("CPMGenerateQR() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("CPMGenerateQR() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestCPMGenerateQR_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2005900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-generate"
	tr := &Transport{}
	resp, err := CPMGenerateQR(context.Background(), tr, hb, CPMGenerateQRRequest{})
	if err == nil {
		t.Fatalf("CPMGenerateQR() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("CPMGenerateQR() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestCPMGenerateQR_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"REF001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-cpm-generate"
	tr := &Transport{}
	resp, err := CPMGenerateQR(context.Background(), tr, hb, CPMGenerateQRRequest{})
	if err == nil {
		t.Fatalf("CPMGenerateQR() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
