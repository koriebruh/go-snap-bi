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

func TestVAInquiryStatus_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2002600",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountData":{
      "partnerServiceId":"12345",
      "customerNo":12345678901234567890,
      "virtualAccountNo":"1234598765",
      "trxId":"trx-1",
      "paymentRequestId":"pay-1",
      "channelCode":6011,
      "paidAmount":{"value":"100000.00","currency":"IDR"},
      "paidBills":"3F",
      "paymentType":1,
      "paymentFlagStatus":"00",
      "transactionDate":"2020-12-20T10:00:00+07:00"
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/status"
	tr := &snap.Transport{}
	resp, err := VAInquiryStatus(context.Background(), tr, hb, VAInquiryStatusRequest{
		PartnerServiceID: "12345",
		CustomerNo:       json.RawMessage(`12345678901234567890`),
		VirtualAccountNo: "1234598765",
	})
	if err != nil {
		t.Fatalf("VAInquiryStatus() error = %v", err)
	}

	want := VAInquiryStatusResponse{
		ResponseCode:    "2002600",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &VAInquiryStatusData{
			PartnerServiceID:  "12345",
			CustomerNo:        json.RawMessage(`12345678901234567890`),
			VirtualAccountNo:  "1234598765",
			TrxID:             "trx-1",
			PaymentRequestID:  "pay-1",
			ChannelCode:       json.RawMessage(`6011`),
			PaidAmount:        &snap.Money{Value: "100000.00", Currency: "IDR"},
			PaidBills:         "3F",
			PaymentType:       json.RawMessage(`1`),
			PaymentFlagStatus: "00",
			TransactionDate:   "2020-12-20T10:00:00+07:00",
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("VAInquiryStatus() = %+v, want %+v", resp, want)
	}
}

// TestVAInquiryStatus_CustomerNoAcceptsBareNumber pins the specific
// wire shape from this endpoint's own worked example (research §5.3
// line 135): a 20-digit bare JSON number, which exceeds int64 range —
// exactly why CustomerNo is json.RawMessage and not a numeric Go type.
func TestVAInquiryStatus_CustomerNoAcceptsBareNumber(t *testing.T) {
	const bareNumber = `12345678901234567890`
	fixture := `{"responseCode":"2002600","responseMessage":"ok","virtualAccountData":{"customerNo":` + bareNumber + `}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/status"
	tr := &snap.Transport{}
	resp, err := VAInquiryStatus(context.Background(), tr, hb, VAInquiryStatusRequest{
		CustomerNo: json.RawMessage(bareNumber),
	})
	if err != nil {
		t.Fatalf("VAInquiryStatus() error = %v", err)
	}
	if resp.VirtualAccountData == nil {
		t.Fatal("VAInquiryStatus() VirtualAccountData = nil")
	}
	if string(resp.VirtualAccountData.CustomerNo) != bareNumber {
		t.Errorf("CustomerNo = %s, want %s", resp.VirtualAccountData.CustomerNo, bareNumber)
	}
}

func TestVAInquiryStatus_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/status"
	tr := &snap.Transport{}
	req := VAInquiryStatusRequest{
		PartnerServiceID: "12345",
		CustomerNo:       json.RawMessage(`12345678901234567890`),
		VirtualAccountNo: "1234598765",
		InquiryRequestID: "inq-1",
		PaymentRequestID: "pay-1",
		AdditionalInfo:   json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := VAInquiryStatus(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("VAInquiryStatus() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["inquiryRequestId"] != "inq-1" {
		t.Errorf(`wire body["inquiryRequestId"] = %v, want "inq-1"`, got["inquiryRequestId"])
	}
	if got["paymentRequestId"] != "pay-1" {
		t.Errorf(`wire body["paymentRequestId"] = %v, want "pay-1"`, got["paymentRequestId"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestVAInquiryStatus_MandatoryFieldsAlwaysSerialized pins that
// PartnerServiceID, CustomerNo, and VirtualAccountNo — the identity
// triple, the only request fields without omitempty — are always
// present on the wire, and that InquiryRequestID/PaymentRequestID
// (both Optional/Conditional here) are omitted when unset.
func TestVAInquiryStatus_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/status"
	tr := &snap.Transport{}
	if _, err := VAInquiryStatus(context.Background(), tr, hb, VAInquiryStatusRequest{}); err != nil {
		t.Fatalf("VAInquiryStatus() error = %v", err)
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
	for _, key := range []string{"inquiryRequestId", "paymentRequestId"} {
		if _, ok := got[key]; ok {
			t.Errorf(`wire body has %q key, want it omitted (Optional/Conditional here)`, key)
		}
	}
}

// TestVAInquiryStatus_MalformedCustomerNoIsMarshalError pins that a
// json.RawMessage field holding invalid JSON fails at json.Marshal, and
// that VAInquiryStatus surfaces that as an error without sending any
// HTTP request.
func TestVAInquiryStatus_MalformedCustomerNoIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2002600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/status"
	tr := &snap.Transport{}
	_, err := VAInquiryStatus(context.Background(), tr, hb, VAInquiryStatusRequest{
		CustomerNo: json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("VAInquiryStatus() error = nil, want non-nil for malformed CustomerNo JSON")
	}
	if requested.Load() {
		t.Error("VAInquiryStatus() sent an HTTP request despite a request-encoding failure")
	}
}

func TestVAInquiryStatus_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4002600","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/status"
	tr := &snap.Transport{}
	_, err := VAInquiryStatus(context.Background(), tr, hb, VAInquiryStatusRequest{})
	if err == nil {
		t.Fatal("VAInquiryStatus() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("VAInquiryStatus() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestVAInquiryStatus_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2002600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/status"
	tr := &snap.Transport{}
	resp, err := VAInquiryStatus(context.Background(), tr, hb, VAInquiryStatusRequest{})
	if err == nil {
		t.Fatalf("VAInquiryStatus() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("VAInquiryStatus() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestVAInquiryStatus_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountData":{"trxId":"trx-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/status"
	tr := &snap.Transport{}
	resp, err := VAInquiryStatus(context.Background(), tr, hb, VAInquiryStatusRequest{})
	if err == nil {
		t.Fatalf("VAInquiryStatus() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
