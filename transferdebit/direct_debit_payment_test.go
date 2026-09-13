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
	"sync/atomic"
	"testing"

	snap "github.com/koriebruh/go-snap-bi"
	"github.com/koriebruh/go-snap-bi/internal/snaptest"
)

// TestDirectDebitPaymentTypes_FieldCounts guards against a field
// silently added to any of these types without updating the
// wire-assertion tests below.
func TestDirectDebitPaymentTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(DirectDebitPaymentURLParam{}).NumField(); n != 3 {
		t.Errorf("DirectDebitPaymentURLParam has %d fields, want 3", n)
	}
	if n := reflect.TypeOf(DirectDebitPayOptionDetail{}).NumField(); n != 7 {
		t.Errorf("DirectDebitPayOptionDetail has %d fields, want 7", n)
	}
	if n := reflect.TypeOf(DirectDebitPaymentRequest{}).NumField(); n != 18 {
		t.Errorf("DirectDebitPaymentRequest has %d fields, want 18", n)
	}
	if n := reflect.TypeOf(DirectDebitPaymentResponse{}).NumField(); n != 8 {
		t.Errorf("DirectDebitPaymentResponse has %d fields, want 8", n)
	}
}

func TestDirectDebitPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2005400",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"REF001",
   "partnerReferenceNo":"PARTNER001",
   "approvalCode":"APPROVAL001",
   "appRedirectUrl":"https://app.example.com/redirect",
   "webRedirectUrl":"https://web.example.com/redirect",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/payment-host-to-host"
	tr := &snap.Transport{}
	resp, err := DirectDebitPayment(context.Background(), tr, hb, DirectDebitPaymentRequest{
		PartnerReferenceNo: "PARTNER001",
	})
	if err != nil {
		t.Fatalf("DirectDebitPayment() error = %v", err)
	}

	want := DirectDebitPaymentResponse{
		ResponseCode:       "2005400",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "REF001",
		PartnerReferenceNo: "PARTNER001",
		ApprovalCode:       "APPROVAL001",
		AppRedirectURL:     "https://app.example.com/redirect",
		WebRedirectURL:     "https://web.example.com/redirect",
		AdditionalInfo:     json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("DirectDebitPayment() = %+v, want %+v", resp, want)
	}
}

func TestDirectDebitPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/payment-host-to-host"
	tr := &snap.Transport{}
	req := DirectDebitPaymentRequest{
		PartnerReferenceNo: "PARTNER001",
		BankCardToken:      "CARDTOKEN01",
		ChargeToken:        "CHARGETOKEN01",
		OTP:                "123456",
		OTPTrxCode:         "01",
		MerchantID:         "MERCH01",
		TerminalID:         "TERM01",
		JourneyID:          "JOURNEY01",
		SubMerchantID:      "SUBMERCH01",
		Amount:             &snap.Money{Value: "50000.00", Currency: "IDR"},
		URLParams: []DirectDebitPaymentURLParam{
			{URL: "https://example.com/return", Type: "PAY_RETURN", IsDeeplink: "N"},
			{URL: "https://example.com/notify", Type: "PAY_NOTIFY", IsDeeplink: "Y"},
		},
		ExternalStoreID:    "STORE01",
		ValidUpTo:          "2026-09-12T10:00:00+07:00",
		PointOfInitiation:  "MOBILE",
		FeeType:            "01",
		DisabledPayMethods: "CARD",
		PayOptionDetails: []DirectDebitPayOptionDetail{
			{
				PayMethod:      "CARD",
				PayOption:      "CREDIT",
				TransAmount:    &snap.Money{Value: "50000.00", Currency: "IDR"},
				FeeAmount:      &snap.Money{Value: "1000.00", Currency: "IDR"},
				CardToken:      "CARDTOKEN02",
				MerchantToken:  "MERCHTOKEN01",
				AdditionalInfo: json.RawMessage(`{"item-note":"item-value"}`),
			},
		},
		AdditionalInfo: json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := DirectDebitPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("DirectDebitPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"partnerReferenceNo": "PARTNER001",
		"bankCardToken":      "CARDTOKEN01",
		"chargeToken":        "CHARGETOKEN01",
		"otp":                "123456",
		"otpTrxCode":         "01",
		"merchantId":         "MERCH01",
		"terminalId":         "TERM01",
		"journeyId":          "JOURNEY01",
		"subMerchantId":      "SUBMERCH01",
		"amount":             map[string]any{"value": "50000.00", "currency": "IDR"},
		"urlParams": []any{
			map[string]any{"url": "https://example.com/return", "type": "PAY_RETURN", "isDeeplink": "N"},
			map[string]any{"url": "https://example.com/notify", "type": "PAY_NOTIFY", "isDeeplink": "Y"},
		},
		"externalStoreId":    "STORE01",
		"validUpTo":          "2026-09-12T10:00:00+07:00",
		"pointOfInitiation":  "MOBILE",
		"feeType":            "01",
		"disabledPayMethods": "CARD",
		"payOptionDetails": []any{
			map[string]any{
				"payMethod":      "CARD",
				"payOption":      "CREDIT",
				"transAmount":    map[string]any{"value": "50000.00", "currency": "IDR"},
				"feeAmount":      map[string]any{"value": "1000.00", "currency": "IDR"},
				"cardToken":      "CARDTOKEN02",
				"merchantToken":  "MERCHTOKEN01",
				"additionalInfo": map[string]any{"item-note": "item-value"},
			},
		},
		"additionalInfo": map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

func TestDirectDebitPayment_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2005400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/payment-host-to-host"
	tr := &snap.Transport{}
	_, err := DirectDebitPayment(context.Background(), tr, hb, DirectDebitPaymentRequest{
		PartnerReferenceNo: "PARTNER001",
		AdditionalInfo:     json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("DirectDebitPayment() error = nil, want non-nil for malformed AdditionalInfo JSON")
	}
	if requested.Load() {
		t.Error("DirectDebitPayment() sent an HTTP request despite a request-encoding failure")
	}
}

// TestDirectDebitPayment_MandatoryFieldAlwaysSerialized pins that
// PartnerReferenceNo — the only request field without omitempty — is
// always present on the wire, even as "", and every other field is
// omitted when unset.
func TestDirectDebitPayment_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/payment-host-to-host"
	tr := &snap.Transport{}
	if _, err := DirectDebitPayment(context.Background(), tr, hb, DirectDebitPaymentRequest{}); err != nil {
		t.Fatalf("DirectDebitPayment() error = %v", err)
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

// TestDirectDebitPaymentURLParam_FieldsHaveNoOmitempty pins that URL,
// Type, and IsDeeplink — all Mandatory per research §5.1 line 114 —
// always serialize, even from a zero-value item. The round-trip test
// above always populates every item field, so this is the only guard
// against a tag silently gaining omitempty.
func TestDirectDebitPaymentURLParam_FieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(DirectDebitPaymentURLParam{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value DirectDebitPaymentURLParam: %v", err)
	}
	for _, key := range []string{"url", "type", "isDeeplink"} {
		if _, ok := got[key]; !ok {
			t.Errorf(`marshaled zero-value DirectDebitPaymentURLParam missing %q key; want it always present`, key)
		}
	}
}

// TestDirectDebitPayOptionDetail_MandatoryFieldsHaveNoOmitempty pins
// that PayMethod and PayOption — Mandatory per research §5.1 line 118
// — always serialize, even from a zero-value item.
func TestDirectDebitPayOptionDetail_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(DirectDebitPayOptionDetail{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value DirectDebitPayOptionDetail: %v", err)
	}
	for _, key := range []string{"payMethod", "payOption"} {
		if _, ok := got[key]; !ok {
			t.Errorf(`marshaled zero-value DirectDebitPayOptionDetail missing %q key; want it always present`, key)
		}
	}
}

// TestDirectDebitPaymentResponse_ZeroValueOmitsOptionalFields pins that
// a zero-value response marshals to just the two envelope fields.
func TestDirectDebitPaymentResponse_ZeroValueOmitsOptionalFields(t *testing.T) {
	b, err := json.Marshal(DirectDebitPaymentResponse{})
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

func TestDirectDebitPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4005400","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/payment-host-to-host"
	tr := &snap.Transport{}
	_, err := DirectDebitPayment(context.Background(), tr, hb, DirectDebitPaymentRequest{})
	if err == nil {
		t.Fatal("DirectDebitPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("DirectDebitPayment() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestDirectDebitPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2005400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/payment-host-to-host"
	tr := &snap.Transport{}
	resp, err := DirectDebitPayment(context.Background(), tr, hb, DirectDebitPaymentRequest{})
	if err == nil {
		t.Fatalf("DirectDebitPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("DirectDebitPayment() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestDirectDebitPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"REF001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/payment-host-to-host"
	tr := &snap.Transport{}
	resp, err := DirectDebitPayment(context.Background(), tr, hb, DirectDebitPaymentRequest{})
	if err == nil {
		t.Fatalf("DirectDebitPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
