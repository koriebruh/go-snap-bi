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

func TestAccountInquiryExternal_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2001600",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "beneficiaryAccountName":"John Doe",
   "beneficiaryAccountNo":"1234567890",
   "beneficiaryBankName":"Bank Example",
   "currency":"IDR",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-external"
	tr := &Transport{}
	resp, err := AccountInquiryExternal(context.Background(), tr, hb, AccountInquiryExternalRequest{
		BeneficiaryAccountNo: "1234567890",
		BeneficiaryBankCode:  "014",
	})
	if err != nil {
		t.Fatalf("AccountInquiryExternal() error = %v", err)
	}

	want := AccountInquiryExternalResponse{
		ResponseCode:           "2001600",
		ResponseMessage:        "Request has been processed successfully",
		ReferenceNo:            "2020102977770000000009",
		PartnerReferenceNo:     "2020102900000000000001",
		BeneficiaryAccountName: "John Doe",
		BeneficiaryAccountNo:   "1234567890",
		BeneficiaryBankName:    "Bank Example",
		Currency:               "IDR",
		AdditionalInfo:         json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AccountInquiryExternal() = %+v, want %+v", resp, want)
	}
}

func TestAccountInquiryExternal_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-external"
	tr := &Transport{}
	req := AccountInquiryExternalRequest{
		PartnerReferenceNo:   "2020102900000000000001",
		BeneficiaryAccountNo: "1234567890",
		BeneficiaryBankCode:  "014",
		AdditionalInfo:       json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := AccountInquiryExternal(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AccountInquiryExternal() error = %v", err)
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
	if got["beneficiaryBankCode"] != "014" {
		t.Errorf(`wire body["beneficiaryBankCode"] = %v, want "014"`, got["beneficiaryBankCode"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestAccountInquiryExternal_MandatoryFieldsAlwaysSerialized pins that
// BeneficiaryAccountNo and BeneficiaryBankCode — the two request fields
// without omitempty — are always present on the wire, even as "",
// mirroring TestCardRegistration_MandatoryFieldsAlwaysSerialized.
func TestAccountInquiryExternal_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-external"
	tr := &Transport{}
	if _, err := AccountInquiryExternal(context.Background(), tr, hb, AccountInquiryExternalRequest{}); err != nil {
		t.Fatalf("AccountInquiryExternal() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"beneficiaryAccountNo", "beneficiaryBankCode"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
}

func TestAccountInquiryExternal_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001600","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-external"
	tr := &Transport{}
	_, err := AccountInquiryExternal(context.Background(), tr, hb, AccountInquiryExternalRequest{BeneficiaryAccountNo: "1234567890", BeneficiaryBankCode: "014"})
	if err == nil {
		t.Fatal("AccountInquiryExternal() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("AccountInquiryExternal() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestAccountInquiryExternal_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2001600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-external"
	tr := &Transport{}
	resp, err := AccountInquiryExternal(context.Background(), tr, hb, AccountInquiryExternalRequest{BeneficiaryAccountNo: "1234567890", BeneficiaryBankCode: "014"})
	if err == nil {
		t.Fatalf("AccountInquiryExternal() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("AccountInquiryExternal() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestAccountInquiryExternal_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiaryAccountNo":"1234567890"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/account-inquiry-external"
	tr := &Transport{}
	resp, err := AccountInquiryExternal(context.Background(), tr, hb, AccountInquiryExternalRequest{BeneficiaryAccountNo: "1234567890", BeneficiaryBankCode: "014"})
	if err == nil {
		t.Fatalf("AccountInquiryExternal() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
