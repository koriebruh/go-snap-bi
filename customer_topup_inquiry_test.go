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

func TestAccountInquiryCustomerTopUp_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003700",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"ref-1",
   "partnerReferenceNo":"partner-ref-1",
   "sessionId":"sess-1",
   "customerNumber":"XXXXXXXXX1857",
   "customerName":"Jane Doe",
   "customerMonthlyInLimit":"5000000.00",
   "minAmount":{"value":"10000.00","currency":"IDR"},
   "maxAmount":{"value":"1000000.00","currency":"IDR"},
   "amount":{"value":"100000.00","currency":"IDR"},
   "feeAmount":{"value":"1000.00","currency":"IDR"},
   "feeType":"01"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-customer-top-up"
	tr := &Transport{}
	resp, err := AccountInquiryCustomerTopUp(context.Background(), tr, hb, AccountInquiryCustomerTopUpRequest{
		Amount: Money{Value: "100000.00", Currency: "IDR"},
	})
	if err != nil {
		t.Fatalf("AccountInquiryCustomerTopUp() error = %v", err)
	}

	want := AccountInquiryCustomerTopUpResponse{
		ResponseCode:           "2003700",
		ResponseMessage:        "Request has been processed successfully",
		ReferenceNo:            "ref-1",
		PartnerReferenceNo:     "partner-ref-1",
		SessionID:              "sess-1",
		CustomerNumber:         "XXXXXXXXX1857",
		CustomerName:           "Jane Doe",
		CustomerMonthlyInLimit: json.RawMessage(`"5000000.00"`),
		MinAmount:              &Money{Value: "10000.00", Currency: "IDR"},
		MaxAmount:              &Money{Value: "1000000.00", Currency: "IDR"},
		Amount:                 &Money{Value: "100000.00", Currency: "IDR"},
		FeeAmount:              &Money{Value: "1000.00", Currency: "IDR"},
		FeeType:                "01",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AccountInquiryCustomerTopUp() = %+v, want %+v", resp, want)
	}
}

// TestAccountInquiryCustomerTopUp_CustomerMonthlyInLimitAcceptsEitherWireShape
// pins that CustomerMonthlyInLimit (json.RawMessage) round-trips both
// the quoted-string shape (this endpoint's own worked example) and a
// bare-number shape, per the package's ambiguous-type rule: a
// documented Numeric field gets json.RawMessage even when only one
// wire shape has been observed.
func TestAccountInquiryCustomerTopUp_CustomerMonthlyInLimitAcceptsEitherWireShape(t *testing.T) {
	tests := []struct {
		name  string
		limit string
	}{
		{name: "quoted string", limit: `"5000000.00"`},
		{name: "bare number", limit: `5000000`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := `{"responseCode":"2003700","responseMessage":"ok","customerName":"Jane Doe","customerMonthlyInLimit":` + tt.limit + `}`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(fixture))
			}))
			defer server.Close()

			hb := testHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/account-inquiry-customer-top-up"
			tr := &Transport{}
			resp, err := AccountInquiryCustomerTopUp(context.Background(), tr, hb, AccountInquiryCustomerTopUpRequest{})
			if err != nil {
				t.Fatalf("AccountInquiryCustomerTopUp() error = %v", err)
			}
			if string(resp.CustomerMonthlyInLimit) != tt.limit {
				t.Errorf("CustomerMonthlyInLimit = %s, want %s", resp.CustomerMonthlyInLimit, tt.limit)
			}
		})
	}
}

func TestAccountInquiryCustomerTopUp_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003700","responseMessage":"ok","customerName":"Jane Doe"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-customer-top-up"
	tr := &Transport{}
	req := AccountInquiryCustomerTopUpRequest{
		PartnerReferenceNo: "partner-ref-1",
		CustomerNumber:     "12345678901234567890",
		Amount:             Money{Value: "100000.00", Currency: "IDR"},
	}
	if _, err := AccountInquiryCustomerTopUp(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AccountInquiryCustomerTopUp() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	amount, ok := got["amount"].(map[string]any)
	if !ok {
		t.Fatalf(`wire body["amount"] = %v, want an object`, got["amount"])
	}
	if amount["value"] != "100000.00" {
		t.Errorf(`wire body["amount"]["value"] = %v, want "100000.00"`, amount["value"])
	}
}

// TestAccountInquiryCustomerTopUp_MandatoryFieldAlwaysSerialized pins
// that Amount — the only request field without omitempty — is always
// present on the wire as an object (mandatory nested Money is a plain
// struct, so omitempty never omits it).
func TestAccountInquiryCustomerTopUp_MandatoryFieldAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003700","responseMessage":"ok","customerName":"Jane Doe"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-customer-top-up"
	tr := &Transport{}
	if _, err := AccountInquiryCustomerTopUp(context.Background(), tr, hb, AccountInquiryCustomerTopUpRequest{}); err != nil {
		t.Fatalf("AccountInquiryCustomerTopUp() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if _, ok := got["amount"].(map[string]any); !ok {
		t.Errorf(`wire body["amount"] = %v, want an object (mandatory nested Money is a plain struct)`, got["amount"])
	}
	if _, ok := got["customerNumber"]; ok {
		t.Error(`wire body has "customerNumber" key, want it omitted (Optional here)`)
	}
}

func TestAccountInquiryCustomerTopUp_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003700","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-customer-top-up"
	tr := &Transport{}
	_, err := AccountInquiryCustomerTopUp(context.Background(), tr, hb, AccountInquiryCustomerTopUpRequest{})
	if err == nil {
		t.Fatal("AccountInquiryCustomerTopUp() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("AccountInquiryCustomerTopUp() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestAccountInquiryCustomerTopUp_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-customer-top-up"
	tr := &Transport{}
	resp, err := AccountInquiryCustomerTopUp(context.Background(), tr, hb, AccountInquiryCustomerTopUpRequest{})
	if err == nil {
		t.Fatalf("AccountInquiryCustomerTopUp() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("AccountInquiryCustomerTopUp() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestAccountInquiryCustomerTopUp_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"customerName":"Jane Doe"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-customer-top-up"
	tr := &Transport{}
	resp, err := AccountInquiryCustomerTopUp(context.Background(), tr, hb, AccountInquiryCustomerTopUpRequest{})
	if err == nil {
		t.Fatalf("AccountInquiryCustomerTopUp() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
