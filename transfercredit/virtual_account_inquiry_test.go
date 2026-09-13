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

func TestVAInquiry_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2002400",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountData":{
      "partnerServiceId":"12345",
      "customerNo":"98765",
      "virtualAccountNo":"1234598765",
      "inquiryStatus":"00",
      "inquiryReason":{"english":"Success","indonesia":"Sukses"},
      "virtualAccountName":"Jane Doe",
      "virtualAccountEmail":"jane@example.com",
      "virtualAccountPhone":"081234567890",
      "inquiryRequestId":"inq-1",
      "totalAmount":{"value":"100000.00","currency":"IDR"},
      "subCompany":"00001",
      "billDetails":[{"billCode":"01","billNo":"bill-1","billReferenceNo":"BILLREF1"}],
      "freeTexts":[{"english":"note","indonesia":"catatan"}],
      "virtualAccountTrxType":"C",
      "feeAmount":{"value":"1000.00","currency":"IDR"},
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry"
	tr := &snap.Transport{}
	resp, err := VAInquiry(context.Background(), tr, hb, VAInquiryRequest{
		PartnerServiceID: "12345",
		CustomerNo:       json.RawMessage(`"98765"`),
		VirtualAccountNo: "1234598765",
		InquiryRequestID: "inq-1",
	})
	if err != nil {
		t.Fatalf("VAInquiry() error = %v", err)
	}

	want := VAInquiryResponse{
		ResponseCode:    "2002400",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &VAInquiryData{
			PartnerServiceID:      "12345",
			CustomerNo:            json.RawMessage(`"98765"`),
			VirtualAccountNo:      "1234598765",
			InquiryStatus:         "00",
			InquiryReason:         &LocalizedText{English: "Success", Indonesia: "Sukses"},
			VirtualAccountName:    "Jane Doe",
			VirtualAccountEmail:   "jane@example.com",
			VirtualAccountPhone:   "081234567890",
			InquiryRequestID:      "inq-1",
			TotalAmount:           &snap.Money{Value: "100000.00", Currency: "IDR"},
			SubCompany:            "00001",
			BillDetails:           []BillDetail{{BillCode: "01", BillNo: "bill-1", BillReferenceNo: json.RawMessage(`"BILLREF1"`)}},
			FreeTexts:             []LocalizedText{{English: "note", Indonesia: "catatan"}},
			VirtualAccountTrxType: "C",
			FeeAmount:             &snap.Money{Value: "1000.00", Currency: "IDR"},
			AdditionalInfo:        json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("VAInquiry() = %+v, want %+v", resp, want)
	}
}

func TestVAInquiry_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry"
	tr := &snap.Transport{}
	req := VAInquiryRequest{
		PartnerServiceID: "12345",
		CustomerNo:       json.RawMessage(`"98765"`),
		VirtualAccountNo: "1234598765",
		ChannelCode:      json.RawMessage(`6011`),
		InquiryRequestID: "inq-1",
	}
	if _, err := VAInquiry(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("VAInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["customerNo"] != "98765" {
		t.Errorf(`wire body["customerNo"] = %v, want "98765"`, got["customerNo"])
	}
	if got["channelCode"] != float64(6011) {
		t.Errorf(`wire body["channelCode"] = %v, want bare number 6011`, got["channelCode"])
	}
	if got["inquiryRequestId"] != "inq-1" {
		t.Errorf(`wire body["inquiryRequestId"] = %v, want "inq-1"`, got["inquiryRequestId"])
	}
}

// TestVAInquiry_CustomerNoAcceptsEitherWireShape pins that CustomerNo
// (json.RawMessage) round-trips both the quoted-string shape (24/25's
// own worked examples) and the bare-number shape (26's worked example),
// per the Phase 14 design doc's ambiguous-type decision.
func TestVAInquiry_CustomerNoAcceptsEitherWireShape(t *testing.T) {
	tests := []struct {
		name       string
		customerNo string
	}{
		{name: "quoted string", customerNo: `"98765"`},
		{name: "bare number", customerNo: `12345678901234567890`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := `{"responseCode":"2002400","responseMessage":"ok","virtualAccountData":{"customerNo":` + tt.customerNo + `}}`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(fixture))
			}))
			defer server.Close()

			hb := snaptest.TestHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry"
			tr := &snap.Transport{}
			resp, err := VAInquiry(context.Background(), tr, hb, VAInquiryRequest{
				CustomerNo: json.RawMessage(tt.customerNo),
			})
			if err != nil {
				t.Fatalf("VAInquiry() error = %v", err)
			}
			if resp.VirtualAccountData == nil {
				t.Fatal("VAInquiry() VirtualAccountData = nil")
			}
			if string(resp.VirtualAccountData.CustomerNo) != tt.customerNo {
				t.Errorf("CustomerNo = %s, want %s", resp.VirtualAccountData.CustomerNo, tt.customerNo)
			}
		})
	}
}

// TestVAInquiry_MandatoryFieldsAlwaysSerialized pins that
// PartnerServiceID, CustomerNo, VirtualAccountNo, and InquiryRequestID
// — the four request fields without omitempty — are always present on
// the wire. CustomerNo (json.RawMessage) serializes as JSON null when
// unset, not "", unlike the string-typed mandatory fields.
func TestVAInquiry_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry"
	tr := &snap.Transport{}
	if _, err := VAInquiry(context.Background(), tr, hb, VAInquiryRequest{}); err != nil {
		t.Fatalf("VAInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo", "inquiryRequestId"} {
		if _, ok := got[key]; !ok {
			t.Errorf(`wire body missing %q key; want it always present`, key)
		}
	}
	if got["customerNo"] != nil {
		t.Errorf(`wire body["customerNo"] = %v, want JSON null for an unset json.RawMessage`, got["customerNo"])
	}
	if got["partnerServiceId"] != "" {
		t.Errorf(`wire body["partnerServiceId"] = %v, want ""`, got["partnerServiceId"])
	}
	if got["virtualAccountNo"] != "" {
		t.Errorf(`wire body["virtualAccountNo"] = %v, want ""`, got["virtualAccountNo"])
	}
	if got["inquiryRequestId"] != "" {
		t.Errorf(`wire body["inquiryRequestId"] = %v, want ""`, got["inquiryRequestId"])
	}
}

// TestVAInquiry_MalformedCustomerNoIsMarshalError pins that a
// json.RawMessage field holding invalid JSON fails at json.Marshal
// (encoding/json validates RawMessage content when marshaling the
// enclosing struct), and that VAInquiry surfaces that as an error
// without sending any HTTP request.
func TestVAInquiry_MalformedCustomerNoIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2002400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry"
	tr := &snap.Transport{}
	_, err := VAInquiry(context.Background(), tr, hb, VAInquiryRequest{
		CustomerNo: json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("VAInquiry() error = nil, want non-nil for malformed CustomerNo JSON")
	}
	if requested.Load() {
		t.Error("VAInquiry() sent an HTTP request despite a request-encoding failure")
	}
}

func TestVAInquiry_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4002400","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry"
	tr := &snap.Transport{}
	_, err := VAInquiry(context.Background(), tr, hb, VAInquiryRequest{})
	if err == nil {
		t.Fatal("VAInquiry() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("VAInquiry() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestVAInquiry_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2002400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry"
	tr := &snap.Transport{}
	resp, err := VAInquiry(context.Background(), tr, hb, VAInquiryRequest{})
	if err == nil {
		t.Fatalf("VAInquiry() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("VAInquiry() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestVAInquiry_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountData":{"inquiryRequestId":"inq-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry"
	tr := &snap.Transport{}
	resp, err := VAInquiry(context.Background(), tr, hb, VAInquiryRequest{})
	if err == nil {
		t.Fatalf("VAInquiry() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
