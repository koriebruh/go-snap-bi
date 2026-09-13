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

func TestVAPaymentIntrabank_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003300",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountdata":{
      "partnerServiceId":"12345",
      "customerNo":12345678901234567890,
      "virtualAccountNo":"1234598765",
      "sourceAccountNo":"1122334455",
      "sourceAccountType":"D",
      "inquiryRequestId":"inq-1",
      "partnerReferenceNo":"ref-1",
      "paidAmount":{"value":"100000.00","currency":"IDR"},
      "cumulativePaymentAmount":{"value":"200000.00","currency":"IDR"},
      "paidBills":"3F",
      "paymentStatus":"Settled",
      "referenceNo":"123456789012345",
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment-intrabank"
	tr := &snap.Transport{}
	resp, err := VAPaymentIntrabank(context.Background(), tr, hb, VAPaymentIntrabankRequest{
		PartnerServiceID:   "12345",
		CustomerNo:         json.RawMessage(`12345678901234567890`),
		VirtualAccountNo:   "1234598765",
		PartnerReferenceNo: "ref-1",
		PaidAmount:         snap.Money{Value: "100000.00", Currency: "IDR"},
	})
	if err != nil {
		t.Fatalf("VAPaymentIntrabank() error = %v", err)
	}

	want := VAPaymentIntrabankResponse{
		ResponseCode:    "2003300",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &VAPaymentIntrabankData{
			PartnerServiceID:        "12345",
			CustomerNo:              json.RawMessage(`12345678901234567890`),
			VirtualAccountNo:        "1234598765",
			SourceAccountNo:         "1122334455",
			SourceAccountType:       "D",
			InquiryRequestID:        "inq-1",
			PartnerReferenceNo:      "ref-1",
			PaidAmount:              &snap.Money{Value: "100000.00", Currency: "IDR"},
			CumulativePaymentAmount: &snap.Money{Value: "200000.00", Currency: "IDR"},
			PaidBills:               "3F",
			PaymentStatus:           "Settled",
			ReferenceNo:             json.RawMessage(`"123456789012345"`),
			AdditionalInfo:          json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("VAPaymentIntrabank() = %+v, want %+v", resp, want)
	}
}

// TestVAPaymentIntrabank_ReferenceNoAcceptsEitherWireShape pins that
// ReferenceNo (json.RawMessage) round-trips both the bare-number shape
// (this endpoint's own worked request, research §5.3 line 137) and the
// quoted-string shape (this endpoint's own worked response).
func TestVAPaymentIntrabank_ReferenceNoAcceptsEitherWireShape(t *testing.T) {
	tests := []struct {
		name        string
		referenceNo string
	}{
		{name: "bare number", referenceNo: `123456789012345`},
		{name: "quoted string", referenceNo: `"123456789012345"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := `{"responseCode":"2003300","responseMessage":"ok","virtualAccountdata":{"referenceNo":` + tt.referenceNo + `}}`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(fixture))
			}))
			defer server.Close()

			hb := snaptest.TestHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment-intrabank"
			tr := &snap.Transport{}
			resp, err := VAPaymentIntrabank(context.Background(), tr, hb, VAPaymentIntrabankRequest{
				ReferenceNo: json.RawMessage(tt.referenceNo),
				PaidAmount:  snap.Money{Value: "1.00", Currency: "IDR"},
			})
			if err != nil {
				t.Fatalf("VAPaymentIntrabank() error = %v", err)
			}
			if resp.VirtualAccountData == nil {
				t.Fatal("VAPaymentIntrabank() VirtualAccountData = nil")
			}
			if string(resp.VirtualAccountData.ReferenceNo) != tt.referenceNo {
				t.Errorf("ReferenceNo = %s, want %s", resp.VirtualAccountData.ReferenceNo, tt.referenceNo)
			}
		})
	}
}

func TestVAPaymentIntrabank_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment-intrabank"
	tr := &snap.Transport{}
	req := VAPaymentIntrabankRequest{
		PartnerServiceID:   "12345",
		CustomerNo:         json.RawMessage(`12345678901234567890`),
		VirtualAccountNo:   "1234598765",
		PartnerReferenceNo: "ref-1",
		PaidAmount:         snap.Money{Value: "100000.00", Currency: "IDR"},
		ReferenceNo:        json.RawMessage(`123456789012345`),
	}
	if _, err := VAPaymentIntrabank(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("VAPaymentIntrabank() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if string(got["referenceNo"]) != "123456789012345" {
		t.Errorf(`wire body["referenceNo"] = %s, want bare number 123456789012345`, got["referenceNo"])
	}
	var paidAmount map[string]any
	if err := json.Unmarshal(got["paidAmount"], &paidAmount); err != nil {
		t.Fatalf("decode paidAmount: %v", err)
	}
	if paidAmount["value"] != "100000.00" {
		t.Errorf(`wire body["paidAmount"]["value"] = %v, want "100000.00"`, paidAmount["value"])
	}
}

// TestVAPaymentIntrabank_MandatoryFieldsAlwaysSerialized pins that
// PartnerServiceID, CustomerNo, VirtualAccountNo, PartnerReferenceNo,
// and PaidAmount — the request fields without omitempty — are always
// present on the wire. PaidAmount is a mandatory nested object (plain
// snap.Money, not *snap.Money), so it always serializes as an object.
func TestVAPaymentIntrabank_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment-intrabank"
	tr := &snap.Transport{}
	if _, err := VAPaymentIntrabank(context.Background(), tr, hb, VAPaymentIntrabankRequest{}); err != nil {
		t.Fatalf("VAPaymentIntrabank() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo", "partnerReferenceNo", "paidAmount"} {
		if _, ok := got[key]; !ok {
			t.Errorf(`wire body missing %q key; want it always present`, key)
		}
	}
	if _, ok := got["paidAmount"].(map[string]any); !ok {
		t.Errorf(`wire body["paidAmount"] = %v, want an object (mandatory nested snap.Money is a plain struct)`, got["paidAmount"])
	}
	if _, ok := got["referenceNo"]; ok {
		t.Error(`wire body has "referenceNo" key, want it omitted (Optional here)`)
	}
}

func TestVAPaymentIntrabank_MalformedReferenceNoIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2003300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment-intrabank"
	tr := &snap.Transport{}
	_, err := VAPaymentIntrabank(context.Background(), tr, hb, VAPaymentIntrabankRequest{
		ReferenceNo: json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("VAPaymentIntrabank() error = nil, want non-nil for malformed ReferenceNo JSON")
	}
	if requested.Load() {
		t.Error("VAPaymentIntrabank() sent an HTTP request despite a request-encoding failure")
	}
}

func TestVAPaymentIntrabank_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003300","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment-intrabank"
	tr := &snap.Transport{}
	_, err := VAPaymentIntrabank(context.Background(), tr, hb, VAPaymentIntrabankRequest{})
	if err == nil {
		t.Fatal("VAPaymentIntrabank() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("VAPaymentIntrabank() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestVAPaymentIntrabank_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment-intrabank"
	tr := &snap.Transport{}
	resp, err := VAPaymentIntrabank(context.Background(), tr, hb, VAPaymentIntrabankRequest{})
	if err == nil {
		t.Fatalf("VAPaymentIntrabank() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("VAPaymentIntrabank() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestVAPaymentIntrabank_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountdata":{"partnerReferenceNo":"ref-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/payment-intrabank"
	tr := &snap.Transport{}
	resp, err := VAPaymentIntrabank(context.Background(), tr, hb, VAPaymentIntrabankRequest{})
	if err == nil {
		t.Fatalf("VAPaymentIntrabank() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
