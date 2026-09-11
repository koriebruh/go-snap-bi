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

func TestVAGetReport_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003500",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountdata":[
      {
         "partnerServiceId":"12345",
         "customerNo":"98765",
         "virtualAccountNo":"1234598765",
         "trxId":"trx-1",
         "paymentRequestId":"pay-1",
         "paidAmount":{"value":"100000.00","currency":"IDR"},
         "transactionDate":"2020-12-20T10:00:00+07:00"
      },
      {
         "partnerServiceId":"12345",
         "customerNo":"98766",
         "virtualAccountNo":"1234598766",
         "trxId":"trx-2"
      }
   ]
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/get-report"
	tr := &Transport{}
	resp, err := VAGetReport(context.Background(), tr, hb, GetReportRequest{
		PartnerServiceID: json.RawMessage(`12345`),
	})
	if err != nil {
		t.Fatalf("VAGetReport() error = %v", err)
	}

	want := GetReportResponse{
		ResponseCode:    "2003500",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: []GetReportData{
			{
				PartnerServiceID: "12345",
				CustomerNo:       json.RawMessage(`"98765"`),
				VirtualAccountNo: "1234598765",
				TrxID:            "trx-1",
				PaymentRequestID: "pay-1",
				PaidAmount:       &Money{Value: "100000.00", Currency: "IDR"},
				TransactionDate:  "2020-12-20T10:00:00+07:00",
			},
			{
				PartnerServiceID: "12345",
				CustomerNo:       json.RawMessage(`"98766"`),
				VirtualAccountNo: "1234598766",
				TrxID:            "trx-2",
			},
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("VAGetReport() = %+v, want %+v", resp, want)
	}
}

func TestVAGetReport_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/get-report"
	tr := &Transport{}
	req := GetReportRequest{
		PartnerServiceID: json.RawMessage(`12345`),
		StartDate:        "2020-12-01",
		EndDate:          "2020-12-31",
	}
	if _, err := VAGetReport(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("VAGetReport() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if string(got["partnerServiceId"]) != "12345" {
		t.Errorf(`wire body["partnerServiceId"] = %s, want bare number 12345`, got["partnerServiceId"])
	}
	var startDate string
	if err := json.Unmarshal(got["startDate"], &startDate); err != nil || startDate != "2020-12-01" {
		t.Errorf(`wire body["startDate"] = %s, want "2020-12-01"`, got["startDate"])
	}
}

// TestVAGetReport_MandatoryFieldAlwaysSerialized pins that
// PartnerServiceID — the only request field without omitempty — is
// always present on the wire, and marshals as JSON null when unset
// (json.RawMessage's zero value), not "".
func TestVAGetReport_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/get-report"
	tr := &Transport{}
	if _, err := VAGetReport(context.Background(), tr, hb, GetReportRequest{}); err != nil {
		t.Fatalf("VAGetReport() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if v, ok := got["partnerServiceId"]; !ok {
		t.Error(`wire body missing "partnerServiceId" key; want it always present`)
	} else if v != nil {
		t.Errorf(`wire body["partnerServiceId"] = %v, want JSON null for an unset json.RawMessage`, v)
	}
	if _, ok := got["startDate"]; ok {
		t.Error(`wire body has "startDate" key, want it omitted (Optional here)`)
	}
}

func TestVAGetReport_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003500","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/get-report"
	tr := &Transport{}
	_, err := VAGetReport(context.Background(), tr, hb, GetReportRequest{})
	if err == nil {
		t.Fatal("VAGetReport() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("VAGetReport() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestVAGetReport_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/get-report"
	tr := &Transport{}
	resp, err := VAGetReport(context.Background(), tr, hb, GetReportRequest{})
	if err == nil {
		t.Fatalf("VAGetReport() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("VAGetReport() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestVAGetReport_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountdata":[]}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/get-report"
	tr := &Transport{}
	resp, err := VAGetReport(context.Background(), tr, hb, GetReportRequest{})
	if err == nil {
		t.Fatalf("VAGetReport() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}

// TestVAGetReport_BareNumberResponseCodeIsAKnownLimitation pins a
// documented, package-wide known limitation: research §5.3 line 141
// shows this endpoint's own worked response with a bare-number
// responseCode ("responseCode":2003500). responseCode is decoded
// twice for every call in this package — once by the shared transport
// layer (transport.go) into an internal string-typed field before any
// per-service Response type sees the body, and again by this
// endpoint's own json.Unmarshal(env.Raw, &resp) into
// GetReportResponse.ResponseCode string. A bare-number responseCode
// fails at the transport layer's decode first (this test pins that
// current behavior); it would also fail at GetReportResponse's own
// decode if the transport layer alone were fixed, per the same
// reasoning verified empirically for InquiryVAResponse in Phase 13
// (a string field cannot receive a bare JSON number). Fixing this
// package-wide is out of scope for this phase — see the Phase 15
// design doc's "Known limitation" section.
func TestVAGetReport_BareNumberResponseCodeIsAKnownLimitation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":2003500,"responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/get-report"
	tr := &Transport{}
	_, err := VAGetReport(context.Background(), tr, hb, GetReportRequest{})
	if err == nil {
		t.Fatal("VAGetReport() error = nil, want non-nil: a bare-number responseCode is a known limitation, not silently accepted")
	}
}
