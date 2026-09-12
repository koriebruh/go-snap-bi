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

func TestDecodeQRMPM_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004800",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"ref-1",
   "redirectUrl":"https://example.com/redirect",
   "merchantName":"Toko Maju",
   "merchantCategory":"5411",
   "merchantLocation":"Jakarta",
   "merchantInfos":[
     {"merchantPAN":"9360001234567890","acquirerName":"Bank ABC"}
   ],
   "transactionAmount":{"value":"75000.00","currency":"IDR"},
   "feeAmount":{"value":"1000.00","currency":"IDR"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &Transport{}
	resp, err := DecodeQRMPM(context.Background(), tr, hb, DecodeQRMPMRequest{
		QRContent: "00020101021226610014ID.CO.QRIS.WWW",
		ScanTime:  "2020-12-20T10:00:00+07:00",
	})
	if err != nil {
		t.Fatalf("DecodeQRMPM() error = %v", err)
	}

	want := DecodeQRMPMResponse{
		ResponseCode:     "2004800",
		ResponseMessage:  "Request has been processed successfully",
		ReferenceNo:      "ref-1",
		RedirectUrl:      "https://example.com/redirect",
		MerchantName:     "Toko Maju",
		MerchantCategory: "5411",
		MerchantLocation: "Jakarta",
		MerchantInfos: []MPMMerchantInfo{
			{MerchantPAN: json.RawMessage(`"9360001234567890"`), AcquirerName: "Bank ABC"},
		},
		TransactionAmount: &Money{Value: "75000.00", Currency: "IDR"},
		FeeAmount:         &Money{Value: "1000.00", Currency: "IDR"},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("DecodeQRMPM() = %+v, want %+v", resp, want)
	}
}

func TestDecodeQRMPM_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004800","responseMessage":"ok","merchantInfos":[]}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &Transport{}
	req := DecodeQRMPMRequest{
		QRContent: "00020101021226610014ID.CO.QRIS.WWW",
		ScanTime:  "2020-12-20T10:00:00+07:00",
		Amount:    &Money{Value: "75000.00", Currency: "IDR"},
	}
	if _, err := DecodeQRMPM(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("DecodeQRMPM() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["qrContent"] != "00020101021226610014ID.CO.QRIS.WWW" {
		t.Errorf(`wire body["qrContent"] = %v, want %q`, got["qrContent"], "00020101021226610014ID.CO.QRIS.WWW")
	}
	if got["scanTime"] != "2020-12-20T10:00:00+07:00" {
		t.Errorf(`wire body["scanTime"] = %v, want "2020-12-20T10:00:00+07:00"`, got["scanTime"])
	}
}

// TestDecodeQRMPM_MandatoryFieldsAlwaysSerialized pins that QRContent
// and ScanTime — the request fields without omitempty — are always
// present on the wire, even as "".
func TestDecodeQRMPM_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004800","responseMessage":"ok","merchantInfos":[]}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &Transport{}
	if _, err := DecodeQRMPM(context.Background(), tr, hb, DecodeQRMPMRequest{}); err != nil {
		t.Fatalf("DecodeQRMPM() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"qrContent", "scanTime"} {
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

// TestDecodeQRMPMResponse_MerchantInfosHasNoOmitempty pins that
// MerchantInfos — Mandatory per research §5.9 line 194 — has no
// omitempty tag, matching the package's mandatory-array convention
// (BulkObject, Phase 12/18): a nil slice still marshals as the
// "merchantInfos" key (as JSON null), not an omitted key.
func TestDecodeQRMPMResponse_MerchantInfosHasNoOmitempty(t *testing.T) {
	b, err := json.Marshal(DecodeQRMPMResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	if _, ok := got["merchantInfos"]; !ok {
		t.Fatal(`marshaled zero-value response missing "merchantInfos" key; MerchantInfos lacks omitempty and must always be present`)
	}
}

// TestMPMMerchantInfo_MerchantPANRoundTripsQuotedString pins the
// ambiguous-type handling of merchantPAN: documented Numeric(19) but
// quoted on the wire, so json.RawMessage must preserve the surrounding
// quotes exactly, not unwrap them into a bare number.
func TestMPMMerchantInfo_MerchantPANRoundTripsQuotedString(t *testing.T) {
	info := MPMMerchantInfo{
		MerchantPAN:  json.RawMessage(`"9360001234567890"`),
		AcquirerName: "Bank ABC",
	}
	b, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal(%+v) error = %v", info, err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled MPMMerchantInfo: %v", err)
	}
	if got["merchantPAN"] != "9360001234567890" {
		t.Errorf(`marshaled merchantPAN = %v (%T), want quoted string "9360001234567890"`, got["merchantPAN"], got["merchantPAN"])
	}
}

func TestDecodeQRMPM_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &Transport{}
	_, err := DecodeQRMPM(context.Background(), tr, hb, DecodeQRMPMRequest{})
	if err == nil {
		t.Fatal("DecodeQRMPM() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("DecodeQRMPM() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestDecodeQRMPM_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &Transport{}
	resp, err := DecodeQRMPM(context.Background(), tr, hb, DecodeQRMPMRequest{})
	if err == nil {
		t.Fatalf("DecodeQRMPM() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("DecodeQRMPM() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestDecodeQRMPM_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"ref-1"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &Transport{}
	resp, err := DecodeQRMPM(context.Background(), tr, hb, DecodeQRMPMRequest{})
	if err == nil {
		t.Fatalf("DecodeQRMPM() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
