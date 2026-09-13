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
	"sync/atomic"
	"testing"

	snap "github.com/koriebruh/go-snap-bi"
	"github.com/koriebruh/go-snap-bi/internal/snaptest"
)

func TestVAPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2002500",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountData":{
      "partnerServiceId":"12345",
      "customerNo":"98765",
      "virtualAccountNo":"1234598765",
      "trxId":"trx-1",
      "paymentRequestId":"pay-1",
      "channelCode":6011,
      "paidAmount":{"value":"100000.00","currency":"IDR"},
      "cumulativePaymentAmount":{"value":"200000.00","currency":"IDR"},
      "paidBills":"3F",
      "totalAmount":{"value":"100000.00","currency":"IDR"},
      "trxDateTime":"2020-12-20T10:00:00+07:00",
      "referenceNo":"ref-1",
      "journalNum":"J001",
      "paymentType":1,
      "flagAdvise":"N",
      "subCompany":"00001",
      "billDetails":[{"billCode":"01","billNo":"bill-1","status":"00"}],
      "freeTexts":[{"english":"note","indonesia":"catatan"}],
      "paymentFlagReason":{"english":"Success","indonesia":"Sukses"},
      "paymentFlagStatus":"00"
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment"
	tr := &snap.Transport{}
	resp, err := VAPayment(context.Background(), tr, hb, VAPaymentRequest{
		PartnerServiceID: "12345",
		CustomerNo:       json.RawMessage(`"98765"`),
		VirtualAccountNo: "1234598765",
		PaymentRequestID: "pay-1",
		PaidAmount:       snap.Money{Value: "100000.00", Currency: "IDR"},
	})
	if err != nil {
		t.Fatalf("VAPayment() error = %v", err)
	}

	want := VAPaymentResponse{
		ResponseCode:    "2002500",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &VAPaymentData{
			PartnerServiceID:        "12345",
			CustomerNo:              json.RawMessage(`"98765"`),
			VirtualAccountNo:        "1234598765",
			TrxID:                   "trx-1",
			PaymentRequestID:        "pay-1",
			ChannelCode:             json.RawMessage(`6011`),
			PaidAmount:              &snap.Money{Value: "100000.00", Currency: "IDR"},
			CumulativePaymentAmount: &snap.Money{Value: "200000.00", Currency: "IDR"},
			PaidBills:               "3F",
			TotalAmount:             &snap.Money{Value: "100000.00", Currency: "IDR"},
			TrxDateTime:             "2020-12-20T10:00:00+07:00",
			ReferenceNo:             "ref-1",
			JournalNum:              "J001",
			PaymentType:             json.RawMessage(`1`),
			FlagAdvise:              "N",
			SubCompany:              "00001",
			BillDetails:             []BillDetail{{BillCode: "01", BillNo: "bill-1", Status: "00"}},
			FreeTexts:               []LocalizedText{{English: "note", Indonesia: "catatan"}},
			PaymentFlagReason:       &LocalizedText{English: "Success", Indonesia: "Sukses"},
			PaymentFlagStatus:       "00",
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("VAPayment() = %+v, want %+v", resp, want)
	}
}

func TestVAPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment"
	tr := &snap.Transport{}
	req := VAPaymentRequest{
		PartnerServiceID: "12345",
		CustomerNo:       json.RawMessage(`"98765"`),
		VirtualAccountNo: "1234598765",
		PaymentRequestID: "pay-1",
		PaymentType:      json.RawMessage(`1`),
		PaidAmount:       snap.Money{Value: "100000.00", Currency: "IDR"},
	}
	if _, err := VAPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("VAPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["paymentType"] != float64(1) {
		t.Errorf(`wire body["paymentType"] = %v, want bare number 1`, got["paymentType"])
	}
	paidAmount, ok := got["paidAmount"].(map[string]any)
	if !ok {
		t.Fatalf(`wire body["paidAmount"] = %v, want an object`, got["paidAmount"])
	}
	if paidAmount["value"] != "100000.00" {
		t.Errorf(`wire body["paidAmount"]["value"] = %v, want "100000.00"`, paidAmount["value"])
	}
}

// TestVAPayment_MandatoryFieldsAlwaysSerialized pins that
// PartnerServiceID, CustomerNo, VirtualAccountNo, PaymentRequestID, and
// PaidAmount — the request fields without omitempty — are always
// present on the wire. PaidAmount is a mandatory nested object (plain
// snap.Money, not *snap.Money), so it always serializes as an object per the
// package's omitempty-is-a-no-op-on-structs rule; CustomerNo
// (json.RawMessage) serializes as JSON null when unset.
func TestVAPayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment"
	tr := &snap.Transport{}
	if _, err := VAPayment(context.Background(), tr, hb, VAPaymentRequest{}); err != nil {
		t.Fatalf("VAPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo", "paymentRequestId", "paidAmount"} {
		if _, ok := got[key]; !ok {
			t.Errorf(`wire body missing %q key; want it always present`, key)
		}
	}
	if got["customerNo"] != nil {
		t.Errorf(`wire body["customerNo"] = %v, want JSON null for an unset json.RawMessage`, got["customerNo"])
	}
	if _, ok := got["paidAmount"].(map[string]any); !ok {
		t.Errorf(`wire body["paidAmount"] = %v, want an object (mandatory nested snap.Money is a plain struct)`, got["paidAmount"])
	}
	if _, ok := got["trxId"]; ok {
		t.Error(`wire body has "trxId" key, want it omitted (Conditional/Optional here)`)
	}
}

// TestVAPayment_MalformedPaymentTypeIsMarshalError pins that a
// json.RawMessage field holding invalid JSON fails at json.Marshal, and
// that VAPayment surfaces that as an error without sending any HTTP
// request.
func TestVAPayment_MalformedPaymentTypeIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2002500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment"
	tr := &snap.Transport{}
	_, err := VAPayment(context.Background(), tr, hb, VAPaymentRequest{
		PaymentType: json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("VAPayment() error = nil, want non-nil for malformed PaymentType JSON")
	}
	if requested.Load() {
		t.Error("VAPayment() sent an HTTP request despite a request-encoding failure")
	}
}

func TestVAPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4002500","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment"
	tr := &snap.Transport{}
	_, err := VAPayment(context.Background(), tr, hb, VAPaymentRequest{})
	if err == nil {
		t.Fatal("VAPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("VAPayment() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestVAPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2002500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment"
	tr := &snap.Transport{}
	resp, err := VAPayment(context.Background(), tr, hb, VAPaymentRequest{})
	if err == nil {
		t.Fatalf("VAPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("VAPayment() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestVAPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountData":{"trxId":"trx-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment"
	tr := &snap.Transport{}
	resp, err := VAPayment(context.Background(), tr, hb, VAPaymentRequest{})
	if err == nil {
		t.Fatalf("VAPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
