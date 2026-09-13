package transferdebit

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

func TestDirectDebitBIFASTEMandateRegistrationTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(DirectDebitBIFASTEMandateRegistrationRequest{}).NumField(); n != 10 {
		t.Errorf("DirectDebitBIFASTEMandateRegistrationRequest has %d fields, want 10", n)
	}
	if n := reflect.TypeOf(DirectDebitBIFASTEMandateRegistrationResponse{}).NumField(); n != 6 {
		t.Errorf("DirectDebitBIFASTEMandateRegistrationResponse has %d fields, want 6", n)
	}
}

func TestDirectDebitBIFASTEMandateRegistration_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2007000",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"REF001",
   "partnerReferenceNo":"PARTNER001",
   "eMandateReffId":"EMANDATE001",
   "additionalInfo":{"note":"resp-note"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-emandate"
	tr := &snap.Transport{}
	resp, err := DirectDebitBIFASTEMandateRegistration(context.Background(), tr, hb, DirectDebitBIFASTEMandateRegistrationRequest{
		BankCode:          "014",
		SourceAccountNo:   "1234567890123",
		SourceAccountName: "Jane Doe",
		BillerID:          "BILLER01",
		BillerName:        "Biller Name",
		CustomerID:        "CUSTOMER01",
		ExpiredDatetime:   "2026-12-31T23:59:59+07:00",
	})
	if err != nil {
		t.Fatalf("DirectDebitBIFASTEMandateRegistration() error = %v", err)
	}

	want := DirectDebitBIFASTEMandateRegistrationResponse{
		ResponseCode:       "2007000",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "REF001",
		PartnerReferenceNo: "PARTNER001",
		EMandateReffID:     "EMANDATE001",
		AdditionalInfo:     json.RawMessage(`{"note":"resp-note"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("DirectDebitBIFASTEMandateRegistration() = %+v, want %+v", resp, want)
	}
}

func TestDirectDebitBIFASTEMandateRegistration_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2007000","responseMessage":"ok","eMandateReffId":"EMANDATE001"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-emandate"
	tr := &snap.Transport{}
	req := DirectDebitBIFASTEMandateRegistrationRequest{
		PartnerReferenceNo: "PARTNER001",
		BankCode:           "014",
		SourceAccountNo:    "1234567890123",
		SourceAccountName:  "Jane Doe",
		MaxAmount:          &snap.Money{Value: "5000000.00", Currency: "IDR"},
		BillerID:           "BILLER01",
		BillerName:         "Biller Name",
		CustomerID:         "CUSTOMER01",
		ExpiredDatetime:    "2026-12-31T23:59:59+07:00",
		AdditionalInfo:     json.RawMessage(`{"note":"req-value"}`),
	}
	if _, err := DirectDebitBIFASTEMandateRegistration(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("DirectDebitBIFASTEMandateRegistration() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"partnerReferenceNo": "PARTNER001",
		"bankCode":           "014",
		"sourceAccountNo":    "1234567890123",
		"sourceAccountName":  "Jane Doe",
		"maxAmount":          map[string]any{"value": "5000000.00", "currency": "IDR"},
		"billerId":           "BILLER01",
		"billerName":         "Biller Name",
		"customerId":         "CUSTOMER01",
		"expiredDatetime":    "2026-12-31T23:59:59+07:00",
		"additionalInfo":     map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestDirectDebitBIFASTEMandateRegistration_MandatoryFieldsAlwaysSerialized
// pins that BankCode, SourceAccountNo, SourceAccountName, BillerID,
// BillerName, CustomerID, and ExpiredDatetime — the request fields
// without omitempty — always serialize, even as "", and every other
// field is omitted when unset.
func TestDirectDebitBIFASTEMandateRegistration_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2007000","responseMessage":"ok","eMandateReffId":"EMANDATE001"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-emandate"
	tr := &snap.Transport{}
	if _, err := DirectDebitBIFASTEMandateRegistration(context.Background(), tr, hb, DirectDebitBIFASTEMandateRegistrationRequest{}); err != nil {
		t.Fatalf("DirectDebitBIFASTEMandateRegistration() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	want := map[string]any{
		"bankCode":          "",
		"sourceAccountNo":   "",
		"sourceAccountName": "",
		"billerId":          "",
		"billerName":        "",
		"customerId":        "",
		"expiredDatetime":   "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestDirectDebitBIFASTEMandateRegistrationResponse_MandatoryFieldAlwaysSerialized
// pins that EMandateReffID — the only response field without
// omitempty beyond the envelope — always serializes from a zero-value
// response.
func TestDirectDebitBIFASTEMandateRegistrationResponse_MandatoryFieldAlwaysSerialized(t *testing.T) {
	b, err := json.Marshal(DirectDebitBIFASTEMandateRegistrationResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{"responseCode": "", "responseMessage": "", "eMandateReffId": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestDirectDebitBIFASTEMandateRegistration_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4007000","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-emandate"
	tr := &snap.Transport{}
	_, err := DirectDebitBIFASTEMandateRegistration(context.Background(), tr, hb, DirectDebitBIFASTEMandateRegistrationRequest{})
	if err == nil {
		t.Fatal("DirectDebitBIFASTEMandateRegistration() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("DirectDebitBIFASTEMandateRegistration() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestDirectDebitBIFASTEMandateRegistration_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2007000","responseMessage":"ok","eMandateReffId":"EMANDATE001"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-emandate"
	tr := &snap.Transport{}
	resp, err := DirectDebitBIFASTEMandateRegistration(context.Background(), tr, hb, DirectDebitBIFASTEMandateRegistrationRequest{})
	if err == nil {
		t.Fatalf("DirectDebitBIFASTEMandateRegistration() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("DirectDebitBIFASTEMandateRegistration() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestDirectDebitBIFASTEMandateRegistration_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"REF001"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/debit/fast-emandate"
	tr := &snap.Transport{}
	resp, err := DirectDebitBIFASTEMandateRegistration(context.Background(), tr, hb, DirectDebitBIFASTEMandateRegistrationRequest{})
	if err == nil {
		t.Fatalf("DirectDebitBIFASTEMandateRegistration() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
