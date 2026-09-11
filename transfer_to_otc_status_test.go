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

func TestTransferToOTCTransferStatus_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004500",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "originalExternalId":"ext-1",
   "serviceCode":"44",
   "transactionDate":"2020-12-20T10:00:00+07:00",
   "amount":{"value":"100000.00","currency":"IDR"},
   "beneficiaryAccountNo":"1122334455",
   "beneficiaryBankCode":"014",
   "previousResponseCode":"2004400",
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

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-status"
	tr := &Transport{}
	resp, err := TransferToOTCTransferStatus(context.Background(), tr, hb, TransferToOTCTransferStatusRequest{
		ServiceCode:    "44",
		CustomerNumber: "98765",
		Amount:         Money{Value: "100000.00", Currency: "IDR"},
	})
	if err != nil {
		t.Fatalf("TransferToOTCTransferStatus() error = %v", err)
	}

	want := TransferToOTCTransferStatusResponse{
		ResponseCode:               "2004500",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		ServiceCode:                "44",
		TransactionDate:            "2020-12-20T10:00:00+07:00",
		Amount:                     &Money{Value: "100000.00", Currency: "IDR"},
		BeneficiaryAccountNo:       "1122334455",
		BeneficiaryBankCode:        "014",
		PreviousResponseCode:       "2004400",
		ReferenceNumber:            "refnum-1",
		SourceAccountNo:            "9988776655",
		TransactionID:              "TX000001",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("TransferToOTCTransferStatus() = %+v, want %+v", resp, want)
	}
}

func TestTransferToOTCTransferStatus_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-status"
	tr := &Transport{}
	req := TransferToOTCTransferStatusRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		ServiceCode:                "44",
		CustomerNumber:             "98765",
		Amount:                     Money{Value: "100000.00", Currency: "IDR"},
	}
	if _, err := TransferToOTCTransferStatus(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("TransferToOTCTransferStatus() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["customerNumber"] != "98765" {
		t.Errorf(`wire body["customerNumber"] = %v, want "98765"`, got["customerNumber"])
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok || amount["value"] != "100000.00" {
		t.Errorf(`wire body["amount"] = %v, want {"value":"100000.00","currency":"IDR"}`, got["amount"])
	}
}

// TestTransferToOTCTransferStatus_MandatoryFieldsAlwaysSerialized
// pins that ServiceCode, CustomerNumber, and Amount — the request
// fields without omitempty — are always present on the wire. Amount
// is a mandatory nested object (plain Money, not *Money), so it
// always serializes as an object.
func TestTransferToOTCTransferStatus_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-status"
	tr := &Transport{}
	if _, err := TransferToOTCTransferStatus(context.Background(), tr, hb, TransferToOTCTransferStatusRequest{}); err != nil {
		t.Fatalf("TransferToOTCTransferStatus() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"serviceCode", "customerNumber"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	if _, ok := got["amount"].(map[string]any); !ok {
		t.Errorf(`wire body["amount"] = %v, want an object (mandatory nested Money is a plain struct)`, got["amount"])
	}
}

// TestTransferToOTCTransferStatus_IdenticalToTransactionStatusInquiryBank
// pins that TransferToOTCTransferStatusResponse (Phase 20) is
// field-identical to TransactionStatusInquiryBankResponse (Phase 16),
// per research §5.8's "originalX/serviceCode/status pattern" (with the
// additions scoped to "on request" only, per the design doc). These
// two types have no compile-time link to each other, so nothing else
// in the package would catch one of them drifting from the other.
func TestTransferToOTCTransferStatus_IdenticalToTransactionStatusInquiryBank(t *testing.T) {
	shape := func(v any) []string {
		typ := reflect.TypeOf(v)
		fields := make([]string, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			fields[i] = f.Name + " " + f.Type.String() + " `" + string(f.Tag) + "`"
		}
		return fields
	}

	if got, want := shape(TransferToOTCTransferStatusResponse{}), shape(TransactionStatusInquiryBankResponse{}); !reflect.DeepEqual(got, want) {
		t.Errorf("TransferToOTCTransferStatusResponse field shape = %v, want (matching TransactionStatusInquiryBankResponse) %v", got, want)
	}
}

func TestTransferToOTCTransferStatus_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004500","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-status"
	tr := &Transport{}
	_, err := TransferToOTCTransferStatus(context.Background(), tr, hb, TransferToOTCTransferStatusRequest{})
	if err == nil {
		t.Fatal("TransferToOTCTransferStatus() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("TransferToOTCTransferStatus() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestTransferToOTCTransferStatus_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-status"
	tr := &Transport{}
	resp, err := TransferToOTCTransferStatus(context.Background(), tr, hb, TransferToOTCTransferStatusRequest{})
	if err == nil {
		t.Fatalf("TransferToOTCTransferStatus() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("TransferToOTCTransferStatus() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestTransferToOTCTransferStatus_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1122334455"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-status"
	tr := &Transport{}
	resp, err := TransferToOTCTransferStatus(context.Background(), tr, hb, TransferToOTCTransferStatusRequest{})
	if err == nil {
		t.Fatalf("TransferToOTCTransferStatus() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
