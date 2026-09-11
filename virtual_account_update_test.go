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

func TestUpdateVA_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2002800",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountData":{
      "partnerServiceId":"12345",
      "customerNo":"98765",
      "virtualAccountNo":"1234598765",
      "virtualAccountName":"Jane Doe",
      "trxId":"trx-1",
      "lastUpdateDate":"2020-12-21T14:56:11+07:00",
      "paymentDate":"2020-12-22T09:00:00+07:00",
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-va"
	tr := &Transport{}
	resp, err := UpdateVA(context.Background(), tr, hb, UpdateVARequest{
		PartnerServiceID:   "12345",
		CustomerNo:         "98765",
		VirtualAccountNo:   "1234598765",
		VirtualAccountName: "Jane Doe",
		TrxID:              "trx-1",
	})
	if err != nil {
		t.Fatalf("UpdateVA() error = %v", err)
	}

	want := UpdateVAResponse{
		ResponseCode:    "2002800",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &UpdateVAData{
			PartnerServiceID:   "12345",
			CustomerNo:         "98765",
			VirtualAccountNo:   "1234598765",
			VirtualAccountName: "Jane Doe",
			TrxID:              "trx-1",
			LastUpdateDate:     "2020-12-21T14:56:11+07:00",
			PaymentDate:        "2020-12-22T09:00:00+07:00",
			AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("UpdateVA() = %+v, want %+v", resp, want)
	}
}

// TestUpdateVA_UsesPUTMethod pins that UpdateVA sets hb.Method to PUT
// regardless of what the caller configured, since PUT is fixed by this
// endpoint rather than caller-configurable.
func TestUpdateVA_UsesPUTMethod(t *testing.T) {
	var mu sync.Mutex
	var gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2002800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL) // testHeaderBuilder sets Method to POST
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-va"
	tr := &Transport{}
	if _, err := UpdateVA(context.Background(), tr, hb, UpdateVARequest{}); err != nil {
		t.Fatalf("UpdateVA() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotMethod != http.MethodPut {
		t.Errorf("request method = %q, want %q", gotMethod, http.MethodPut)
	}
}

func TestUpdateVA_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-va"
	tr := &Transport{}
	req := UpdateVARequest{
		PartnerServiceID:   "12345",
		CustomerNo:         "98765",
		VirtualAccountNo:   "1234598765",
		VirtualAccountName: "Jane Doe",
		TrxID:              "trx-1",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := UpdateVA(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("UpdateVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerServiceId"] != "12345" || got["customerNo"] != "98765" || got["virtualAccountNo"] != "1234598765" {
		t.Errorf("wire body identity triple = %v, want the test's values", got)
	}
	if got["virtualAccountName"] != "Jane Doe" || got["trxId"] != "trx-1" {
		t.Errorf("wire body[virtualAccountName/trxId] = %v/%v, want \"Jane Doe\"/\"trx-1\"", got["virtualAccountName"], got["trxId"])
	}
}

// TestUpdateVA_MandatoryFieldsAlwaysSerialized pins that
// PartnerServiceID, CustomerNo, VirtualAccountNo, VirtualAccountName,
// and TrxID — the five request fields without omitempty — are always
// present on the wire, even as "".
func TestUpdateVA_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-va"
	tr := &Transport{}
	if _, err := UpdateVA(context.Background(), tr, hb, UpdateVARequest{}); err != nil {
		t.Fatalf("UpdateVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo", "virtualAccountName", "trxId"} {
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

func TestUpdateVA_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4002800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-va"
	tr := &Transport{}
	_, err := UpdateVA(context.Background(), tr, hb, UpdateVARequest{})
	if err == nil {
		t.Fatal("UpdateVA() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("UpdateVA() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestUpdateVA_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2002800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-va"
	tr := &Transport{}
	resp, err := UpdateVA(context.Background(), tr, hb, UpdateVARequest{})
	if err == nil {
		t.Fatalf("UpdateVA() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("UpdateVA() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestUpdateVA_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountData":{"trxId":"trx-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-va"
	tr := &Transport{}
	resp, err := UpdateVA(context.Background(), tr, hb, UpdateVARequest{})
	if err == nil {
		t.Fatalf("UpdateVA() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
