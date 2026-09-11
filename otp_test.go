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

func TestOTP_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2008100",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "chargeToken":"abcd63617264746f6b656e",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp"
	tr := &Transport{}
	resp, err := OTP(context.Background(), tr, hb, OTPRequest{JourneyID: "20190329175623MTISTORE"})
	if err != nil {
		t.Fatalf("OTP() error = %v", err)
	}

	want := OTPResponse{
		ResponseCode:       "2008100",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		ChargeToken:        "abcd63617264746f6b656e",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("OTP() = %+v, want %+v", resp, want)
	}
}

func TestOTP_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2008100","responseMessage":"ok","chargeToken":"tok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp"
	tr := &Transport{}
	req := OTPRequest{
		PartnerReferenceNo: "2020102900000000000001",
		JourneyID:          "20190329175623MTISTORE",
		MerchantID:         "00001",
		SubMerchant:        "310928924949487",
		ExternalStoreID:    "124928924949487",
		TrxDateTime:        "2020-12-21T14:56:11+07:00",
		BankCardToken:      "6d7963617264746f6b656e",
		OTPTrxCode:         "54",
		OTPReasonCode:      "01",
		OTPReasonMessage:   "invalid otp",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := OTP(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("OTP() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["journeyId"] != "20190329175623MTISTORE" {
		t.Errorf(`wire body["journeyId"] = %v, want "20190329175623MTISTORE"`, got["journeyId"])
	}
	if got["subMerchant"] != "310928924949487" {
		t.Errorf(`wire body["subMerchant"] = %v, want "310928924949487"`, got["subMerchant"])
	}
	if _, ok := got["subMerchantId"]; ok {
		t.Error(`wire body has "subMerchantId" key, want only "subMerchant" (this endpoint's own field name)`)
	}
	if got["otpReasonCode"] != "01" {
		t.Errorf(`wire body["otpReasonCode"] = %v, want "01"`, got["otpReasonCode"])
	}
	if got["otpReasonMessage"] != "invalid otp" {
		t.Errorf(`wire body["otpReasonMessage"] = %v, want "invalid otp"`, got["otpReasonMessage"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestOTP_JourneyIDAlwaysSerialized pins that JourneyID — the one
// request field without omitempty — is always present on the wire, even
// as "", mirroring TestAccountBinding_MerchantIDAlwaysSerialized.
func TestOTP_JourneyIDAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2008100","responseMessage":"ok","chargeToken":"tok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp"
	tr := &Transport{}
	if _, err := OTP(context.Background(), tr, hb, OTPRequest{}); err != nil {
		t.Fatalf("OTP() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	journeyID, ok := got["journeyId"]
	if !ok {
		t.Fatal(`wire body missing "journeyId" key; JourneyID lacks omitempty and must always be present, even as ""`)
	}
	if journeyID != "" {
		t.Errorf(`wire body["journeyId"] = %v, want ""`, journeyID)
	}
}

func TestOTP_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4008100","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp"
	tr := &Transport{}
	_, err := OTP(context.Background(), tr, hb, OTPRequest{JourneyID: "j-1"})
	if err == nil {
		t.Fatal("OTP() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("OTP() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestOTP_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2008100","responseMessage":"ok","chargeToken":"tok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp"
	tr := &Transport{}
	resp, err := OTP(context.Background(), tr, hb, OTPRequest{JourneyID: "j-1"})
	if err == nil {
		t.Fatalf("OTP() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("OTP() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestOTP_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"chargeToken":"tok"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp"
	tr := &Transport{}
	resp, err := OTP(context.Background(), tr, hb, OTPRequest{JourneyID: "j-1"})
	if err == nil {
		t.Fatalf("OTP() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
