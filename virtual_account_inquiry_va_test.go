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

func TestInquiryVA_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003000",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountData":{
      "partnerServiceId":"12345",
      "customerNo":"98765",
      "virtualAccountNo":"1234598765",
      "virtualAccountName":"Jane Doe",
      "trxId":"trx-1",
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &Transport{}
	resp, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{
		PartnerServiceID: "12345",
		CustomerNo:       "98765",
		VirtualAccountNo: "1234598765",
		TrxID:            "trx-1",
	})
	if err != nil {
		t.Fatalf("InquiryVA() error = %v", err)
	}

	want := InquiryVAResponse{
		ResponseCode:    "2003000",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &InquiryVAData{
			PartnerServiceID:   "12345",
			CustomerNo:         "98765",
			VirtualAccountNo:   "1234598765",
			VirtualAccountName: "Jane Doe",
			TrxID:              "trx-1",
			AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("InquiryVA() = %+v, want %+v", resp, want)
	}
}

func TestInquiryVA_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &Transport{}
	req := InquiryVARequest{
		PartnerServiceID: "12345",
		CustomerNo:       "98765",
		VirtualAccountNo: "1234598765",
		TrxID:            "trx-1",
		AdditionalInfo:   json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := InquiryVA(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("InquiryVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerServiceId"] != "12345" || got["customerNo"] != "98765" || got["virtualAccountNo"] != "1234598765" || got["trxId"] != "trx-1" {
		t.Errorf("wire body = %v, want the test's identity triple + trxId", got)
	}
}

// TestInquiryVA_MandatoryFieldsAlwaysSerialized pins that
// PartnerServiceID, CustomerNo, VirtualAccountNo, and TrxID — the four
// request fields without omitempty — are always present on the wire,
// even as "".
func TestInquiryVA_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &Transport{}
	if _, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{}); err != nil {
		t.Fatalf("InquiryVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo", "trxId"} {
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

func TestInquiryVA_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003000","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &Transport{}
	_, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{})
	if err == nil {
		t.Fatal("InquiryVA() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("InquiryVA() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestInquiryVA_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &Transport{}
	resp, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{})
	if err == nil {
		t.Fatalf("InquiryVA() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("InquiryVA() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestInquiryVA_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountData":{"trxId":"trx-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &Transport{}
	resp, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{})
	if err == nil {
		t.Fatalf("InquiryVA() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
