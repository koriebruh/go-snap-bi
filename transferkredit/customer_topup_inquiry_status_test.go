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

// TestCustomerTopUpInquiryStatus_IdenticalToTransactionStatusInquiryBank
// pins that CustomerTopUpInquiryStatusRequest/Response (Phase 17) are
// field-identical to TransactionStatusInquiryBankRequest/Response
// (Phase 16), per research §5.5's "same originalX/serviceCode/status
// shape as Transaction Status Inquiry Bank." These types have no
// compile-time link to each other, so nothing else in the package
// would catch one of them drifting from the other.
func TestCustomerTopUpInquiryStatus_IdenticalToTransactionStatusInquiryBank(t *testing.T) {
	shape := func(v any) []string {
		typ := reflect.TypeOf(v)
		fields := make([]string, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			fields[i] = f.Name + " " + f.Type.String() + " `" + string(f.Tag) + "`"
		}
		return fields
	}

	if got, want := shape(CustomerTopUpInquiryStatusRequest{}), shape(TransactionStatusInquiryBankRequest{}); !reflect.DeepEqual(got, want) {
		t.Errorf("CustomerTopUpInquiryStatusRequest field shape = %v, want (matching TransactionStatusInquiryBankRequest) %v", got, want)
	}
	if got, want := shape(CustomerTopUpInquiryStatusResponse{}), shape(TransactionStatusInquiryBankResponse{}); !reflect.DeepEqual(got, want) {
		t.Errorf("CustomerTopUpInquiryStatusResponse field shape = %v, want (matching TransactionStatusInquiryBankResponse) %v", got, want)
	}
}

func TestCustomerTopUpInquiryStatus_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003900",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "originalExternalId":"ext-1",
   "serviceCode":"38",
   "transactionDate":"2020-12-20T10:00:00+07:00",
   "amount":{"value":"100000.00","currency":"IDR"},
   "beneficiaryAccountNo":"1122334455",
   "beneficiaryBankCode":"014",
   "previousResponseCode":"2003800",
   "referenceNumber":"refnum-1",
   "sourceAccountNo":"9988776655",
   "transactionId":"TX000001",
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up-inquiry-status"
	tr := &snap.Transport{}
	resp, err := CustomerTopUpInquiryStatus(context.Background(), tr, hb, CustomerTopUpInquiryStatusRequest{
		ServiceCode: "38",
	})
	if err != nil {
		t.Fatalf("CustomerTopUpInquiryStatus() error = %v", err)
	}

	want := CustomerTopUpInquiryStatusResponse{
		ResponseCode:               "2003900",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		ServiceCode:                "38",
		TransactionDate:            "2020-12-20T10:00:00+07:00",
		Amount:                     &snap.Money{Value: "100000.00", Currency: "IDR"},
		BeneficiaryAccountNo:       "1122334455",
		BeneficiaryBankCode:        "014",
		PreviousResponseCode:       "2003800",
		ReferenceNumber:            "refnum-1",
		SourceAccountNo:            "9988776655",
		TransactionID:              "TX000001",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CustomerTopUpInquiryStatus() = %+v, want %+v", resp, want)
	}
}

func TestCustomerTopUpInquiryStatus_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up-inquiry-status"
	tr := &snap.Transport{}
	req := CustomerTopUpInquiryStatusRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		ServiceCode:                "38",
		Amount:                     &snap.Money{Value: "100000.00", Currency: "IDR"},
	}
	if _, err := CustomerTopUpInquiryStatus(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CustomerTopUpInquiryStatus() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["serviceCode"] != "38" {
		t.Errorf(`wire body["serviceCode"] = %v, want "38"`, got["serviceCode"])
	}
}

// TestCustomerTopUpInquiryStatus_MandatoryFieldAlwaysSerialized pins
// that ServiceCode — the only request field without omitempty — is
// always present on the wire, even as "".
func TestCustomerTopUpInquiryStatus_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up-inquiry-status"
	tr := &snap.Transport{}
	if _, err := CustomerTopUpInquiryStatus(context.Background(), tr, hb, CustomerTopUpInquiryStatusRequest{}); err != nil {
		t.Fatalf("CustomerTopUpInquiryStatus() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	v, ok := got["serviceCode"]
	if !ok {
		t.Fatal(`wire body missing "serviceCode" key; want it always present, even as ""`)
	}
	if v != "" {
		t.Errorf(`wire body["serviceCode"] = %v, want ""`, v)
	}
}

// TestCustomerTopUpInquiryStatus_MalformedAdditionalInfoIsMarshalError
// pins that a json.RawMessage field holding invalid JSON fails at
// json.Marshal, and that CustomerTopUpInquiryStatus surfaces that as
// an error without sending any HTTP request.
func TestCustomerTopUpInquiryStatus_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2003900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up-inquiry-status"
	tr := &snap.Transport{}
	_, err := CustomerTopUpInquiryStatus(context.Background(), tr, hb, CustomerTopUpInquiryStatusRequest{
		AdditionalInfo: json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("CustomerTopUpInquiryStatus() error = nil, want non-nil for malformed AdditionalInfo JSON")
	}
	if requested.Load() {
		t.Error("CustomerTopUpInquiryStatus() sent an HTTP request despite a request-encoding failure")
	}
}

func TestCustomerTopUpInquiryStatus_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003900","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up-inquiry-status"
	tr := &snap.Transport{}
	_, err := CustomerTopUpInquiryStatus(context.Background(), tr, hb, CustomerTopUpInquiryStatusRequest{})
	if err == nil {
		t.Fatal("CustomerTopUpInquiryStatus() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("CustomerTopUpInquiryStatus() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestCustomerTopUpInquiryStatus_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up-inquiry-status"
	tr := &snap.Transport{}
	resp, err := CustomerTopUpInquiryStatus(context.Background(), tr, hb, CustomerTopUpInquiryStatusRequest{})
	if err == nil {
		t.Fatalf("CustomerTopUpInquiryStatus() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("CustomerTopUpInquiryStatus() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestCustomerTopUpInquiryStatus_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1122334455"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/customer-top-up-inquiry-status"
	tr := &snap.Transport{}
	resp, err := CustomerTopUpInquiryStatus(context.Background(), tr, hb, CustomerTopUpInquiryStatusRequest{})
	if err == nil {
		t.Fatalf("CustomerTopUpInquiryStatus() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
