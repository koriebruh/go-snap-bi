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
	"sync/atomic"
	"testing"

	snap "github.com/koriebruh/go-snap-bi"
	"github.com/koriebruh/go-snap-bi/internal/snaptest"
)

func TestVAInquiryPaymentIntrabank_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003200",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountdata":{
      "partnerServiceId":"12345",
      "customerNo":12345678901234567890,
      "virtualAccountNo":"1234598765",
      "partnerReferenceNo":"ref-1",
      "sourceAccountNo":"1122334455",
      "sourceAccountType":"D",
      "productName":"Savings",
      "billAmountLabel":"Total",
      "billAmountValue":"100000.00",
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-payment-intrabank"
	tr := &snap.Transport{}
	resp, err := VAInquiryPaymentIntrabank(context.Background(), tr, hb, VAInquiryPaymentIntrabankRequest{
		PartnerServiceID: "12345",
		CustomerNo:       json.RawMessage(`12345678901234567890`),
		VirtualAccountNo: "1234598765",
	})
	if err != nil {
		t.Fatalf("VAInquiryPaymentIntrabank() error = %v", err)
	}

	want := VAInquiryPaymentIntrabankResponse{
		ResponseCode:    "2003200",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &VAInquiryPaymentIntrabankData{
			PartnerServiceID:   "12345",
			CustomerNo:         json.RawMessage(`12345678901234567890`),
			VirtualAccountNo:   "1234598765",
			PartnerReferenceNo: "ref-1",
			SourceAccountNo:    "1122334455",
			SourceAccountType:  "D",
			ProductName:        "Savings",
			BillAmountLabel:    "Total",
			BillAmountValue:    "100000.00",
			AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("VAInquiryPaymentIntrabank() = %+v, want %+v", resp, want)
	}
}

func TestVAInquiryPaymentIntrabank_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-payment-intrabank"
	tr := &snap.Transport{}
	req := VAInquiryPaymentIntrabankRequest{
		PartnerServiceID:   "12345",
		CustomerNo:         json.RawMessage(`12345678901234567890`),
		VirtualAccountNo:   "1234598765",
		PartnerReferenceNo: "ref-1",
		SourceAccountType:  "D",
	}
	if _, err := VAInquiryPaymentIntrabank(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("VAInquiryPaymentIntrabank() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if string(got["customerNo"]) != "12345678901234567890" {
		t.Errorf(`wire body["customerNo"] = %s, want bare number 12345678901234567890`, got["customerNo"])
	}
	var sourceAccountType string
	if err := json.Unmarshal(got["sourceAccountType"], &sourceAccountType); err != nil || sourceAccountType != "D" {
		t.Errorf(`wire body["sourceAccountType"] = %s, want "D"`, got["sourceAccountType"])
	}
}

// TestVAInquiryPaymentIntrabank_MandatoryFieldsAlwaysSerialized pins
// that PartnerServiceID, CustomerNo, and VirtualAccountNo — the
// identity triple, the only request fields without omitempty — are
// always present on the wire.
func TestVAInquiryPaymentIntrabank_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-payment-intrabank"
	tr := &snap.Transport{}
	if _, err := VAInquiryPaymentIntrabank(context.Background(), tr, hb, VAInquiryPaymentIntrabankRequest{}); err != nil {
		t.Fatalf("VAInquiryPaymentIntrabank() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo"} {
		if _, ok := got[key]; !ok {
			t.Errorf(`wire body missing %q key; want it always present`, key)
		}
	}
	if got["customerNo"] != nil {
		t.Errorf(`wire body["customerNo"] = %v, want JSON null for an unset json.RawMessage`, got["customerNo"])
	}
	if _, ok := got["partnerReferenceNo"]; ok {
		t.Error(`wire body has "partnerReferenceNo" key, want it omitted (Optional here)`)
	}
}

// TestVAInquiryPaymentIntrabank_MalformedCustomerNoIsMarshalError pins
// that a json.RawMessage field holding invalid JSON fails at
// json.Marshal, and that VAInquiryPaymentIntrabank surfaces that as an
// error without sending any HTTP request.
func TestVAInquiryPaymentIntrabank_MalformedCustomerNoIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2003200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-payment-intrabank"
	tr := &snap.Transport{}
	_, err := VAInquiryPaymentIntrabank(context.Background(), tr, hb, VAInquiryPaymentIntrabankRequest{
		CustomerNo: json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("VAInquiryPaymentIntrabank() error = nil, want non-nil for malformed CustomerNo JSON")
	}
	if requested.Load() {
		t.Error("VAInquiryPaymentIntrabank() sent an HTTP request despite a request-encoding failure")
	}
}

func TestVAInquiryPaymentIntrabank_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003200","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-payment-intrabank"
	tr := &snap.Transport{}
	_, err := VAInquiryPaymentIntrabank(context.Background(), tr, hb, VAInquiryPaymentIntrabankRequest{})
	if err == nil {
		t.Fatal("VAInquiryPaymentIntrabank() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("VAInquiryPaymentIntrabank() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestVAInquiryPaymentIntrabank_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003200","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-payment-intrabank"
	tr := &snap.Transport{}
	resp, err := VAInquiryPaymentIntrabank(context.Background(), tr, hb, VAInquiryPaymentIntrabankRequest{})
	if err == nil {
		t.Fatalf("VAInquiryPaymentIntrabank() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("VAInquiryPaymentIntrabank() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestVAInquiryPaymentIntrabank_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountdata":{"virtualAccountNo":"1234598765"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-payment-intrabank"
	tr := &snap.Transport{}
	resp, err := VAInquiryPaymentIntrabank(context.Background(), tr, hb, VAInquiryPaymentIntrabankRequest{})
	if err == nil {
		t.Fatalf("VAInquiryPaymentIntrabank() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
