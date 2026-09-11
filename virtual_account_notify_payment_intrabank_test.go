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
	"sync/atomic"
	"testing"
)

func TestVANotifyPaymentIntrabank_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003400",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountdata":{
      "partnerServiceId":"12345",
      "customerNo":12345678901234567890,
      "virtualAccountNo":"1234598765",
      "inquiryRequestId":"inq-1",
      "paymentRequestId":"pay-1",
      "partnerReferenceNo":"ref-1",
      "trxDateTime":"2020-12-20T10:00:00+07:00",
      "paymentStatus":"Settled",
      "paymentFlagReason":{"english":"Success","indonesia":"Sukses"},
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/notify-payment-intrabank"
	tr := &Transport{}
	resp, err := VANotifyPaymentIntrabank(context.Background(), tr, hb, VANotifyPaymentIntrabankRequest{
		PartnerServiceID: "12345",
		CustomerNo:       json.RawMessage(`12345678901234567890`),
		VirtualAccountNo: "1234598765",
	})
	if err != nil {
		t.Fatalf("VANotifyPaymentIntrabank() error = %v", err)
	}

	want := VANotifyPaymentIntrabankResponse{
		ResponseCode:    "2003400",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &VANotifyPaymentIntrabankData{
			PartnerServiceID:   "12345",
			CustomerNo:         json.RawMessage(`12345678901234567890`),
			VirtualAccountNo:   "1234598765",
			InquiryRequestID:   "inq-1",
			PaymentRequestID:   "pay-1",
			PartnerReferenceNo: "ref-1",
			TrxDateTime:        "2020-12-20T10:00:00+07:00",
			PaymentStatus:      "Settled",
			PaymentFlagReason:  &LocalizedText{English: "Success", Indonesia: "Sukses"},
			AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("VANotifyPaymentIntrabank() = %+v, want %+v", resp, want)
	}
}

func TestVANotifyPaymentIntrabank_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/notify-payment-intrabank"
	tr := &Transport{}
	req := VANotifyPaymentIntrabankRequest{
		PartnerServiceID:  "12345",
		CustomerNo:        json.RawMessage(`12345678901234567890`),
		VirtualAccountNo:  "1234598765",
		PaymentStatus:     "Settled",
		PaymentFlagReason: &LocalizedText{English: "Success", Indonesia: "Sukses"},
	}
	if _, err := VANotifyPaymentIntrabank(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("VANotifyPaymentIntrabank() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	var paymentStatus string
	if err := json.Unmarshal(got["paymentStatus"], &paymentStatus); err != nil || paymentStatus != "Settled" {
		t.Errorf(`wire body["paymentStatus"] = %s, want "Settled"`, got["paymentStatus"])
	}
	var reason map[string]any
	if err := json.Unmarshal(got["paymentFlagReason"], &reason); err != nil {
		t.Fatalf("decode paymentFlagReason: %v", err)
	}
	if reason["english"] != "Success" {
		t.Errorf(`wire body["paymentFlagReason"]["english"] = %v, want "Success"`, reason["english"])
	}
}

// TestVANotifyPaymentIntrabank_MandatoryFieldsAlwaysSerialized pins
// that PartnerServiceID, CustomerNo, and VirtualAccountNo — the
// identity triple, the only request fields without omitempty — are
// always present on the wire.
func TestVANotifyPaymentIntrabank_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/notify-payment-intrabank"
	tr := &Transport{}
	if _, err := VANotifyPaymentIntrabank(context.Background(), tr, hb, VANotifyPaymentIntrabankRequest{}); err != nil {
		t.Fatalf("VANotifyPaymentIntrabank() error = %v", err)
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
	if _, ok := got["paymentStatus"]; ok {
		t.Error(`wire body has "paymentStatus" key, want it omitted (Optional here)`)
	}
}

func TestVANotifyPaymentIntrabank_MalformedCustomerNoIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2003400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/notify-payment-intrabank"
	tr := &Transport{}
	_, err := VANotifyPaymentIntrabank(context.Background(), tr, hb, VANotifyPaymentIntrabankRequest{
		CustomerNo: json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("VANotifyPaymentIntrabank() error = nil, want non-nil for malformed CustomerNo JSON")
	}
	if requested.Load() {
		t.Error("VANotifyPaymentIntrabank() sent an HTTP request despite a request-encoding failure")
	}
}

func TestVANotifyPaymentIntrabank_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003400","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/notify-payment-intrabank"
	tr := &Transport{}
	_, err := VANotifyPaymentIntrabank(context.Background(), tr, hb, VANotifyPaymentIntrabankRequest{})
	if err == nil {
		t.Fatal("VANotifyPaymentIntrabank() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("VANotifyPaymentIntrabank() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestVANotifyPaymentIntrabank_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/notify-payment-intrabank"
	tr := &Transport{}
	resp, err := VANotifyPaymentIntrabank(context.Background(), tr, hb, VANotifyPaymentIntrabankRequest{})
	if err == nil {
		t.Fatalf("VANotifyPaymentIntrabank() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("VANotifyPaymentIntrabank() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestVANotifyPaymentIntrabank_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountdata":{"partnerReferenceNo":"ref-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/notify-payment-intrabank"
	tr := &Transport{}
	resp, err := VANotifyPaymentIntrabank(context.Background(), tr, hb, VANotifyPaymentIntrabankRequest{})
	if err == nil {
		t.Fatalf("VANotifyPaymentIntrabank() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
