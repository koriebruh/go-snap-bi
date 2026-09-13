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

func TestTransferToOTCCreatePayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004400",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"ref-1",
   "transactionDate":"2020-12-20T10:00:00+07:00"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cashout"
	tr := &snap.Transport{}
	resp, err := TransferToOTCCreatePayment(context.Background(), tr, hb, TransferToOTCCreatePaymentRequest{
		PartnerReferenceNo: "partner-ref-1",
		CustomerNumber:     "98765",
		OTP:                "12345678",
		Amount:             snap.Money{Value: "100000.00", Currency: "IDR"},
	})
	if err != nil {
		t.Fatalf("TransferToOTCCreatePayment() error = %v", err)
	}

	want := TransferToOTCCreatePaymentResponse{
		ResponseCode:    "2004400",
		ResponseMessage: "Request has been processed successfully",
		ReferenceNo:     "ref-1",
		TransactionDate: "2020-12-20T10:00:00+07:00",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("TransferToOTCCreatePayment() = %+v, want %+v", resp, want)
	}
}

func TestTransferToOTCCreatePayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cashout"
	tr := &snap.Transport{}
	req := TransferToOTCCreatePaymentRequest{
		PartnerReferenceNo: "partner-ref-1",
		CustomerNumber:     "98765",
		OTP:                "12345678",
		Amount:             snap.Money{Value: "100000.00", Currency: "IDR"},
		FeeType:            "01",
	}
	if _, err := TransferToOTCCreatePayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("TransferToOTCCreatePayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["otp"] != "12345678" {
		t.Errorf(`wire body["otp"] = %v, want "12345678"`, got["otp"])
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok || amount["value"] != "100000.00" {
		t.Errorf(`wire body["amount"] = %v, want {"value":"100000.00","currency":"IDR"}`, got["amount"])
	}
	if got["feeType"] != "01" {
		t.Errorf(`wire body["feeType"] = %v, want "01"`, got["feeType"])
	}
}

// TestTransferToOTCCreatePayment_MandatoryFieldsAlwaysSerialized pins
// that PartnerReferenceNo, CustomerNumber, OTP, and Amount — the
// request fields without omitempty — are always present on the wire.
func TestTransferToOTCCreatePayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cashout"
	tr := &snap.Transport{}
	if _, err := TransferToOTCCreatePayment(context.Background(), tr, hb, TransferToOTCCreatePaymentRequest{}); err != nil {
		t.Fatalf("TransferToOTCCreatePayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerReferenceNo", "customerNumber", "otp"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	if _, ok := got["amount"].(map[string]any); !ok {
		t.Errorf(`wire body["amount"] = %v, want an object (mandatory nested snap.Money is a plain struct)`, got["amount"])
	}
	if _, ok := got["feeType"]; ok {
		t.Error(`wire body has "feeType" key, want it omitted (Optional here)`)
	}
}

func TestTransferToOTCCreatePayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004400","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cashout"
	tr := &snap.Transport{}
	_, err := TransferToOTCCreatePayment(context.Background(), tr, hb, TransferToOTCCreatePaymentRequest{})
	if err == nil {
		t.Fatal("TransferToOTCCreatePayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("TransferToOTCCreatePayment() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestTransferToOTCCreatePayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cashout"
	tr := &snap.Transport{}
	resp, err := TransferToOTCCreatePayment(context.Background(), tr, hb, TransferToOTCCreatePaymentRequest{})
	if err == nil {
		t.Fatalf("TransferToOTCCreatePayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("TransferToOTCCreatePayment() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestTransferToOTCCreatePayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"ref-1"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cashout"
	tr := &snap.Transport{}
	resp, err := TransferToOTCCreatePayment(context.Background(), tr, hb, TransferToOTCCreatePaymentRequest{})
	if err == nil {
		t.Fatalf("TransferToOTCCreatePayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
