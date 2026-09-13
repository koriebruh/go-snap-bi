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

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &snap.Transport{}
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
		RedirectURL:      "https://example.com/redirect",
		MerchantName:     "Toko Maju",
		MerchantCategory: "5411",
		MerchantLocation: "Jakarta",
		MerchantInfos: []MPMMerchantInfo{
			{MerchantPAN: json.RawMessage(`"9360001234567890"`), AcquirerName: "Bank ABC"},
		},
		TransactionAmount: &snap.Money{Value: "75000.00", Currency: "IDR"},
		FeeAmount:         &snap.Money{Value: "1000.00", Currency: "IDR"},
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

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &snap.Transport{}
	req := DecodeQRMPMRequest{
		PartnerReferenceNo: "partner-ref-1",
		QRContent:          "00020101021226610014ID.CO.QRIS.WWW",
		Amount:             &snap.Money{Value: "75000.00", Currency: "IDR"},
		MerchantID:         "MERCH01",
		SubMerchantID:      "SUBMERCH01",
		ScanTime:           "2020-12-20T10:00:00+07:00",
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
	want := map[string]any{
		"partnerReferenceNo": "partner-ref-1",
		"qrContent":          "00020101021226610014ID.CO.QRIS.WWW",
		"amount":             map[string]any{"value": "75000.00", "currency": "IDR"},
		"merchantId":         "MERCH01",
		"subMerchantId":      "SUBMERCH01",
		"scanTime":           "2020-12-20T10:00:00+07:00",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
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

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &snap.Transport{}
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
	if len(got) != 2 {
		t.Errorf(`wire body = %v, want exactly {"qrContent":"","scanTime":""} (every other field Optional and unset)`, got)
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

// TestMPMMerchantInfo_FieldsHaveNoOmitempty pins that both MerchantPAN
// and AcquirerName — Mandatory per research §5.9 line 194 — always
// serialize, even from a zero-value MPMMerchantInfo. Only the array
// itself (MerchantInfos) was previously pinned; this asymmetry in the
// same mandatory-array convention was unguarded.
func TestMPMMerchantInfo_FieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(MPMMerchantInfo{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value MPMMerchantInfo: %v", err)
	}
	for _, key := range []string{"merchantPAN", "acquirerName"} {
		if _, ok := got[key]; !ok {
			t.Errorf(`marshaled zero-value MPMMerchantInfo missing %q key; want it always present`, key)
		}
	}
}

// TestDecodeQRMPM_MerchantPANAcceptsBareNumber pins the other half of
// merchantPAN's ambiguous-type contract: documented Numeric(19), so a
// bare (unquoted) JSON number must also round-trip byte-for-byte,
// mirroring TestVAInquiryStatus_CustomerNoAcceptsBareNumber's precedent
// for the same ambiguous-type rule (opposite direction: customerNo is
// documented String with a bare number on the wire).
func TestDecodeQRMPM_MerchantPANAcceptsBareNumber(t *testing.T) {
	const bareNumber = `9360001234567890123`
	fixture := `{"responseCode":"2004800","responseMessage":"ok","merchantInfos":[{"merchantPAN":` + bareNumber + `,"acquirerName":"Bank ABC"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &snap.Transport{}
	resp, err := DecodeQRMPM(context.Background(), tr, hb, DecodeQRMPMRequest{
		QRContent: "00020101021226610014ID.CO.QRIS.WWW",
		ScanTime:  "2020-12-20T10:00:00+07:00",
	})
	if err != nil {
		t.Fatalf("DecodeQRMPM() error = %v", err)
	}
	if len(resp.MerchantInfos) != 1 {
		t.Fatalf("DecodeQRMPM() MerchantInfos = %v, want 1 entry", resp.MerchantInfos)
	}
	if string(resp.MerchantInfos[0].MerchantPAN) != bareNumber {
		t.Errorf("MerchantPAN = %s, want %s", resp.MerchantInfos[0].MerchantPAN, bareNumber)
	}
}

func TestDecodeQRMPM_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &snap.Transport{}
	_, err := DecodeQRMPM(context.Background(), tr, hb, DecodeQRMPMRequest{})
	if err == nil {
		t.Fatal("DecodeQRMPM() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("DecodeQRMPM() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestDecodeQRMPM_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &snap.Transport{}
	resp, err := DecodeQRMPM(context.Background(), tr, hb, DecodeQRMPMRequest{})
	if err == nil {
		t.Fatalf("DecodeQRMPM() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("DecodeQRMPM() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestDecodeQRMPM_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"ref-1"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-decode"
	tr := &snap.Transport{}
	resp, err := DecodeQRMPM(context.Background(), tr, hb, DecodeQRMPMRequest{})
	if err == nil {
		t.Fatalf("DecodeQRMPM() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
