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
	"testing"

	snap "github.com/koriebruh/go-snap-bi"
	"github.com/koriebruh/go-snap-bi/internal/snaptest"
)

func TestTransactionStatusInquiryNonBank_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2005300",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "originalExternalId":"ext-1",
   "serviceCode":"53",
   "transactionDate":"2020-12-20T10:00:00+07:00",
   "amount":{"value":"25000.00","currency":"IDR"},
   "beneficiaryAccountNo":"1122334455",
   "beneficiaryBankCode":"014",
   "previousResponseCode":"2005200",
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
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-status"
	tr := &snap.Transport{}
	resp, err := TransactionStatusInquiryNonBank(context.Background(), tr, hb, TransactionStatusInquiryNonBankRequest{
		ServiceCode: "53",
	})
	if err != nil {
		t.Fatalf("TransactionStatusInquiryNonBank() error = %v", err)
	}

	want := TransactionStatusInquiryNonBankResponse{
		ResponseCode:               "2005300",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		ServiceCode:                "53",
		TransactionDate:            "2020-12-20T10:00:00+07:00",
		Amount:                     &snap.Money{Value: "25000.00", Currency: "IDR"},
		BeneficiaryAccountNo:       "1122334455",
		BeneficiaryBankCode:        "014",
		PreviousResponseCode:       "2005200",
		ReferenceNumber:            "refnum-1",
		SourceAccountNo:            "9988776655",
		TransactionID:              "TX000001",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("TransactionStatusInquiryNonBank() = %+v, want %+v", resp, want)
	}
}

func TestTransactionStatusInquiryNonBank_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005300","responseMessage":"ok","beneficiaryAccountNo":"","referenceNumber":"","sourceAccountNo":"","latestTransactionStatus":""}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-status"
	tr := &snap.Transport{}
	req := TransactionStatusInquiryNonBankRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		ServiceCode:                "53",
		TransactionDate:            "2020-12-20T10:00:00+07:00",
		Amount:                     &snap.Money{Value: "25000.00", Currency: "IDR"},
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
		OriginalResponseCode:       "4005300",
		OriginalResponseMessage:    "Bad Request",
		SessionID:                  "session-1",
		RequestID:                  "request-1",
	}
	if _, err := TransactionStatusInquiryNonBank(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("TransactionStatusInquiryNonBank() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"originalPartnerReferenceNo": "partner-ref-1",
		"originalReferenceNo":        "ref-1",
		"originalExternalId":         "ext-1",
		"serviceCode":                "53",
		"transactionDate":            "2020-12-20T10:00:00+07:00",
		"amount":                     map[string]any{"value": "25000.00", "currency": "IDR"},
		"additionalInfo":             map[string]any{"channel": "mobilephone"},
		"originalResponseCode":       "4005300",
		"originalResponseMessage":    "Bad Request",
		"sessionId":                  "session-1",
		"requestId":                  "request-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestTransactionStatusInquiryNonBank_MandatoryFieldAlwaysSerialized
// pins that ServiceCode — the request field without omitempty — is
// always present on the wire, even as "".
func TestTransactionStatusInquiryNonBank_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2005300","responseMessage":"ok","beneficiaryAccountNo":"","referenceNumber":"","sourceAccountNo":"","latestTransactionStatus":""}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-status"
	tr := &snap.Transport{}
	if _, err := TransactionStatusInquiryNonBank(context.Background(), tr, hb, TransactionStatusInquiryNonBankRequest{}); err != nil {
		t.Fatalf("TransactionStatusInquiryNonBank() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{"serviceCode": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestTransactionStatusInquiryNonBankTypes_FieldCounts guards against
// a field silently added to either type without updating the
// wire-assertion tests, which catch renamed/omitempty-flipped fields
// but not an addition, since a new field defaults to unset and
// unasserted.
func TestTransactionStatusInquiryNonBankTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(TransactionStatusInquiryNonBankRequest{}).NumField(); n != 11 {
		t.Errorf("TransactionStatusInquiryNonBankRequest has %d fields, want 11", n)
	}
	if n := reflect.TypeOf(TransactionStatusInquiryNonBankResponse{}).NumField(); n != 17 {
		t.Errorf("TransactionStatusInquiryNonBankResponse has %d fields, want 17", n)
	}
}

// TestTransactionStatusInquiryNonBank_IdenticalToTransactionStatusInquiryBank
// pins that TransactionStatusInquiryNonBankResponse (Phase 25) is
// field-identical to TransactionStatusInquiryBankResponse (Phase 16),
// per research §5.10's "same shape" wording (with the four additions
// scoped to the request only, per the Phase 25 design doc). These two
// types have no compile-time link to each other, so nothing else in
// the package would catch one of them drifting from the other.
func TestTransactionStatusInquiryNonBank_IdenticalToTransactionStatusInquiryBank(t *testing.T) {
	shape := func(v any) []string {
		typ := reflect.TypeOf(v)
		fields := make([]string, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			fields[i] = f.Name + " " + f.Type.String() + " `" + string(f.Tag) + "`"
		}
		return fields
	}

	if got, want := shape(TransactionStatusInquiryNonBankResponse{}), shape(TransactionStatusInquiryBankResponse{}); !reflect.DeepEqual(got, want) {
		t.Errorf("TransactionStatusInquiryNonBankResponse field shape = %v, want (matching TransactionStatusInquiryBankResponse) %v", got, want)
	}
}

func TestTransactionStatusInquiryNonBank_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4005300","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-status"
	tr := &snap.Transport{}
	_, err := TransactionStatusInquiryNonBank(context.Background(), tr, hb, TransactionStatusInquiryNonBankRequest{})
	if err == nil {
		t.Fatal("TransactionStatusInquiryNonBank() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("TransactionStatusInquiryNonBank() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestTransactionStatusInquiryNonBank_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2005300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-status"
	tr := &snap.Transport{}
	resp, err := TransactionStatusInquiryNonBank(context.Background(), tr, hb, TransactionStatusInquiryNonBankRequest{})
	if err == nil {
		t.Fatalf("TransactionStatusInquiryNonBank() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("TransactionStatusInquiryNonBank() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestTransactionStatusInquiryNonBank_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1122334455"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/qr-mpm-status"
	tr := &snap.Transport{}
	resp, err := TransactionStatusInquiryNonBank(context.Background(), tr, hb, TransactionStatusInquiryNonBankRequest{})
	if err == nil {
		t.Fatalf("TransactionStatusInquiryNonBank() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
