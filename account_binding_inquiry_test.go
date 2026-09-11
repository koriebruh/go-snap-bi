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

func TestAccountBindingInquiry_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000800",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "accountCurrency":"IDR",
   "accountName":"Alen Miucic",
   "accountNo":"11231271284140",
   "accountTransactionLimit":"1000000",
   "endDatePeriod":"2022-05-21",
   "startDatePeriod":"2020-05-21",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &Transport{}
	resp, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
	if err != nil {
		t.Fatalf("AccountBindingInquiry() error = %v", err)
	}

	want := AccountBindingInquiryResponse{
		ResponseCode:            "2000800",
		ResponseMessage:         "Request has been processed successfully",
		ReferenceNo:             "2020102977770000000009",
		PartnerReferenceNo:      "2020102900000000000001",
		AccountCurrency:         "IDR",
		AccountName:             "Alen Miucic",
		AccountNo:               "11231271284140",
		AccountTransactionLimit: "1000000",
		EndDatePeriod:           "2022-05-21",
		StartDatePeriod:         "2020-05-21",
		AdditionalInfo:          json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AccountBindingInquiry() = %+v, want %+v", resp, want)
	}
}

// TestAccountBindingInquiry_UnquotedTransactionLimitFailsDecodeCleanly
// documents a known risk of typing AccountTransactionLimit as string: the
// Guides tab labels it Numeric, and this package trusts a single portal
// worked example that renders it quoted. If some issuer instead sends an
// unquoted JSON number, decode fails for the whole response (not just this
// field) — this test pins that the failure is a clean wrapped decode error,
// not silent corruption or a partially-populated response being returned
// as if it were valid.
func TestAccountBindingInquiry_UnquotedTransactionLimitFailsDecodeCleanly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2000800","responseMessage":"ok","accountTransactionLimit":1000000}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &Transport{}
	resp, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
	if err == nil {
		t.Fatalf("AccountBindingInquiry() error = nil, want non-nil for an unquoted numeric accountTransactionLimit; got %+v", resp)
	}
	if !reflect.DeepEqual(resp, AccountBindingInquiryResponse{}) {
		t.Errorf("AccountBindingInquiry() response = %+v, want zero value on decode failure", resp)
	}
}

func TestAccountBindingInquiry_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &Transport{}
	req := AccountBindingInquiryRequest{
		PartnerReferenceNo: "ref-1",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := AccountBindingInquiry(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AccountBindingInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerReferenceNo"] != "ref-1" {
		t.Errorf(`wire body["partnerReferenceNo"] = %v, want "ref-1"`, got["partnerReferenceNo"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

func TestAccountBindingInquiry_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &Transport{}
	_, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
	if err == nil {
		t.Fatal("AccountBindingInquiry() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("AccountBindingInquiry() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestAccountBindingInquiry_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &Transport{}
	resp, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
	if err == nil {
		t.Fatalf("AccountBindingInquiry() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("AccountBindingInquiry() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestAccountBindingInquiry_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accountNo":"11231271284140"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &Transport{}
	resp, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
	if err == nil {
		t.Fatalf("AccountBindingInquiry() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
