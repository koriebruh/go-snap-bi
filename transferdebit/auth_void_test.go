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

func TestAuthVoidTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(AuthVoidRequest{}).NumField(); n != 9 {
		t.Errorf("AuthVoidRequest has %d fields, want 9", n)
	}
	if n := reflect.TypeOf(AuthVoidResponse{}).NumField(); n != 9 {
		t.Errorf("AuthVoidResponse has %d fields, want 9", n)
	}
}

func TestAuthVoidRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "merchantId":"MERCHANT001",
   "subMerchantId":"SUBMERCHANT001",
   "voidAmount":{"value":"25000.00","currency":"IDR"},
   "partnerVoidNo":"PVOID001",
   "voidRemainingAmount":"TRUE",
   "reason":"customer cancelled",
   "additionalInfo":{"note":"req-value"}
}`
	var got AuthVoidRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthVoidRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		SubMerchantID:              "SUBMERCHANT001",
		VoidAmount:                 &snap.Money{Value: "25000.00", Currency: "IDR"},
		PartnerVoidNo:              "PVOID001",
		VoidRemainingAmount:        "TRUE",
		Reason:                     "customer cancelled",
		AdditionalInfo:             json.RawMessage(`{"note":"req-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}

	b, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatalf("decode marshaled request: %v", err)
	}
	wantWire := map[string]any{
		"originalReferenceNo":        "ref-1",
		"originalPartnerReferenceNo": "partner-ref-1",
		"merchantId":                 "MERCHANT001",
		"subMerchantId":              "SUBMERCHANT001",
		"voidAmount":                 map[string]any{"value": "25000.00", "currency": "IDR"},
		"partnerVoidNo":              "PVOID001",
		"voidRemainingAmount":        "TRUE",
		"reason":                     "customer cancelled",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestAuthVoidRequest_MandatoryFieldsHaveNoOmitempty pins that
// OriginalReferenceNo, OriginalPartnerReferenceNo, MerchantID, and
// PartnerVoidNo — the fields without omitempty — always serialize,
// even from a zero-value request.
func TestAuthVoidRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthVoidRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{
		"originalReferenceNo":        "",
		"originalPartnerReferenceNo": "",
		"merchantId":                 "",
		"partnerVoidNo":              "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional and unset)", got, want)
	}
}

func TestAuthVoidResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2006700",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "voidNo":"VOID001",
   "partnerVoidNo":"PVOID001",
   "voidAmount":{"value":"25000.00","currency":"IDR"},
   "voidTime":"2026-09-13T10:00:00+07:00",
   "additionalInfo":{"note":"resp-value"}
}`
	var got AuthVoidResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthVoidResponse{
		ResponseCode:               "2006700",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		VoidNo:                     "VOID001",
		PartnerVoidNo:              "PVOID001",
		VoidAmount:                 snap.Money{Value: "25000.00", Currency: "IDR"},
		VoidTime:                   "2026-09-13T10:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestAuthVoidResponse_MandatoryFieldsHaveNoOmitempty pins
// ResponseCode, ResponseMessage, PartnerVoidNo, and VoidAmount (a
// plain struct, always present) as always-serializing.
func TestAuthVoidResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthVoidResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{
		"responseCode":    "",
		"responseMessage": "",
		"partnerVoidNo":   "",
		"voidAmount":      map[string]any{"value": "", "currency": ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestAuthVoid_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := AuthVoidRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerVoidNo:              "PVOID001",
		AdditionalInfo:             json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}

func TestAuthVoid_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2006700",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "voidNo":"VOID001",
   "partnerVoidNo":"PVOID001",
   "voidAmount":{"value":"25000.00","currency":"IDR"},
   "voidTime":"2026-09-13T10:00:00+07:00",
   "additionalInfo":{"note":"resp-value"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void"
	tr := &snap.Transport{}
	resp, err := AuthVoid(context.Background(), tr, hb, AuthVoidRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerVoidNo:              "PVOID001",
	})
	if err != nil {
		t.Fatalf("AuthVoid() error = %v", err)
	}

	want := AuthVoidResponse{
		ResponseCode:               "2006700",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		VoidNo:                     "VOID001",
		PartnerVoidNo:              "PVOID001",
		VoidAmount:                 snap.Money{Value: "25000.00", Currency: "IDR"},
		VoidTime:                   "2026-09-13T10:00:00+07:00",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AuthVoid() = %+v, want %+v", resp, want)
	}
}

func TestAuthVoid_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2006700","responseMessage":"ok","partnerVoidNo":"PVOID001","voidAmount":{"value":"","currency":""}}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void"
	tr := &snap.Transport{}
	req := AuthVoidRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerVoidNo:              "PVOID001",
		VoidAmount:                 &snap.Money{Value: "25000.00", Currency: "IDR"},
	}
	if _, err := AuthVoid(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AuthVoid() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerVoidNo"] != "PVOID001" {
		t.Errorf(`wire body["partnerVoidNo"] = %v, want "PVOID001"`, got["partnerVoidNo"])
	}
	amount, ok := got["voidAmount"].(map[string]any)
	if !ok {
		t.Fatalf(`wire body["voidAmount"] = %v, want an object`, got["voidAmount"])
	}
	if amount["value"] != "25000.00" {
		t.Errorf(`wire body["voidAmount"]["value"] = %v, want "25000.00"`, amount["value"])
	}
}

func TestAuthVoid_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4006700","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void"
	tr := &snap.Transport{}
	_, err := AuthVoid(context.Background(), tr, hb, AuthVoidRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerVoidNo:              "PVOID001",
	})
	if err == nil {
		t.Fatal("AuthVoid() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("AuthVoid() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestAuthVoid_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2006700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void"
	tr := &snap.Transport{}
	resp, err := AuthVoid(context.Background(), tr, hb, AuthVoidRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerVoidNo:              "PVOID001",
	})
	if err == nil {
		t.Fatalf("AuthVoid() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("AuthVoid() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestAuthVoid_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"voidNo":"VOID001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/auth/void"
	tr := &snap.Transport{}
	resp, err := AuthVoid(context.Background(), tr, hb, AuthVoidRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		MerchantID:                 "MERCHANT001",
		PartnerVoidNo:              "PVOID001",
	})
	if err == nil {
		t.Fatalf("AuthVoid() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
