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

func TestAccountInquiryInternal_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2001500",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "beneficiaryAccountName":"John Doe",
   "beneficiaryAccountNo":"1234567890",
   "beneficiaryAccountStatus":"Active",
   "beneficiaryAccountType":"D",
   "currency":"IDR"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-internal"
	tr := &Transport{}
	resp, err := AccountInquiryInternal(context.Background(), tr, hb, AccountInquiryInternalRequest{BeneficiaryAccountNo: "1234567890"})
	if err != nil {
		t.Fatalf("AccountInquiryInternal() error = %v", err)
	}

	want := AccountInquiryInternalResponse{
		ResponseCode:             "2001500",
		ResponseMessage:          "Request has been processed successfully",
		ReferenceNo:              "2020102977770000000009",
		PartnerReferenceNo:       "2020102900000000000001",
		BeneficiaryAccountName:   "John Doe",
		BeneficiaryAccountNo:     "1234567890",
		BeneficiaryAccountStatus: "Active",
		BeneficiaryAccountType:   "D",
		Currency:                 "IDR",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AccountInquiryInternal() = %+v, want %+v", resp, want)
	}
}

func TestAccountInquiryInternal_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-internal"
	tr := &Transport{}
	req := AccountInquiryInternalRequest{
		PartnerReferenceNo:   "2020102900000000000001",
		BeneficiaryAccountNo: "1234567890",
		AdditionalInfo:       json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := AccountInquiryInternal(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AccountInquiryInternal() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerReferenceNo"] != "2020102900000000000001" {
		t.Errorf(`wire body["partnerReferenceNo"] = %v, want "2020102900000000000001"`, got["partnerReferenceNo"])
	}
	if got["beneficiaryAccountNo"] != "1234567890" {
		t.Errorf(`wire body["beneficiaryAccountNo"] = %v, want "1234567890"`, got["beneficiaryAccountNo"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestAccountInquiryInternal_BeneficiaryAccountNoAlwaysSerialized pins that
// BeneficiaryAccountNo — the one request field without omitempty — is
// always present on the wire, even as "", mirroring
// TestAccountBinding_MerchantIDAlwaysSerialized.
func TestAccountInquiryInternal_BeneficiaryAccountNoAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-internal"
	tr := &Transport{}
	if _, err := AccountInquiryInternal(context.Background(), tr, hb, AccountInquiryInternalRequest{}); err != nil {
		t.Fatalf("AccountInquiryInternal() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	beneficiaryAccountNo, ok := got["beneficiaryAccountNo"]
	if !ok {
		t.Fatal(`wire body missing "beneficiaryAccountNo" key; BeneficiaryAccountNo lacks omitempty and must always be present, even as ""`)
	}
	if beneficiaryAccountNo != "" {
		t.Errorf(`wire body["beneficiaryAccountNo"] = %v, want ""`, beneficiaryAccountNo)
	}
}

func TestAccountInquiryInternal_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001500","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-internal"
	tr := &Transport{}
	_, err := AccountInquiryInternal(context.Background(), tr, hb, AccountInquiryInternalRequest{BeneficiaryAccountNo: "1234567890"})
	if err == nil {
		t.Fatal("AccountInquiryInternal() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("AccountInquiryInternal() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestAccountInquiryInternal_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2001500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-internal"
	tr := &Transport{}
	resp, err := AccountInquiryInternal(context.Background(), tr, hb, AccountInquiryInternalRequest{BeneficiaryAccountNo: "1234567890"})
	if err == nil {
		t.Fatalf("AccountInquiryInternal() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("AccountInquiryInternal() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestAccountInquiryInternal_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1234567890"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-internal"
	tr := &Transport{}
	resp, err := AccountInquiryInternal(context.Background(), tr, hb, AccountInquiryInternalRequest{BeneficiaryAccountNo: "1234567890"})
	if err == nil {
		t.Fatalf("AccountInquiryInternal() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
